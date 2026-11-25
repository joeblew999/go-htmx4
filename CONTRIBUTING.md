# Contributing

go-htmx4 is a demo and a proving ground: a gsx + htmx 4 app on Cloudflare Workers that uses
[mise-intl](https://github.com/joeblew999/mise-intl) for its languages, [mise-seo](https://github.com/joeblew999/mise-seo)
for search engines, and [charter](https://github.com/joeblew999/charter) for everything Cloudflare. Fixes and ideas
for the app are welcome here; for one of those three, in its own repository.

## Set up

1. Install [mise](https://mise.jdx.dev). `mise install` then brings every tool this repo pins (Go, workerd, gsx,
   Chrome for the browser tests). There is no Node anywhere.
2. `mise run dev` runs the app on local workerd at http://localhost:8913, rebuilt on every save.
3. To bring pins up to date, use `mise run outdated` (what's behind) and `mise run update` (move it all, then check
   and e2e), not `mise upgrade`: mise sees only its own tools, never go.mod's (gsx, structpages) or gsxui's commit.
   `deps.toml` holds back what mustn't move yet, and says why.

`docs/commands.md` lists every task; `docs/architecture.md` says how the app works.

## A pull request

- Start from an issue, or open one: plans and bugs live there. Say `Closes #N` in the pull request.
- Run `mise run check` before you push (it is what CI runs), and `mise run e2e` after a change to pages, the board or
  JavaScript.
- CI runs the same checks on your pull request, read-only: no secrets, nothing published, no preview.

## Text and translations

Every word a reader sees is a message in `locales/en.toml`, never text in the markup (the `intl` skill and
[mise-intl's guide](https://github.com/joeblew999/mise-intl/blob/main/templates/docs/intl/README.md) say how to write one). The app ships in 14 languages; you don't need to know them:

- **Leave the translations to us:** say so in the pull request, and the maintainer translates them before merging
  (`mise run intl:translate`). The check fails while a language is incomplete, so nothing ships with a gap.
- **Or translate one you know well:** edit its catalog, `locales/<language>.toml`; `mise run intl:todo` lists what is
  left.

## The rules

[AGENTS.md](AGENTS.md) holds the rules every change follows, whoever writes it: no Node, every visual piece a gsxui
component, htmx 4 syntax, every word and number through intl, and so on. A few of its steps are the maintainer's,
marked there: merging, releases and deploys, and anything that needs Cloudflare or Google credentials.
