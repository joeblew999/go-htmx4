# Upgrade the Workers demo UI to gsxui + theme toggle in both demos

**Status:** done, deployed 2026-09-14 · **Created:** 2026-09-14 09:48

## Goal

Give `demos/workers` (the home page and the shared board) the same UI as `demos/gsxui`: gsxui components, the same
theme and fonts, Tailwind, a proper layout, instead of plain HTML + a 12-line `demo.css`. Add a **light/dark theme
toggle to both demos**, done the way gsxui's own showcase site does it. Behaviour, wire format, tests and deploys stay
as they are.

## What's already current (checked 2026-09-14)

Nothing to bump: gsxui pin `c7fd6a8` **is** upstream HEAD (0 commits ahead), gsx `v0.1.0` is the latest tag, htmx `4.0.0` is
the newest 4.x (npm `latest` is still 2.x, `next` = 4.0.0), Tailwind `v4.3.3` is the latest release. `demos/gsxui` already
uses gsxui. So "upgrade" = the Workers demo's look.

## Ground rules

- **Don't reinvent GUI: use all of gsxui's stuff** (your call, 2026-09-14). Every visual piece is a gsxui component added
  with `gsxui add`, the gsxui theme/fonts, and gsxui's own showcase-site patterns (layout, header, theme toggle). No
  hand-written CSS: `demos/workers/public/demo.css` is deleted; no custom components beyond thin page composition.
- **Use the gsx CLI for the Workers dev loop** (your call): `gsx dev` drives the TinyGo build + workerd, so we know gsx
  tooling works for Workers, not just `go tool gsx generate`.
- Same as the demos: TinyGo for what ships to Workers, no Node (gsxui **npm-free mode**, standalone Tailwind), upstream
  docs, work on `main`, local checks in the dev loop (no CI watching), deploys need an OK.
- Add components with `gsxui add` (never hand-copy); never edit `*.x.go`.

## Design

- **Setup (gsxui npm-free guide):** `gsxui init` in `demos/workers` using the **same preset** as `demos/gsxui`
  (`gsxui.preset.json`), so both demos look alike; `gsx.toml` gains the `class_merger`. Components: `button card input
  badge separator label` (+ `field` if the note form uses it).
- **Static Assets layout (same as the gsxui demo):** `/gsxui/` = `web/gsxui/` behaviours, `/assets/` = compiled
  `gsxui.css` + fonts; `public/` keeps `htmx.min.js` / `hx-ws.js`. Local workerd serves them via `assets-first.mjs`;
  deploy uploads them.
- **Layout component** (`views` or package main): header with the two pages (Demo, Shared board), consistent
  `<head>` (theme CSS, htmx, hx-ws on the board only), max-width container.
- **Home page** (`page.html` → gsx): the three demo sections as `Card`s, `Button`s for the htmx actions, `Badge` for the
  runtime target (`tinygo js/wasm`) and env, results in muted text.
- **Board:** value in a large card with −1 / +1 `Button`s, note form as `Input` + `Button`, notes as a list with muted
  timestamps, presence as a `Badge`, version in the card footer.
- **Wire format:** the Room and the page's version guard only rely on `id="board"`, `hx-swap-oob="true"` and
  `data-version="N"`. The byte-for-byte golden test gets replaced **on purpose** by a structural one (those three
  attributes + escaped content), since the markup now carries Tailwind classes.
- **Presence stays untouched in the Room:** the page wraps `<span id="presence">` inside a `Badge`, so the Room's OOB swap
  of the bare span keeps the badge styling.

## `gsx dev` for Workers (both demos)

gsx `[dev]` config (gsx Configuration guide): `build` and `run` are command arrays (no shell), `no_web = true`, and
`upstream` + `health` retarget the health probe when the backend doesn't listen on gsx's `GO_PORT`.

```toml
[dev]
no_web = true
build = ["mise", "run", "demo:workers:build"]      # workers-assets-gen + TinyGo + css (size-gated)
run = ["workerd", "serve", "config.capnp"]         # fixed :8913 from config.capnp
upstream = "http://127.0.0.1:8913"
health = "/healthz"
```

- `mise run demo:workers:dev` = `go tool gsx dev` in `demos/workers`: a `.gsx`/Go save → gsx regenerates → TinyGo build →
  workerd restart → `/healthz` probe. Same for `demos/gsxui` as `demo:gsxui:workers:dev` (:8918); the existing native
  `demo:gsxui:dev` stays.
