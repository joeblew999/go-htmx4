#!/usr/bin/env bash
# The Go modules other apps use (badge, intl, seo): at a release, a new version of each that changed since its last tag, on
# the released commit (#69), so an app gets every fix with go get, never by pointing at main. While v0, the next minor
# (intl/v0.5.0 → intl/v0.6.0) for any change.
#
#   ci/module-tags.sh SHA             tag what changed (CI's release step; GITHUB_TOKEN, GITHUB_REPOSITORY)
#   ci/module-tags.sh --dry-run SHA   only say what it would tag
set -euo pipefail
dry=
if [ "${1:-}" = --dry-run ]; then
  dry=1
  shift
fi
sha=${1:?the released commit}
api="https://api.github.com/repos/${GITHUB_REPOSITORY:?}"
auth=(-H "Authorization: Bearer ${GITHUB_TOKEN:?}" -H "Accept: application/vnd.github+json")
jq() { mise exec -- jq "$@"; }

for m in badge intl seo; do # badge first: intl and seo require it
  last=$(curl -fsS "${auth[@]}" "$api/git/matching-refs/tags/$m/v" | jq -r '.[].ref | ltrimstr("refs/tags/")' | sort -V | tail -1)
  if [ -z "$last" ]; then
    next=$m/v0.1.0 why="its first version"
  else
    changed=$(curl -fsS "${auth[@]}" "$api/compare/$last...$sha" |
      jq --arg dir "$m/" '[.files[]?.filename | select(startswith($dir))] | length')
    if [ "$changed" = 0 ]; then
      echo "· $m: unchanged since $last"
      continue
    fi
    v=${last#"$m"/v}
    major=${v%%.*} rest=${v#*.}
    next="$m/v$major.$((${rest%%.*} + 1)).0" why="$changed files changed since $last"
  fi
  if [ -n "$dry" ]; then
    echo "· $m: would tag $next ($why)"
    continue
  fi
  curl -fsS "${auth[@]}" -X POST "$api/git/refs" -d "{\"ref\":\"refs/tags/$next\",\"sha\":\"$sha\"}" >/dev/null ||
    { echo "✗ couldn't create $next"; exit 1; }
  echo "✓ $m: $next → ${sha:0:7} ($why)"
done
