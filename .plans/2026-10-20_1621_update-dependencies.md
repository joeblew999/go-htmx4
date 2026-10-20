# 2026-10-20_1621_update-dependencies.md

Bring every pinned tool and Go module up to its latest release, in one change, proven locally with the same commands
CI runs (`ci/check.sh`: `mise run check` then `mise run e2e`).

## Versions

| What | From | To | Where it's pinned |
| --- | --- | --- | --- |
| Go | 1.27.1 | 1.27.2 | `go.mod` + `e2e/go.mod` `toolchain` (mise reads it), README, AGENTS.md |
| TinyGo | 0.42.0 | 0.43.0 | `mise.toml`, smoke check in `tasks/app.toml`, README, AGENTS.md |
| binaryen (wasm-opt) | version_132 | version_133 | `mise.toml` (TinyGo 0.43 builds against 133) |
| workerd | 1.20260911.1 | 1.20261019.1 | via `cmd/i18npins -write` (`mise.toml`, `cldrgen/sources.go`, ICU commit), README, kit/i18n/README |
| fnox | 1.35.0 | 1.40.0 | `mise.toml`, README |
| jq | 1.8.1 | 1.8.2 | `mise.toml` |
| usage | 6.8.0 | 7.0.2 | `mise.toml` |
| gsxui CLI | c7fd6a8 | 915241b | `mise.toml`, `tasks/upstream.toml`, README |
| mise (CI) | v2026.9.5 | v2026.10.14 | `ci/check.sh`, `ci/i18n-verify.sh`, `min_version` in `mise.toml` |
| chrome-headless-shell (e2e) | — (runner's Chrome) | 155.0.8059.84 | `mise.toml` (`http:` backend, linux-x64 + macos-arm64/x64); `mise run e2e` passes it as `E2E_CHROME` |
| gsx | v0.1.0 | v0.4.0 | `go.mod`, README, AGENTS.md |
| workers-go | v0.35.0 | v0.36.0 | `go.mod`, `tasks/upstream.toml`, `main.go`, `tasks/app.toml`, README, AGENTS.md |
| golang.org/x/net | v0.59.0 | v0.61.0 | `go.mod` |
| chromedp (e2e) | v0.16.0 | — (held) | v0.17–v0.21 rewrote the API (decision 6) |
| Tailwind | 4.3.3 | — | already latest |
| htmx | 4.0.0 | — | already latest stable (only v4.0.0 tagged; betas are older) |

## Decisions

1. **One change, all of it.** Each bump is small; a baseline `mise run check` on the old pins separates sandbox
   problems from update problems.
2. **gsx v0.1.0 → v0.4.0 in one step.** Upstream notes: v0.2.0 CSS-filters styles in attrs bags (trusted CSS needs
   `gsx.RawCSS`), v0.3.0 JS-encodes Go values on `on…` attributes (handler code must be `js` literals — this repo
   already uses them, AGENTS.md), v0.4.0 adds `/*line*/` comments to `.x.go`. All require regenerating, which
   `mise run check` does. v0.1.1 fixed gsx#200 (nested modules); AGENTS.md's rule against nested `.gsx` modules
   stays until it's checked here — the rule costs nothing.
3. **workerd moves through its documented path** (kit/i18n/README "Moving a pin"): `i18npins -write`, `mise install`,
   `i18n:golden`, `go test ./kit/i18n`, `i18n:verify`. If the new release changed Chromium ICU or the CLDR major, new
   `known` entries need evidence; if that turns into real work, workerd is split out rather than holding the rest back.
4. **Go 1.27.2** also matters for tzdata: the i18n plan notes 1.27.1 is wrong for Inuvik from 2026-11-01.
5. **gsxui:** only the CLI pin moves. Vendored components in `ui/` are not re-added (that's a UI change, not a
   dependency bump); `gsxui.json` hashes stay valid.

6. **chromedp stays on v0.16.0.** v0.17–v0.21 is a ground-up rewrite (generic `Action[T]`, `Run` returns values,
   `Events` iterators instead of `ListenTarget`, typed cdproto; an 11-part `docs/MIGRATION.md`), with v0.20/v0.21 out
   in the last two weeks and the API still moving per release. Rewriting the e2e suite against it is its own change,
   once it settles. `ExecPath` (for `E2E_CHROME`) exists in v0.16.

## Steps

- [ ] Baseline: `mise run check` on the old pins
- [ ] Bump tools in `mise.toml`, Go toolchain in both `go.mod`s, CI mise version; `mise install`
- [ ] workerd via `i18npins -write`; regenerate goldens; `go test ./kit/i18n`; `mise run i18n:verify`
- [ ] `go get` gsx, workers-go, x/net (app) and chromedp (e2e); `go mod tidy` both
- [ ] Update version mentions: README Stack table, AGENTS.md, `main.go`, `tasks/app.toml`, `tasks/upstream.toml`,
      kit/i18n/README
- [ ] `mise run check` (as CI)
- [ ] `mise run e2e` (as CI)
- [ ] `mise run rename` on a scratch clone + `check` (AGENTS.md: keep rename working)
- [ ] Commit locally; push and PR only when Gerard says

## Findings

- **`mise upgrade --bump --dry-run` agrees** with these pins. It holds back releases younger than mise's 24 h
  `minimum_release_age`: Go 1.27.3, fnox 1.41.0 and usage 7.1.0 (all out 2026-10-20). Kept to mise's policy; they go
  in on the next bump.
- **workerd 1.20261019.1** builds with a newer Chromium ICU commit (`19dfb44a…`) but the same ICU 78.2 and CLDR 48, so
  the CLDR pins don't move.
- **mise 2026.10 fetches a `SKILL.md` for fnox and usage** (packslip backend) from raw.githubusercontent.com. In a
  sandbox whose `GITHUB_TOKEN` only covers some repos, that fetch 404s and the install fails ("declared skill(s) …
  are not installed"). Unset the token and it installs. Not an issue on a normal machine or in CI.
- **e2e needs a Chrome on PATH.** CI's runner has one; elsewhere, Chrome for Testing's `chrome-headless-shell` linked as
  `headless-shell` works with chromedp.
- **CI walk-through** (what the PR will run):
  - `check.yml` → `ci/check.sh`: `mise install` now also fetches the pinned Chrome (~7 s); `e2e` uses it instead of the
    runner's Chrome, `CI=true` still adds `--no-sandbox`. Its `GITHUB_TOKEN` is a real Actions token that reads public
    repos, so the sandbox's skill-file 404 shouldn't recur — only a GitHub run proves it.
  - `i18n-verify.yml` also runs (the PR touches `mise.toml`). `ci/i18n-verify.sh` pinned mise separately at v2026.9.5:
    now v2026.10.14 like `check.sh`, and `min_version` in `mise.toml` makes an older mise (a Mac) stop with a clear
    message. `i18npins -latest -strict` passes: 1.20261019.1 is the newest workerd.
- **`i18n:verify` never generated the views** (pre-existing bug, first triggered by this PR touching `mise.toml`): its
  last step, `go test -run TestReferencePins .`, compiles the app package, whose `views/*.x.go` aren't committed. A
  fresh checkout (CI) failed with `undefined: views.BoardPage`; locally they were left over from `check`. Fixed with
  `depends = ["generate"]`, proven in a fresh clone (fails before, passes after). Everything it verifies passed on
  GitHub, which also confirmed workerd 1.20261020.1 (released today) uses the same Chromium ICU.
- **chromedp v0.21 broke e2e at compile time** (`cannot use generic type chromedp.Action[T any] without
  instantiation`). Missed at first because a `| tail` hid the exit code of the local run; caught on reading the log.
  Held back (decision 6).
