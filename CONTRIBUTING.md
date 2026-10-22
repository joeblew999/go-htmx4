# Contributing

go-htmx4 is a demo and a proving ground: a gsx + htmx 4 app on Cloudflare Workers that uses
[intl](intl/README.md) and [seo](seo/README.md), Go modules any gsx app can add, and
[charter](https://github.com/joeblew999/charter) for everything Cloudflare. Fixes and ideas for any of them are welcome.

## Set up

1. Install [mise](https://mise.jdx.dev). `mise install` then brings every tool this repo pins (Go, TinyGo, workerd,
   gsx, Chrome for the browser tests). There is no Node anywhere.
2. `mise run dev` runs the app on local workerd at http://localhost:8913, rebuilt on every save.

`docs/commands.md` lists every task; `docs/architecture.md` says how the app works.

## A pull request

- Start from an issue, or open one: plans and bugs live there. Say `Closes #N` in the pull request.
- Run `mise run check` before you push (it is what CI runs), and `mise run e2e` after a change to pages, the board or
  JavaScript.
- CI runs the same checks on your pull request, read-only: no secrets, nothing published, no preview.

## Text and translations

Every word a reader sees is a message in `locales/en.toml`, never text in the markup (the `intl` skill and
[intl/README.md](intl/README.md) say how to write one). The app ships in 14 languages; you don't need to know them:

- **Leave the translations to us.** CI on your pull request reports the messages your change leaves untranslated
  without failing, and the maintainer translates them after merging (`mise run intl:translate`). A release never goes
  out with anything missing.
- **Or translate one you know well:** edit its catalog, `locales/<language>.toml`; `mise run intl:todo` lists what is
  left.

## The rules

[AGENTS.md](AGENTS.md) holds the rules every change follows, whoever writes it: no Node, every visual piece a gsxui
component, htmx 4 syntax, every word and number through intl, and so on. A few of its steps are the maintainer's,
marked there: merging, releases and deploys, and anything that needs Cloudflare or Google credentials.

`intl/`, `seo/` and `badge/` are Go modules of their own, with their own READMEs and versions; the fresh-app tests
(`e2e/fresh`) build apps from nothing on them, the way an outside developer would.
