# gsxkit

[gsx](https://github.com/gsxhq/gsx), [gsxui](https://ui.gsxhq.dev) and [structpages](https://github.com/jackielii/structpages)
in any app: gsx's and gsxui's mise tasks, and the agent skills these projects ship, written into the app the way [intl](../intl/README.md) and [seo](../seo/README.md) write theirs. All three are
Jackie Li's. gsxkit only runs their CLIs and copies their skills; it adds no templates, CSS or rules of its own.

## Set up an app

```sh
go get -tool github.com/joeblew999/go-htmx4/gsxkit/cmd/gsxkit
go tool gsxkit tasks   # .mise/tasks/gsx.toml: the tasks below
go tool gsxkit skill   # .claude/skills: each project's own skills, at the versions in go.mod
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
| `gsx:tasks`, `gsx:skill` | write the tasks and the skills again, after upgrading gsxkit, gsx or structpages |

## The skills

`go tool gsxkit skill` copies every skill in the `skills/` of each of these projects the app's go.mod requires, unchanged:
gsx's (`gsx`, `templ-to-gsx-migration`) and structpages' (`structpages`). A project the app doesn't use is skipped.

## Why mise and skills in the repo, not their Claude Code plugins

gsx and structpages publish their skills as Claude Code plugins (a marketplace in each repository). gsxkit copies the
same skills into the app instead, unchanged, for the same reason intl, seo and charter put their tasks and skills there: every
piece of an app arrives one way. A Go tool, at the version go.mod pins, writes its tasks and skill into the repository.
The app then carries them, CI and every agent see the same ones, and they change only when go.mod does. mise is what
runs the tasks, from a terminal, CI or VS Code's mise panel. We may switch to their plugins later. When gsxui publishes
a skill of its own ([gsxhq/gsxui#36](https://github.com/gsxhq/gsxui/issues/36)), gsxkit will deliver it the same way.
