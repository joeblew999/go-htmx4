package main

import (
	"net/http"

	"github.com/joeblew999/go-htmx4/kit/httpx"
)

// robotsTxt answers /robots.txt (search plan: .plans/2026-09-14_0938_search-indexing-google-gemini.md). One `*`
// group allows everything: a named group (e.g. Googlebot) would make that crawler ignore `*`, and no Google-Extended
// group means Gemini training and grounding are allowed. Content-Signal isn't read by Google but replaces Cloudflare's
// default notice with the same intent for other crawlers. The sitemap URL uses the request's own origin, so it's
// right locally, on workers.dev, on a custom domain and after `mise run rename` without a configured site URL. It's a
// Go route, not a static file, for that reason (Static Assets has no robots.txt, so the request reaches the Worker).
func robotsTxt(w http.ResponseWriter, r *http.Request) {
	if !allow(w, r, http.MethodGet) {
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
