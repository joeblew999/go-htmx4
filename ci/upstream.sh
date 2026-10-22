#!/usr/bin/env bash
# Upstream work from this repo (#64): checkouts of the projects go-htmx4 builds on, worked on with this repo's tools.
#
#   ci/upstream.sh status                 every checkout: where it is, its branch, local changes, the fork's main
#   ci/upstream.sh sync [repo]            clone or fetch (every repo without one); never pushes
#   ci/upstream.sh fork <repo>            create the user's fork on GitHub and add it as origin
#   ci/upstream.sh branch <repo> <name>   a new branch from upstream's main
#   ci/upstream.sh check <repo>           the repo's Go checks with its own pinned Go; failures re-run on upstream's main
#   ci/upstream.sh tinygo <repo> [pkg]    the package's own tests compiled by this repo's TinyGo
#   ci/upstream.sh use <repo>             this app on the local checkout: go.work for a Go module, gsxui re-vendored
#   ci/upstream.sh unuse <repo>           back to the pinned version
#   ci/upstream.sh review <repo>          the commits and diff against upstream's main, to read before anything is sent
#
# A checkout is $(go env GOPATH)/src/github.com/<owner>/<repo>, where Go has always put them: remote upstream is the
# original, origin the user's fork when there is one. Nothing here sends anything to GitHub except fork, which creates
# the user's own fork when asked: a PR or an issue on another owner's repository is opened by hand, after the user has
# read it (AGENTS.md). No npm: a check that needs it fails the same way on upstream's main and is reported as such.
set -euo pipefail
cd "$(dirname "$0")/.."
root=$PWD

# The projects, as name=owner/repo.
repos="gsx=gsxhq/gsx gsxui=gsxhq/gsxui structpages=jackielii/structpages"
names() { for r in $repos; do echo "${r%%=*}"; done; }
slug() {
  local r
  for r in $repos; do [ "${r%%=*}" = "$1" ] && { echo "${r#*=}"; return; }; done
  echo "✗ $1: not an upstream project (one of: $(names | tr '\n' ' '))" >&2
  exit 2
}
dir() { echo "$(go env GOPATH)/src/github.com/$(slug "$1")"; }
need() { [ -d "$(dir "$1")/.git" ] || { echo "✗ $1 isn't checked out: mise run upstream:sync $1" >&2; exit 1; }; }

# The user's fork of owner/repo on GitHub, "" without one (a fork may have another name).
fork_of() {
  gh repo list --fork --limit 500 --json nameWithOwner,parent \
    -q ".[] | select(.parent.owner.login + \"/\" + .parent.name == \"$1\") | .nameWithOwner" | head -1
}

# The Go a checkout pins: go.mod's toolchain line, else its go line.
go_of() {
  local v
  v=$(sed -n 's/^toolchain go//p' "$1/go.mod")
  [ -n "$v" ] || v=$(sed -n 's/^go //p' "$1/go.mod")
  echo "$v"
}
# with_go DIR COMMAND…: run in DIR with DIR's pinned Go (mise installs it if needed), whatever GOROOT the shell has.
with_go() {
  local d=$1 v
  shift
  v=$(go_of "$d")
  mise install -q "go@$v" >/dev/null 2>&1 || true
  (cd "$d" && env -u GOROOT -u GOTOOLCHAIN mise exec "go@$v" -- "$@")
}

behind_ahead() { git -C "$1" rev-list --left-right --count "$2...$3" 2>/dev/null | awk '{print $2 " ahead, " $1 " behind"}'; }

cmd_status() {
  local n d b
  for n in $(names); do
    d=$(dir "$n")
    if [ ! -d "$d/.git" ]; then
      echo "$n: not checked out (mise run upstream:sync $n)"
      continue
    fi
    b=$(git -C "$d" branch --show-current)
    echo "$n: $d"
    echo "  branch ${b:-(detached)}: $(behind_ahead "$d" upstream/main HEAD) upstream/main; $(git -C "$d" status --porcelain | wc -l | tr -d ' ') changed files"
    if git -C "$d" remote get-url origin >/dev/null 2>&1; then
      echo "  fork $(git -C "$d" remote get-url origin): its main $(behind_ahead "$d" upstream/main origin/main) upstream/main"
    else
      echo "  no fork (mise run upstream:fork $n to make one)"
    fi
  done
  [ ! -s .upstream-use ] || echo "this app uses the checkouts of: $(tr '\n' ' ' <.upstream-use)(mise run upstream:unuse <repo>)"
}