- Verify: start it, edit a `.gsx` text, confirm the served page on workerd changes without manual steps; break the Go
  build and confirm the last working server keeps serving (gsx dev-loop guide); record rebuild time.
- Open detail to check in Phase 1: whether `gsx dev` passes the TinyGo build's non-`go` command cleanly and how it treats
  `run` exits (workerd is long-running, like a server binary).

## Theme toggle (both demos): copy gsxui's showcase site

Upstream (`.upstream/gsxui`, pin `c7fd6a8`): `site/pages/document.gsx` has a paint-blocking head script; `site/pages/layout.gsx`
has a header `<button data-site-theme-toggle aria-label="Toggle theme">` with an icon; `web/site.js` flips the class on click.
Our gsxui theme (`web/gsxui/theme.css`) already defines the `.dark { … }` tokens, so no CSS work.

- **`ThemeScript` component** in each demo's `<head>`, before any stylesheet paints:
  `try { const t = localStorage.getItem("gsxui-theme"); if (t === "dark" || (!t && matchMedia("(prefers-color-scheme: dark)").matches)) document.documentElement.classList.add("dark") } catch {}`.
  Stored choice wins, otherwise the OS preference, no flash of light on load.
- **`ThemeToggle` component** in each demo's header: a gsxui `Button` (ghost, icon size) with a sun/moon icon and
  `aria-label="Toggle theme"`; `hx-on:click=js\`…\`` flips `document.documentElement.classList.toggle("dark")` and stores
  `"dark"`/`"light"` (try/catch for private mode). No extra JS file; htmx is on every page already.
- **Easy to use:** each demo gets exactly two components in its layout (`<ThemeScript/>` in head, `<ThemeToggle/>` in the
  header). With hx-boost + `outerMorph` in the gsxui demo, the class lives on `<html>` and survives navigation.
- **Checks:** browser check: toggle adds/removes `dark` on `<html>`, colours change (computed background differs),
  reload keeps the choice, `dark` is already set at `DOMContentLoaded` (no flash), boosted nav keeps it, OS dark
  preference respected when nothing is stored (emulated `prefers-color-scheme`), no console errors.

## Plan

### Phase 1: gsxui + `gsx dev` in `demos/workers` (local)

