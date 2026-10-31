# gsxkit

[gsx](https://github.com/gsxhq/gsx), [gsxui](https://ui.gsxhq.dev) and [structpages](https://github.com/jackielii/structpages)
in any app: gsx's and gsxui's mise tasks, and the agent skills these projects ship, written into the app the way [intl](../intl/README.md) and [seo](../seo/README.md) write theirs. All three are
Jackie Li's; gsxkit runs their CLIs and copies their skills unchanged.

And [components gsxui doesn't have](#components), a registry of our own: a map and live video, made of gsxui
components, added to an app with `go tool gsxkit add` the way `gsxui add` adds gsxui's.

## Set up an app

```sh
go get -tool github.com/joeblew999/go-htmx4/gsxkit/cmd/gsxkit
go tool gsxkit tasks   # .mise/tasks/gsx.toml: the tasks below
go tool gsxkit skill   # .claude/skills: each project's own skills, at the versions the app pins
go get -tool github.com/jackielii/structpages/tools/lint/cmd/structpages-lint   # for structpages:lint
```

The tasks run gsx at the version in the app's go.mod (`go tool gsx`), structpages' linter from go.mod too, and the
gsxui CLI from its mise.toml, so pin gsxui there (it has no tags yet: a commit,
`"go:github.com/gsxhq/gsxui/cmd/gsxui" = "65f4813e"`). Tailwind comes with `gsxui:css`. Run `gsx:editor` after every
`mise install` with `[hooks] postinstall = { task = "gsx:editor" }`. Make gsx's lints errors in `gsx.toml` (`[lint]`,
each lint `"error"`; `go tool gsx info` lists them), so `gsx generate` fails on one, and run `structpages:lint` in the
app's checks.

| Task | What it does |
| --- | --- |
| `gsx:generate` | `go tool gsx generate`, after intl's messages when intl runs too |
| `gsx:fmt [--check]` | format every `.gsx` file; `--check` fails on an unformatted one (CI) |
| `gsx:editor` | install go.mod's gsx where the VS Code gsx extension looks |
| `gsxui:css [out]` | gsxui's CSS (`gsxui.json`'s entry) compiled by the standalone Tailwind CLI, gsxui's npm-free mode |
| `gsxui:theme <style \| code>` | switch the look with `gsxui apply`: a style's standard theme (nova, vega, maia, lyra, mira, luma, sera, rhea), or any share code from gsxui's [theme editor](https://ui.gsxhq.dev/theme) |
| `structpages:lint` | structpages-lint over the app: links, ids and partials against the page tree, internal URLs written as strings, htmx without a target selector |
| `gsx:tasks`, `gsx:skill` | write the tasks and the skills again, after upgrading gsxkit, gsx, gsxui or structpages |

## The skills

`go tool gsxkit skill` copies every skill in the `skills/` of each of these projects the app uses, unchanged, at the
version it pins: gsx's (`gsx`, `templ-to-gsx-migration`) and structpages' (`structpages`) at go.mod's versions, and
gsxui's (`gsxui`) at the commit mise.toml installs its CLI at. A project the app doesn't use is skipped.

## Why mise and skills in the repo, not their Claude Code plugins

gsx and structpages publish their skills as Claude Code plugins (a marketplace in each repository). gsxkit copies the
same skills into the app instead, unchanged, for the same reason intl, seo and charter put their tasks and skills there: every
piece of an app arrives one way. A Go tool, at the version go.mod pins, writes its tasks and skill into the repository.
The app then carries them, CI and every agent see the same ones, and they change only when go.mod does. mise is what
runs the tasks, from a terminal, CI or VS Code's mise panel. We may switch to their plugins later. gsxui's skill
([gsxhq/gsxui#58](https://github.com/gsxhq/gsxui/pull/58)) arrives the same way.

## Components

gsxkit's components are a registry beside gsxui's: each one a gsx file made of gsxui components, its text in every
language [intl](../intl/README.md) ships, and, for one that runs in the browser, its script and stylesheet, bundled
from npm with no Node. `go tool gsxkit add` copies them into the app, which owns its copies from then on, as gsxui's:

```sh
go tool gsxkit add map moq   # views/gsxkit/*.gsx, web/gsxkit/*, their text in the catalogs
```

It runs `gsxui add` for the gsxui components they are made of, writes the gsx into `views/gsxkit` (package `gsxkit`;
`-dir` for another folder), the browser files into `web/gsxkit` (`-assets`), and the text into the app's catalogs under
`[gsxkit.<component>]`, already translated. A copy the app changed is left alone: `-diff` shows how it differs,
`-overwrite` replaces it. The app needs gsxui and intl set up first (`gsxui init`, `go tool intl init`), serves
`web/gsxkit` at `/gsxkit/`, and allows in its Content Security Policy what each component loads.

| Component | What it is | Its parts | It loads |
| --- | --- | --- | --- |
| `map` | A [Leaflet](https://leafletjs.com) map over [OpenStreetMap](https://www.openstreetmap.org)'s tiles: markers with gsx popups, zoom and the reader's own location as gsxui buttons, the attribution OpenStreetMap asks for. Works on a page reached by a boosted navigation. | `Map`, `MapMarker`, `MapPin`, `LatLng` | `/gsxkit/map.js`, `/gsxkit/map.css`; tiles from `https://tile.openstreetmap.org` (img-src) |
| `moq` | Live video over Media over QUIC: a camera broadcast with its link to send (copy, share), and a player, on [@moq](https://github.com/moq-dev/moq)'s elements; connection details for testing on phones. | `MoqBroadcast`, `MoqWatch` | `/gsxkit/moq.js`; the relay (connect-src), `blob:` (script-src, worker-src) |

```gsx
<gsxkit.Map center={gsxkit.LatLng{48.137, 11.575}} zoom={12} label={locales.MapLabel} locate={true}>
	<gsxkit.MapMarker at={gsxkit.LatLng{48.137, 11.575}} title="Munich">
		<p class="font-medium">Munich</p>
	</gsxkit.MapMarker>
</gsxkit.Map>

<gsxkit.MoqBroadcast url="https://draft-14.cloudflare.mediaoverquic.com/" name="my-app/live/abc.hang" link="https://example.com/live?stream=abc"/>
<gsxkit.MoqWatch url="https://draft-14.cloudflare.mediaoverquic.com/" name="my-app/live/abc.hang"/>
```

Every label is in the gsx; the browser files only hand the library what the markup rendered and set `data-*` state that
Tailwind's group-data variants read. OpenStreetMap's tiles are for light use
([its tile policy](https://operations.osmfoundation.org/policies/tiles/)): a busy site points `tiles` in its copy at a
provider of its own. The `moq` component holds @moq at its last releases that speak moq-transport drafts 14 and 16,
which Cloudflare's relays speak: when they ship draft-18, @moq moves on and gsxkit's patches to it go
([#158](https://github.com/joeblew999/go-htmx4/issues/158)).

### Making one

A component is `ui/<name>.gsx.tmpl` (package `gsxkit`, gsxui imported from `github.com/gsxhq/gsxui/ui`, its text from
`github.com/joeblew999/go-htmx4/gsxkit/ui/locales`: add points both at the app's), its messages in `ui/locales/*.toml`
under `[gsxkit.<name>]`, and its entry in `ui/ui.go`. One that runs in the browser has `js/<name>/<name>.mjs` and the
`npm-lock.json` its npm packages are pinned in (`npm-patches.json` for patches, each with its reason), bundled into
`ui/assets/` by go-htmx4's `mise run gsxkit:bundle` (`gsxkit:update <name> <package@version>…` to move a package). The
copying is [intl's registry](../intl/registry/registry.go), which intl's own screens use too.