cmd_sync() {
  local n d s f
  for n in ${1:-$(names)}; do
    s=$(slug "$n")
    d=$(dir "$n")
    if [ ! -d "$d/.git" ]; then
      mkdir -p "$(dirname "$d")"
      git clone -q -o upstream "https://github.com/$s.git" "$d"
      echo "✓ $n: cloned into $d"
    fi
    if ! git -C "$d" remote get-url origin >/dev/null 2>&1; then
      f=$(fork_of "$s")
      [ -z "$f" ] || git -C "$d" remote add origin "https://github.com/$f.git"
    fi
    git -C "$d" fetch -q --all --prune
  done
  cmd_status
}

cmd_fork() {
  local n=$1 s d f
  need "$n"
  s=$(slug "$n")
  d=$(dir "$n")
  f=$(fork_of "$s")
  if [ -z "$f" ]; then
    gh repo fork "$s" --clone=false --remote=false >/dev/null
    f=$(fork_of "$s")
  fi
  git -C "$d" remote get-url origin >/dev/null 2>&1 || git -C "$d" remote add origin "https://github.com/$f.git"
  git -C "$d" fetch -q origin
  echo "✓ $n: your fork is $f (remote origin)"
}

cmd_branch() {
  local n=$1 name=$2 d
  need "$n"
  d=$(dir "$n")
  [ -z "$(git -C "$d" status --porcelain)" ] || { echo "✗ $d has uncommitted changes" >&2; exit 1; }
  git -C "$d" fetch -q upstream
  git -C "$d" switch -q -c "$name" upstream/main
  echo "✓ $n: branch $name from upstream/main ($(git -C "$d" log -1 --format=%h upstream/main)) in $d"
}

# run_checks DIR LOG: the Go checks a checkout has, with its own Go; the failing packages go to LOG.failed.
run_checks() {
  local d=$1 log=$2 status=0
  : >"$log.failed"
  if grep -q 'github.com/gsxhq/gsx/cmd/gsx' "$d/go.mod"; then
    with_go "$d" go tool gsx generate -q . || status=1
    local unformatted
    unformatted=$(with_go "$d" go tool gsx fmt -l .) || status=1
    [ -z "$unformatted" ] || { echo "✗ gsx fmt: $unformatted"; status=1; }
  fi
  # Checks a repo keeps as commands of its own (gsxui's), run when it has them.
  [ ! -d "$d/internal/generatedcheck/cmd" ] || with_go "$d" go run ./internal/generatedcheck/cmd || status=1
  [ ! -d "$d/cmd/stylegen" ] || with_go "$d" go run ./cmd/stylegen --check || status=1
  with_go "$d" go vet ./... || status=1
  with_go "$d" go test ./... >"$log" 2>&1 || status=1
  sed -n 's/^FAIL[[:space:]]\{1,\}\([^[:space:]]\{1,\}\).*/\1/p' "$log" | sort -u >"$log.failed"
  return $status
}

cmd_check() {
  local n=$1 d log base new=0 p
  need "$n"
  d=$(dir "$n")
  log=$(mktemp)
  echo "── $n at $(git -C "$d" log -1 --format='%h %s') with Go $(go_of "$d")"
  if run_checks "$d" "$log"; then
    echo "✓ $n: every check passed"
    return
  fi
  if [ ! -s "$log.failed" ]; then
    echo "✗ $n: a check failed (above)"
    exit 1
  fi
  grep -E '^(--- FAIL|FAIL|panic)|_test\.go:[0-9]+' "$log" | head -40
  # Failing packages again on upstream's main: what fails there too isn't this branch's doing.
  base=$(mktemp -d)
  git -C "$d" worktree add -q --detach "$base/main" upstream/main
  run_checks "$base/main" "$log.main" >/dev/null 2>&1 || true
  git -C "$d" worktree remove --force "$base/main"
  echo
  while read -r p; do
    if grep -qx "$p" "$log.main.failed"; then
      echo "· $p fails on upstream/main too: not this branch"
    else
      echo "✗ $p: fails here, passes on upstream/main"
      new=1
    fi
  done <"$log.failed"
  [ $new = 0 ] || exit 1
}