- [x] `gsxui init` (npm-free, gsxui demo's preset), `gsxui add …`; `demo:workers:css` (standalone Tailwind) and asset
      layout; build/serve/smoke tasks pick them up.
- [x] TinyGo size check (gsxui demo on Workers is 624 KB gzip; expect similar). — 661 KB gzip (+icons, item, empty, field, button-group)
- [x] `gsx.toml` `[dev]` + `demo:workers:dev`; verify the save → regenerate → TinyGo → workerd loop; delete `demo.css`. — see Findings

### Phase 1b: theme toggle in `demos/gsxui` (local)

- [x] `demo:gsxui:workers:dev` (`gsx dev` driving the gsxui demo's TinyGo build on workerd :8918), verified the same way.
- [x] `ThemeScript` + `ThemeToggle` in `views`, wired into `Layout`; `demo:gsxui:test` + `demo:gsxui:workers:smoke`; browser
      checks above (native and TinyGo on workerd).

### Phase 2: pages

- [x] Layout + home page + board in gsx with gsxui components, as designed above, including `ThemeScript` + `ThemeToggle`.
- [x] Replace the golden fragment test with the structural wire-format test; keep `TestBoard` behaviour checks.

### Phase 3: verify (local)

- [x] `mise run check`; `demo:workers:load` (1,000 sockets, presence, burst); two-tab browser check; gsxui-style
      browser checks (fonts load, no console errors); **screenshots of both pages for your review** before deploying.

### Phase 4: deploy (⚠ needs OK)

- [x] `demo:workers:deploy` + `demo:gsxui:workers:deploy`, remote smoke, live browser checks (incl. theme toggle).

## Open questions

- Same theme/preset as the gsxui demo (recommended), or a different gsxui preset for the Workers demo?
- Resolved 2026-09-14: **theme toggle yes, in both demos**, "make sure it's easy" → the two-component upstream pattern above.

## Findings

### 2026-09-14 10:20: Phases 1–3 (local)

- **gsxui in `demos/workers`:** `gsxui init -preset ../gsxui/gsxui.preset.json` (npm-free, kept gsx v0.1.0, added the class
  merger), `gsxui add button card badge input separator label icon item empty button-group field`. `demo.css` and
  `page.html` deleted; pages are gsx composing gsxui only: `layout.gsx` (mirrors the gsxui demo layout), `home.gsx`
  (cards, field, icon buttons, badges), `board.gsx` (card + presence badge in `CardAction`, `ButtonGroup` −/+ icon buttons
  posting `delta` in the query, `Item`s for notes, `Empty` state, input + send button). Assets: `demo:workers:assets`
  compiles `web/gsxui/index.css` with standalone Tailwind and assembles `dist/site` (`/`, `/gsxui/`, `/assets/`), used by
  workerd, `go run .` and deploy. TinyGo: 2,050,089 B raw / **660,956 B gzip**.
- **Theme toggle (both demos):** `theme.gsx` = gsxui site's head init script + delegated click handler (copied, cited),
  toggle = gsxui ghost icon `Button` with `icon.SunMoon`. Browser checks: flips `dark` and body colour
  (`oklch(0.145 0 0)` ↔ `oklch(1 0 0)`), persists across reload, survives boosted nav in the gsxui demo.
- **Regression caught by the browser check:** moving page head tags after `htmx.min.js` put the `htmx-config` meta after
  htmx, so `htmx.config.ws` was `{}` and the 1–3 s reconnect spread was lost. Layout now renders `htmxConfig` before the
  htmx script; a Go test pins the order meta < htmx.min.js < hx-ws.js.
- **`gsx dev` on Workers (the gsx CLI drives the real architecture):** `demos/workers/gsx.toml` `[dev]` build =
  `mise run demo:workers:build`, run = workerd, `upstream`/`health` → :8913. First build + serve 24 s; `.gsx` edit served by
  the rebuilt TinyGo worker after 28 s (revert 23 s); a broken Go file → gsx dev logs `regen failed` and **keeps the last
  worker serving**; stopping gsx dev stops workerd. `demos/gsxui` uses `upstream = http://127.0.0.1:${GSXUI_DEV_PORT}` so one
  gsx.toml serves the native loop (7777) and `demo:gsxui:workers:dev` (8918, `-build`/`-run` flags): first build 28 s,
  edit served in 27 s.
- **Durable Object under gsx dev:** `wsload -n 2` through gsx dev's workerd: presence 2 → 1, push, pong, cache. A real
  browser tab across a gsx dev rebuild (workerd PID replaced): hx-ws reconnected and a push after the restart reached the
  tab (v1 → v2; DO SQLite state survived).
- **Checks:** Workers board browser check 13/13 (presence, push 60 ms, escaped note, form reset, version guard, reconnect
  config, theme, no console errors); gsxui demo 21/21 (+ theme); `mise run check` green (38); `demo:workers:load`: 1,000
  sockets → 1000 online / 500 after closing half, push to 1000/1000, 50-write burst → 2 broadcasts, late joiner cached.
- Screenshots (light/dark, home/board/empty state, gsxui demo) reviewed; consistent with the gsxui demo.

### 2026-09-14 10:25: Phase 4 (deployed)

- `demo:workers:deploy`: 6 modules, 24 assets from `dist/site`, D1 + Room DO migration already applied.
  `demo:gsxui:workers:deploy`: assets unchanged, script uploaded.
- `demo:workers:smoke-remote`: HTTP checks (gsxui CSS, `/gsxui/index.js`, theme toggle, board, D1 write) pass. The wsload
  step run **seconds after the deploy** failed once (no broadcast; presence stuck at 2): the deploy restarts the Room DO and
  the test straddled it. Rerun ~1 min later: all green (push 2/2, p50 226 ms, presence 2 → 1, late joiner cached).
  `demo:gsxui:workers:smoke-remote`: 17/17.
- Live browser checks: Workers board 13/13 (two tabs, presence 2 → 1, push 737 ms over a slow link, escaped note, form
  reset, version guard, reconnect config from meta, theme flip + persistence, no console errors); gsxui demo 21/21 including
  theme across boosted nav and reload. Live light/dark screenshots match local.
