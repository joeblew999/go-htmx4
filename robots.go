package main

import (
	"net/http"
	"strings"

	"github.com/joeblew999/go-htmx4/internal/httpx"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
)

// robotsTxt answers /robots.txt. One `*`
// group allows everything: a named group (e.g. Googlebot) would make that crawler ignore `*`, and no Google-Extended
// group means Gemini training and grounding are allowed. Content-Signal isn't read by Google but replaces Cloudflare's
// default notice with the same intent for other crawlers. The sitemap URL uses the request's own origin, so it's
// right locally, on workers.dev, on a custom domain and after `mise run rename` without a configured site URL. It's a
// Go route, not a static file, for that reason (Static Assets has no robots.txt, so the request reaches the Worker).
func robotsTxt(w http.ResponseWriter, r *http.Request) {
	if !allow(w, r, http.MethodGet) {
		return
	}
	// Crawlers only read /robots.txt at the host root; the locale middleware would also route /de/robots.txt here.
	if req, ok := intl.FromContext(r.Context()); ok && req.Locale != locales.Shipped[0] {
		http.NotFound(w, r)
		return
	}
	body := "User-agent: *\n" +
		"Content-Signal: search=yes, ai-input=yes, ai-train=yes\n" +
		"Allow: /\n" +
		"\n" +
		"Sitemap: " + httpx.Origin(r) + "/sitemap.xml\n"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(body))
}

// noindexNonPages marks every response that isn't a page X-Robots-Tag: noindex: fragments
// under /fragments/, the health check, and every request that isn't GET or HEAD (form posts, now and future). One
// rule around the mux instead of per-handler wrapping, so a new fragment or POST route can't forget it. It runs
// inside site.Handler, so /de/fragments/… is matched on its unprefixed path. Pages (every /board topic included),
// /robots.txt and /sitemap.xml stay indexable, except on a Worker Preview (APP_PREVIEW set by `mise run preview`),
// where nothing is: a preview must never compete with production in search results.
func noindexNonPages(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if getenv("APP_PREVIEW") != "" || (r.Method != http.MethodGet && r.Method != http.MethodHead) ||
			strings.HasPrefix(r.URL.Path, "/fragments/") || strings.HasPrefix(r.URL.Path, "/components/") || r.URL.Path == "/healthz" {
			w.Header().Set("X-Robots-Tag", "noindex")
		}
		h.ServeHTTP(w, r)
	})
}