cmd_tinygo() {
  local n=$1 pkg=${2:-./...} d
  need "$n"
  d=$(dir "$n")
  echo "── $n $pkg: its tests compiled by TinyGo $(tinygo version | awk '{print $3}') (the reflect Workers builds get)"
  (cd "$d" && tinygo test "$pkg")
}

go_modules() { git ls-files '*go.mod' | sed 's#/\{0,1\}go.mod$##; s#^$#.#'; }

cmd_use() {
  local n=$1 d
  need "$n"
  d=$(dir "$n")
  ! grep -qx "$n" .upstream-use 2>/dev/null || { echo "· $n: already in use"; return; }
  if [ "$n" = gsxui ]; then
    # gsxui is vendored, not imported: its CLI, built from the checkout, re-vendors every component this app has.
    mkdir -p .upstream
    with_go "$d" go build -o "$root/.upstream/gsxui" ./cmd/gsxui
    jq -r '.managed | keys[]' gsxui.json | sed -n 's#^ui/\([a-z0-9-]*\)\.gsx$#\1#p' | xargs .upstream/gsxui add --overwrite
  else
    [ -f go.work ] || go work init $(go_modules)
    go work use "$d"
  fi
  echo "$n" >>.upstream-use
  echo "✓ this app now builds with $n from $d ($(git -C "$d" branch --show-current)): mise run check fails while it does"
}

cmd_unuse() {
  local n=$1 d
  d=$(dir "$n")
  grep -qx "$n" .upstream-use 2>/dev/null || { echo "· $n: not in use"; return; }
  if [ "$n" = gsxui ]; then
    jq -r '.managed | keys[]' gsxui.json | sed -n 's#^ui/\([a-z0-9-]*\)\.gsx$#\1#p' | xargs gsxui add --overwrite
  else
    go work edit -dropuse "$d"
    if [ "$(go work edit -json | jq '.Use | length')" = "$(go_modules | wc -l | tr -d ' ')" ]; then
      rm -f go.work go.work.sum
    fi
  fi
  grep -vx "$n" .upstream-use >.upstream-use.tmp || true
  mv .upstream-use.tmp .upstream-use
  [ -s .upstream-use ] || rm -f .upstream-use
  echo "✓ $n: back to the pinned version"
}

cmd_review() {
  local n=$1 d
  need "$n"
  d=$(dir "$n")
  git -C "$d" fetch -q upstream
  echo "$n: $d, branch $(git -C "$d" branch --show-current)"
  echo
  git -C "$d" log --format='%h %s' upstream/main..HEAD
  echo
  git -C "$d" diff --stat upstream/main...HEAD
  [ -z "$(git -C "$d" status --porcelain)" ] || { echo; echo "not committed:"; git -C "$d" status --short; }
}

c=${1:-status}
[ $# -eq 0 ] || shift
case $c in
  status) cmd_status ;;
  sync) cmd_sync "${1:-}" ;;
  fork | check | use | unuse | review) [ $# -ge 1 ] || { echo "usage: ci/upstream.sh $c <repo>" >&2; exit 2; }; "cmd_$c" "$1" ;;
  branch) [ $# -ge 2 ] || { echo "usage: ci/upstream.sh branch <repo> <name>" >&2; exit 2; }; cmd_branch "$1" "$2" ;;
  tinygo) [ $# -ge 1 ] || { echo "usage: ci/upstream.sh tinygo <repo> [package]" >&2; exit 2; }; cmd_tinygo "$1" "${2:-}" ;;
  *) sed -n '2,13p' "$0" | sed 's/^# \{0,1\}//'; exit 2 ;;
esac
