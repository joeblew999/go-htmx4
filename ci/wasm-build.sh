#!/usr/bin/env bash
# The Worker's wasm build into build/ (mise run build / build:dev). WASM_PROFILE:
#   prod  TinyGo -opt=z -no-debug: a small wasm (~1 MB gzipped) that must fit the Workers Paid limit. Deploy, staging,
#         CI and the tests use it. ~1–2 min: TinyGo compiles the whole program and runs wasm-opt --asyncify every time.
#   dev   the standard Go compiler (GOOS=js GOARCH=wasm): seconds after an edit (Go caches per package, and real
#         goroutines need no asyncify), a larger wasm (~3 MB gzipped). mise run dev uses it; never deployed.
# workers-go generates the matching JS glue for each (workers-assets-gen -mode tinygo|go); the files are the same.
set -euo pipefail
profile=${WASM_PROFILE:-prod}
rm -f build/.prod build/.dev
case $profile in
  prod)
    go run github.com/syumai/workers-go/cmd/workers-assets-gen -mode=tinygo
    # `.`, not the template's `./...`: keep local-only tools under cmd/ out of the wasm.
    tinygo build -o build/app.wasm -target wasm -opt=z -no-debug .
    ;;
  dev)
    go run github.com/syumai/workers-go/cmd/workers-assets-gen -mode=go
    GOOS=js GOARCH=wasm go build -o build/app.wasm .
    ;;
  *) echo "✗ WASM_PROFILE=$profile: prod or dev"; exit 1 ;;
esac
raw=$(wc -c <build/app.wasm | tr -d ' ')
gz=$(gzip -9 -c build/app.wasm | wc -c | tr -d ' ')
if [ "$profile" = prod ]; then
  limit=10000000 # Workers Paid (account checked 2026-09-13): 10 MB compressed; free plan is 3 MB
  [ "$gz" -le "$limit" ] || { echo "✗ app.wasm gzip ${gz}B exceeds ${limit}B"; exit 1; }
  echo "✓ build/app.wasm ($profile, TinyGo) raw=${raw}B gzip=${gz}B (limit ${limit}B)"
else
  echo "✓ build/app.wasm ($profile, standard Go, not for deploy) raw=${raw}B gzip=${gz}B"
fi
touch "build/.$profile"
