#!/usr/bin/env bash
# The Worker's wasm build into build/ (mise run build / build:dev).
#
#   mise run build      production: WORKER_COMPILER from mise.toml, and app.wasm must fit the Workers Paid limit.
#                       Deploy, staging, CI and the tests use it.
#   mise run build:dev  the standard Go compiler, always: seconds after an edit. mise run dev uses it; never deployed.
#
# WORKER_COMPILER (mise.toml [env]):
#   tinygo  charter's build (go tool workerbuild, github.com/joeblew999/charter/go/lib/cmd/workerbuild): TinyGo with
#           the runtime patch for tinygo#5800, an 8 MB heap, 128 KB stacks, -opt=z, and charter's Worker glue
#           (go.mjs: Go runtimes kept between requests) copied into build/. ~1–3 min for a new build; an unchanged
#           one comes from charter's cache in under a second.
#   go      GOOS=js GOARCH=wasm: seconds after an edit (Go caches per package; real goroutines need no asyncify), a
#           bigger wasm (~3 MB gzipped). charter's go.mjs is TinyGo's, so build/go.mjs here is a stand-in with
#           its interface over workers-go's own entry: a Go runtime per request, as before.
set -euo pipefail
profile=${WASM_PROFILE:-prod}
compiler=${WORKER_COMPILER:-tinygo}
[ "$profile" = dev ] && compiler=go
rm -f build/.prod build/.dev
case $compiler in
  tinygo)
    go tool workerbuild
    ;;
  go)
    go run github.com/syumai/workers-go/cmd/workers-assets-gen -mode=go
    GOOS=js GOARCH=wasm go build -trimpath -ldflags=-s -o build/app.wasm .
    # The modules worker/index.mjs and workerd/config.capnp name, then go.mjs as the stand-in.
    lib=$(go list -m -f '{{.Dir}}' github.com/joeblew999/charter/go/lib)
    cp "$lib"/worker/*.mjs build/ && chmod u+w build/*.mjs
    cat >build/go.mjs <<'JS'
// Stand-in for charter's go.mjs in the standard-Go dev build (ci/wasm-build.sh): its interface over
// workers-go's own entry, which starts a Go runtime per request.
import worker from "./worker.mjs";
export function goWorker() {
	return { fetch: (request, env, ctx) => worker.fetch(request, env, ctx), warm: async () => {} };
}
JS
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
