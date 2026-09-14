// Package ratelimit calls a Cloudflare Workers Rate Limiting binding from Go (TinyGo on workers-go).
//
// Docs: https://developers.cloudflare.com/workers/runtime-apis/bindings/rate-limit/. The binding is a simple
// fixed window per key (limit per 10 or 60 seconds), local to each Cloudflare location and eventually
// consistent: good for stopping abuse, not for accounting. Configure it at deploy time (kit/cfdeploy
// Config.RateLimits). Locally, plain workerd has no such binding: the go-htmx4 app's workerd/local-entry.mjs
// provides an in-memory stand-in with the same API.
package ratelimit

import (
	"errors"
	"net/http"
)

// ErrNoBinding means the named binding isn't configured. Allow still returns true (fail open), so a missing
// binding never takes the app down; log it.
var ErrNoBinding = errors.New("ratelimit: binding not configured")

// ClientKey is the key for per-connection limits: the client IP Cloudflare puts in CF-Connecting-IP, or
// "local" when there is none (local workerd, `go run .`).
func ClientKey(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	return "local"
}
