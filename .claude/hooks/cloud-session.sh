#!/usr/bin/env bash
# cloud-session.sh — Claude Code SessionStart hook (.claude/settings.json): make a Claude cloud session ready for this
# repo's mise tooling. Does nothing anywhere else (your Mac, CI): it only acts when CLAUDE_CODE_REMOTE=true.
#
# - mise: installed if missing (https://mise.run) and put on PATH; the repo's mise.toml trusted.
# - GITHUB_TOKEN hidden: the cloud session's token only reaches repositories attached to the session, and mise sends
#   it when it downloads tools' files from GitHub, so every download from any other repository fails (404) and with
#   it every mise command. gh uses GH_TOKEN, which stays.
#
# Tools themselves are installed by mise on first use (mise run <task>), not here, so the session starts quickly.
set -euo pipefail
[ "${CLAUDE_CODE_REMOTE:-}" = true ] || exit 0

bin="$HOME/.local/bin"
if [ ! -x "$bin/mise" ]; then
  curl -fsSL https://mise.run | sh >/dev/null
fi
"$bin/mise" trust -q "${CLAUDE_PROJECT_DIR:-.}"

# CLAUDE_ENV_FILE is sourced before every command Claude runs in this session.
if [ -n "${CLAUDE_ENV_FILE:-}" ]; then
  {
    echo "export PATH=\"$bin:\$PATH\""
    echo "unset GITHUB_TOKEN"
  } >>"$CLAUDE_ENV_FILE"
fi
