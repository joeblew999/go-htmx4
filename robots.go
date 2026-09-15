package main

import (
	"net/http"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/httpx"
	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
)

// robotsTxt answers /robots.txt (search plan: .plans/done/2026-09-14_0938_search-indexing-google-gemini.md). One `*`
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
	if req, ok := i18n.FromContext(r.Context()); ok && req.Locale.Data != cldr.Data.Locales[0] {
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

// noindexNonPages marks every response that isn't a page X-Robots-Tag: noindex (search plan Phase 1): fragments
// under /fragments/, the health check, and every request that isn't GET or HEAD (form posts, now and future). One
// rule around the mux instead of per-handler wrapping, so a new fragment or POST route can't forget it. It runs
// inside withLocale, so /de/fragments/… is matched on its unprefixed path. Pages (every /board topic included),
// /robots.txt and /sitemap.xml stay indexable.
func noindexNonPages(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method != http.MethodGet && r.Method != http.MethodHead) || strings.HasPrefix(r.URL.Path, "/fragments/") || r.URL.Path == "/healthz" {
			w.Header().Set("X-Robots-Tag", "noindex")
		}
		h.ServeHTTP(w, r)
	})
}
