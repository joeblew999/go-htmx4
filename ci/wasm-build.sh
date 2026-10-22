#!/usr/bin/env bash
# The Worker's wasm build into build/ (mise run build / build:dev).
#
#   mise run build      production: WORKER_COMPILER from mise.toml, and app.wasm must fit the Workers Paid limit.
#                       Deploy, staging, CI and the tests use it.
#   mise run build:dev  the standard Go compiler, always: seconds after an edit. mise run dev uses it; never deployed.
#
# WORKER_COMPILER (mise.toml [env]):
#   tinygo  go tool charter build: TinyGo patched for tinygo#5800, an 8 MB heap, 128 KB stacks, -opt=z, and
#           charter's Worker glue (go.mjs: Go runtimes kept between requests) in build/. An unchanged build comes
#           from charter's cache in under a second. WORKERBUILD_FLAGS adds flags (mise run bench:variants turns
#           each optimisation off in turn: -reuse=false, -patches=false, -heap=0).
#   go      go tool charter build -compiler=go: the standard compiler, seconds per build, a bigger Wasm and a Go runtime
#           per request. mise run dev uses it; never deployed.
set -euo pipefail
profile=${WASM_PROFILE:-prod}
compiler=${WORKER_COMPILER:-tinygo}
[ "$profile" = dev ] && compiler=go
# One build at a time into build/ (build and build:dev share it): go tool charter build holds a lock on it.
rm -f build/.prod build/.dev
case $compiler in
  tinygo) go tool charter build ${WORKERBUILD_FLAGS:-} ;;
  go) go tool charter build -compiler=go ;;
  *) echo "✗ WORKER_COMPILER=$compiler: tinygo or go"; exit 1 ;;
esac
# intl's Worker modules beside the build: intl.mjs, intl-worker.mjs and the catalogs (intl/README.md).
go tool intl assets -worker build
raw=$(wc -c <build/app.wasm | tr -d ' ')
gz=$(gzip -9 -c build/app.wasm | wc -c | tr -d ' ')
if [ "$profile" = prod ]; then
  limit=10000000 # Workers Paid (account checked 2026-09-13): 10 MB compressed; free plan is 3 MB
  [ "$gz" -le "$limit" ] || { echo "✗ app.wasm gzip ${gz}B exceeds ${limit}B"; exit 1; }
  echo "✓ build/app.wasm ($profile, $compiler) raw=${raw}B gzip=${gz}B (limit ${limit}B)"
else
  echo "✓ build/app.wasm ($profile, $compiler, not for deploy) raw=${raw}B gzip=${gz}B"
fi
touch "build/.$profile"
