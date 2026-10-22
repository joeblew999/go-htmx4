# gsxkit

[gsx](https://github.com/gsxhq/gsx) and [gsxui](https://ui.gsxhq.dev) in any app: their mise tasks, and gsx's own
agent skill, written into the app the way [intl](../intl/README.md) and [seo](../seo/README.md) write theirs. gsx and
gsxui are Jackie Li's. gsxkit only runs their CLIs and delivers gsx's skill; it adds no templates, CSS or rules of its
own.

## Set up an app

```sh
go get -tool github.com/joeblew999/go-htmx4/gsxkit/cmd/gsxkit
go tool gsxkit tasks   # .mise/tasks/gsx.toml: the tasks below
go tool gsxkit skill   # .claude/skills/gsx: gsx's own skill, at the gsx version in go.mod
```

The tasks run gsx at the version in the app's go.mod (`go tool gsx`) and the gsxui CLI from its mise.toml, so pin
gsxui there (it has no tags yet: a commit, `"go:github.com/gsxhq/gsxui/cmd/gsxui" = "ee41247"`). Tailwind comes with
`gsxui:css`. Run `gsx:editor` after every `mise install` with `[hooks] postinstall = { task = "gsx:editor" }`.

| Task | What it does |
| --- | --- |
| `gsx:generate` | `go tool gsx generate`, after intl's messages when intl runs too |
| `gsx:fmt [--check]` | format every `.gsx` file; `--check` fails on an unformatted one (CI) |
| `gsx:editor` | install go.mod's gsx where the VS Code gsx extension looks |
| `gsxui:css [out]` | gsxui's CSS (`gsxui.json`'s entry) compiled by the standalone Tailwind CLI, gsxui's npm-free mode |
| `gsxui:theme <style \| code>` | switch the look with `gsxui apply`: a style's standard theme (nova, vega, maia, lyra, mira, luma, sera, rhea), or any share code from gsxui's [theme editor](https://ui.gsxhq.dev/theme) |
| `gsx:tasks`, `gsx:skill` | write the tasks and the skill again, after upgrading gsxkit or gsx |

## Why mise and a skill in the repo, not gsx's Claude Code plugin

gsx publishes its skill as a Claude Code plugin (the `gsx` marketplace in its repository). gsxkit copies the same skill
into the app instead, unchanged, for the same reason intl, seo and charter put their tasks and skills there: every
piece of an app arrives one way. A Go tool, at the version go.mod pins, writes its tasks and skill into the repository.
The app then carries them, CI and every agent see the same ones, and they change only when go.mod does. mise is what
runs the tasks, from a terminal, CI or VS Code's mise panel. We may switch to gsx's plugin later. When gsxui publishes
a skill of its own ([gsxhq/gsxui#36](https://github.com/gsxhq/gsxui/issues/36)), gsxkit will deliver it the same way.
