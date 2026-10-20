#!/usr/bin/env bash
# TinyGo build of the Worker into build/ (mise run build / build:dev). TINYGO_PROFILE:
#   prod  -opt=z -no-debug, and app.wasm must fit the Workers Paid limit (deploy, staging, CI, tests)
#   dev   -opt=0: about 20% faster, a much larger wasm, never deployed (mise run dev)
# Most of the time is TinyGo's wasm-opt --asyncify pass (goroutines that wait on JS promises), which every profile
# runs: CPU cores matter more than the profile.
set -euo pipefail
profile=${TINYGO_PROFILE:-prod}
go run github.com/syumai/workers-go/cmd/workers-assets-gen -mode=tinygo
rm -f build/.prod build/.dev
case $profile in
  prod) flags=(-opt=z -no-debug) ;;
  dev) flags=(-opt=0) ;;
  *) echo "✗ TINYGO_PROFILE=$profile: prod or dev"; exit 1 ;;
esac
# `.`, not the template's `./...`: keep local-only tools under cmd/ out of the wasm.
tinygo build -o build/app.wasm -target wasm "${flags[@]}" .
raw=$(wc -c <build/app.wasm | tr -d ' ')
gz=$(gzip -9 -c build/app.wasm | wc -c | tr -d ' ')
if [ "$profile" = prod ]; then
  limit=10000000 # Workers Paid (account checked 2026-09-13): 10 MB compressed; free plan is 3 MB
  [ "$gz" -le "$limit" ] || { echo "✗ app.wasm gzip ${gz}B exceeds ${limit}B"; exit 1; }
  echo "✓ build/app.wasm ($profile) raw=${raw}B gzip=${gz}B (limit ${limit}B)"
else
  echo "✓ build/app.wasm ($profile, not for deploy) raw=${raw}B gzip=${gz}B"
fi
touch "build/.$profile"
