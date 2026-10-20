#!/usr/bin/env bash
# The Worker's wasm build into build/ (mise run build / build:dev).
#
#   mise run build      production: WORKER_COMPILER from mise.toml, and app.wasm must fit the Workers Paid limit.
#                       Deploy, staging, CI and the tests use it.
#   mise run build:dev  the standard Go compiler, always: seconds after an edit. mise run dev uses it; never deployed.
#
# WORKER_COMPILER (mise.toml [env]):
#   tinygo  -opt=z -no-debug: the smallest wasm (~1 MB gzipped), ~1–2 min per build (TinyGo compiles the whole
#           program and runs wasm-opt --asyncify every time).
#   go      GOOS=js GOARCH=wasm: seconds after an edit (Go caches per package; real goroutines need no asyncify), a
#           bigger wasm (~3 MB gzipped) that each new Worker isolate loads at cold start.
# workers-go generates the matching JS glue (workers-assets-gen -mode tinygo|go); the files are the same.
set -euo pipefail
profile=${WASM_PROFILE:-prod}
compiler=${WORKER_COMPILER:-tinygo}
[ "$profile" = dev ] && compiler=go
rm -f build/.prod build/.dev
case $compiler in
  tinygo)
    go run github.com/syumai/workers-go/cmd/workers-assets-gen -mode=tinygo
    # `.`, not the template's `./...`: keep local-only tools under cmd/ out of the wasm.
    tinygo build -o build/app.wasm -target wasm -opt=z -no-debug .
    ;;
  go)
    go run github.com/syumai/workers-go/cmd/workers-assets-gen -mode=go
    GOOS=js GOARCH=wasm go build -trimpath -ldflags=-s -o build/app.wasm .
    ;;
  *) echo "✗ WORKER_COMPILER=$compiler: tinygo or go"; exit 1 ;;
esac
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
