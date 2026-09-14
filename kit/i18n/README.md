# kit/i18n

CLDR-exact internationalization for Go on Cloudflare Workers (TinyGo) and standard Go, with the options and output of
JavaScript's `Intl`. The package overview is in [doc.go](doc.go); app rules are in [AGENTS.md](../../AGENTS.md)
("kit/i18n").

## Reference data

kit/i18n doesn't hand-write locale data. It generates tables from Unicode CLDR and checks every output against what
`Intl` really prints in Chrome and Cloudflare Workers.

**Day-to-day work needs no network.** The generated tables and the recorded `Intl` output are committed, so
`mise run check` only compares against files in the repo. Upstream data is fetched only when you regenerate or verify.

### Sources and pins

| Source | Pinned version | Pinned in | Used for |
|---|---|---|---|
| [Unicode CLDR, cldr-json](https://github.com/unicode-org/cldr-json) | `48.2.1` | `DefaultTag` in [cldrgen/sources.go](cldrgen/sources.go) | the data tables in `cldr/` |
| [Unicode CLDR, XML](https://github.com/unicode-org/cldr) | `release-48-2` | `DefaultCLDRTag`, same file | what the JSON lacks: root and number-system symbols, draft status, pattern order |
| [Chromium's ICU fork](https://chromium.googlesource.com/chromium/deps/icu) (ICU 78.2) | commit `8cc91d9b…` | `ChromiumICU`, same file | Chrome's data filter (time zone names Chrome drops), region-name check, evidence for known differences |
| [workerd](https://github.com/cloudflare/workerd), Chrome's V8 + ICU | `1.20260911.1` | `workerd` in [mise.toml](../../mise.toml) and `Workerd` in sources.go | recording `Intl` output ("goldens") |
| IANA tzdata | `2026c` | the Go toolchain (`go` in mise.toml) | time zone offsets: `time/tzdata` in the Worker, `$GOROOT/lib/time/zoneinfo.zip` in tests |

- **Why the Chromium commit:** it is the one the pinned workerd release builds with. See
  `com_googlesource_chromium_icu` in workerd's `build/deps/gen/deps.MODULE.bazel` at tag `v1.20260911.1`; the
  commit's `README.chromium` says ICU 78.2.
- **CLDR versions:** CLDR 48.2 is newer than the CLDR 48.0 data inside ICU 78.2. That gap causes the few accepted
  differences in `known`.
- **Offsets:** tzdata is authoritative for offsets, not `Intl`. Browsers and workerd ship older tzdata.
- **Pin tests:** `TestReferencePins` (root `i18n_test.go`, part of `mise run check`) fails when these pins, the golden
  file's runtime, `I18N_LOCALES`, the tables and `intltest.Locales` disagree.

### What is committed and what is cached

Committed (reviewed like code, never hand-edited):

| Path | What | Regenerate with |
|---|---|---|
| `cldr/*_cldr.go` | generated tables for the shipped locales (`I18N_LOCALES` in mise.toml) | `mise run i18n:generate` |
| `testdata/golden/workerd.json` | `Intl` output for every conformance case | `mise run i18n:golden` |
| `intltest/cases*.go` | the conformance cases | hand-written |
| `known` + `knownEvidence` in `conformance_test.go` | accepted differences from `Intl`, each with checkable upstream evidence | hand-written |
| `locales/*.toml` → `locales/*_msg_gen.go` (app) | UI messages; `en.toml` is the source | `mise run i18n:messages` |

Upstream files are cached per machine, one directory per source and version. Delete the cache any time; it refills
on the next run. Override the location with `I18N_CACHE` (e.g. a CI cache).

```
<user cache dir>/go-htmx4/          macOS ~/Library/Caches/go-htmx4, Linux ~/.cache/go-htmx4
  cldr-json/48.2.1/…                cldr-json files, fetched file by file
  cldr-json/cldr-release-48-2/…     CLDR XML files
  chromium-icu/<commit>/…           Chromium ICU filter, locale and region files
```

### Tasks

| Task | When | Network |
|---|---|---|
| `mise run check` | always | no |
| `mise run i18n:generate` | after changing `I18N_LOCALES`, a CLDR or Chromium pin, or the generator | first run |
| `mise run i18n:golden` | after adding or changing conformance cases, or moving the workerd pin | no (local workerd) |
| `mise run i18n:messages` | after editing `locales/*.toml` (also runs inside `mise run generate`) | no |
| `mise run i18n:verify` | before committing a pin, generator or case change; when reviewing a `known` entry | first run |
| `mise run i18n:browsers` | when changing browser-side formatting (`static/relative-time.js`) or after browser updates | no (local Chrome/Firefox) |

`i18n:verify` takes about 1.5 minutes and downloads 17 MB from an empty cache, and seconds after that. It proves
that the committed data is what the pins produce:
- the tables regenerate byte for byte (`TestTablesReproduce`);
- region names match Chromium's ICU data (`TestChromiumRegionNames`);
- a fresh recording with the pinned workerd equals the golden file (`TestGoldenReproduces`);
- every `known` entry's evidence still holds in the pinned files (`TestKnownEvidence`).

### Browser drift

kit/i18n matches the pinned workerd, and the browser only re-formats text where it knows better (the viewer's clock
and zone). `mise run i18n:browsers` runs every conformance case in headless Chrome and Firefox using the oracle's
`intl.mjs`, then prints differences from the golden by area. JSON reports go to `build/i18n-browsers/`; use
`$CHROME`, `$FIREFOX` and `I18N_BROWSERS=chrome` to choose. It is a report, not a gate.

On 2026-09-14:
- Chrome 152 differed in 1 case, a host-default locale.
- Firefox 155 differed in 887: dates, numbers, durations and units, but not relative time, lists or plurals.

Check it before browser code formats anything new.

### Rules

- Tables and goldens change only through the tasks. Commit them in the same commit as the change that caused them
  (pin, generator, cases, locales).
- A difference from `Intl` is fixed in kit/i18n. It is accepted only as a `known` entry (case ID prefix + reason) with
  `knownEvidence`: text the pinned Chromium ICU or cldr-json has or lacks. Unimplemented features, such as
  non-Gregorian calendars, are the only entries without evidence.
- Chrome-specific data rules come from the pinned Chromium files, never from memory. If a rule has to live in code (like
  `chromeRegionAlt`), a verify test must check it against those files.
- Option combinations the app uses need conformance cases (V8 has surprises, e.g. ja `dateStyle: "full"` +
  `hourCycle: "h12"` prints "2026/5/10日曜日").

### Moving a pin

**workerd (newer Chrome V8/ICU)**
1. Set the new release in `mise.toml` (`workerd`), `Workerd` in `cldrgen/sources.go`, and `compatibilityDate` in
   `intltest/oracle.go`. Run `mise install`.
2. Read that tag's `build/deps/gen/deps.MODULE.bazel` in the workerd repo and copy the `com_googlesource_chromium_icu`
   commit into `ChromiumICU`. Check the ICU version in `README.chromium` at that commit.
3. If that ICU uses a newer CLDR major, move the CLDR pins too (below).
4. `mise run i18n:golden`, then `go test ./kit/i18n`. For each new mismatch, fix kit/i18n, or add a `known` entry
   with evidence.
5. `mise run i18n:generate`. Chromium's filter may have changed, which changes zone names.
6. `mise run i18n:verify`. Drop `known` entries whose evidence no longer holds; the difference is gone. Then
   `mise run check` and commit.

**CLDR**
1. Set `DefaultTag` (cldr-json) and `DefaultCLDRTag` (unicode-org/cldr) to the same release.
2. `mise run i18n:generate`, then `go test ./kit/i18n`. CLDR newer than Chrome's ICU data shows up as mismatches:
   decide per case (evidence-backed `known` entries).
3. `mise run i18n:verify`, `mise run check`, commit.

**tzdata**
It comes with Go: move `go` in mise.toml, and check `DATA=` in `$(go env GOROOT)/lib/time/update.bash`. Zone-offset
expectations come from tzdata, not the goldens.

**Adding a locale**
1. Add it to `I18N_LOCALES` (mise.toml) and to `intltest.Locales`.
2. `mise run i18n:generate`, then `mise run i18n:golden`.
3. Add `locales/<id>.toml`, then `mise run i18n:messages`.
4. `mise run i18n:verify`, then `mise run check`.
