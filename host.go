package main

import (
	"net/http"
	"strings"
)

// canonicalHost redirects the Worker's *.workers.dev URL to the custom domain (custom-domains plan Phase 3), so
// search engines index one host: canonical, hreflang, og:url, the sitemap and robots.txt are all built from the
// request origin and follow automatically. The domain comes from the APP_DOMAIN text binding (mise.toml [env],
// passed by `mise run deploy`); without it nothing redirects (a renamed template, local workerd, `go run .`).
// Only *.workers.dev hosts redirect, never localhost. Exempt: /healthz (monitoring), /live/* (a WebSocket upgrade
// can't follow a redirect) and non-GET/HEAD requests (a redirected POST loses its body).
func canonicalHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := getenv("APP_DOMAIN")
		host := r.URL.Host
		if host == "" {
			host = r.Host
		}
		host, _, _ = strings.Cut(host, ":")
		if domain == "" || !strings.HasSuffix(strings.ToLower(host), ".workers.dev") ||
			(r.Method != http.MethodGet && r.Method != http.MethodHead) ||
			r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/live/") {
			next.ServeHTTP(w, r)
			return
		}
		target := "https://" + domain + r.URL.EscapedPath()
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}
