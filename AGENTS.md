# Plans

- Write all plans to the `.plans/` folder in the repository root — never anywhere else.
- Prefix every plan file name with a date-time stamp so they sort chronologically:
  `YYYY-MM-DD_HHMM_short-kebab-slug.md` (local time), e.g. `2026-09-13_1105_add-auth.md`.
- Before starting non-trivial work, check `.plans/` for an existing plan on the same topic
  and update it rather than creating a duplicate.
- Keep plans current as work progresses: tick off completed steps and note decisions or
  changes of direction.
- Commit plans alongside the code they describe.

# Toolchain

- All tools are pinned in `mise.toml` (Go 1.27.1, TinyGo 0.42.0, binaryen, watchexec).
  Run `mise install` first; use `mise run <task>` rather than ad-hoc commands.
- When bumping Go, update `mise.toml`, `go.mod` (`go` + `toolchain` lines) and the README
  Stack table together.

# Commands

- Run: `mise run run` (serves on http://localhost:8080, override with `ADDR=:3000`)
- Dev (auto-restart): `mise run dev`
- Build: `mise run build` → `bin/go-htmx4`
- Check (fmt + vet + test): `mise run check`
- List all tasks: `mise tasks`

# Project Structure

- `main.go` - HTTP server, routes and handlers.
- `templates/` - page template plus named fragment templates (`{{define "name"}}`)
  returned by htmx endpoints.
- `static/` - embedded static assets, including vendored `htmx.min.js` (htmx 4).
- `.plans/` - timestamped plans (see above).

# Code Style

- Follow standard Go conventions (Effective Go); standard library only unless there is a clear need.
- htmx endpoints return HTML fragments rendered via `render(w, "<template>", data)`.
