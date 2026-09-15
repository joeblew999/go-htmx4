# gsx language server in VS Code for everyone who clones the repo

Goal: the VS Code gsx extension (`gsxhq.gsx`) starts `gsx lsp` on a fresh clone after `mise install`, with no
per-machine settings. `.vscode/` is committed.

## Findings (2026-09-15)

- The extension looks for the compiler in `gsx.server.path`, then PATH, then GOBIN, then GOPATH/bin
  (gsx Editor support guide: https://github.com/gsxhq/gsx/blob/main/docs/guide/editor.md). Each candidate must print
  `gsx …` from `gsx version` within 3 s.
- The old setup pinned gsx in `mise.toml` "for the VS Code language server". That only worked when VS Code was started from a
  shell in this repo. VS Code's PATH comes from wherever it was launched (here: another project's mise env). The mise
  shim was on PATH, but the extension runs `gsx version` and `go env` in the extension host's working directory, and
  outside the repo the shim fails (`No version is set for shim: gsx`). On 2026-09-15 07:12 the extension logged
  `No gsx compiler found on PATH/GOBIN/GOPATH/bin`.
- `gsx.server.path` is used as-is (no `${workspaceFolder}` or `~`), so a committed value can only be one machine's
  absolute path. mise-vscode's shared-settings mode writes `${workspaceFolder}/.vscode/mise-tools/…`, which gsx can't use,
  and its PATH injection finished after gsx had already given up.
- The extension's *Install gsx* prompt runs `go install …@latest`, not the pinned version.
- Claude Code sessions on this machine set `GOPATH=/Users/apple/workspace/go` (`~/.claude/settings.json`); the user's
  terminal and VS Code use `~/go`. The install lands in whichever GOPATH `mise install` runs with.

## Decision

Install task (chosen by the user over relying on mise-vscode or waiting for an upstream fix): `mise install`'s
postinstall hook runs `editor:gsx` (`tasks/editor.toml`), which runs `go install github.com/gsxhq/gsx/cmd/gsx` from the
module into `GOPATH/bin`. go.mod is the only gsx pin (the same version `go tool gsx` builds with).

## Steps

- [x] `tasks/editor.toml` `editor:gsx` + `[hooks] postinstall` in `mise.toml`; dropped the `go:github.com/gsxhq/gsx/cmd/gsx` pin
- [x] `.vscode/` un-ignored; `.vscode/extensions.json` recommends `gsxhq.gsx` (no settings with machine paths)
- [x] README (VS Code note) and AGENTS.md (Toolchain) updated
- [x] Verified: `mise install` runs the hook and installs `gsx v0.1.0` to `~/go/bin`. A replay of the extension's
  discovery with VS Code's real environment, from `/`, rejects the shim and picks `~/go/bin/gsx`
- [x] `mise run check` passes
- [x] Old mise gsx install and shim removed (`mise uninstall`; `mise prune` kept it because two stale scratch clones
  from another session still pin it)
- [x] Upstream: https://github.com/gsxhq/vscode-gsx/issues/9 (expand `${workspaceFolder}`/`~` in `gsx.server.path`,
  run discovery from the workspace folder, install the go.mod version)
- [ ] User: run **gsx: Restart Language Server** and confirm the output channel logs `…/go/bin/gsx lsp`
- [ ] When vscode-gsx#9 ships: commit a `gsx.server.path` using `${workspaceFolder}` (or rely on workspace-folder
  discovery) and drop the GOPATH/bin install

## Known limits of the workaround

- VS Code needs some `go` on its PATH (the extension runs `go env GOPATH`).
- `GOPATH/bin/gsx` is shared by every project on the machine; a project pinning another gsx version overwrites it.
- A `mise install` run with a different GOPATH (e.g. from Claude Code here) installs to that GOPATH's `bin`. Re-run
  `env -u GOPATH mise run editor:gsx` from a normal shell if VS Code can't find the new version.
