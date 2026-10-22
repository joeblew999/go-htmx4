package app

import (
	"net/http"

	"github.com/joeblew999/charter/go/lib/bindings"
	"github.com/joeblew999/charter/go/lib/web"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/pages"
	"github.com/joeblew999/go-htmx4/seo"
)

// The app on the web: its languages (intl.Site) and how it answers search engines (seo.Site), around every route.
// Which paths are pages, which of them Google may index and what the sitemap lists all come from the page table
// (package pages); the rest is the app's own: Cloudflare's guess of the viewer's time zone (platform.go) and the
// origin behind the proxy.

// withSite puts the app behind both: search engines first, at the paths crawlers ask for, then the locale.
func withSite(mux http.Handler) http.Handler {
	h := languages.Handler(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { search().Handler(h).ServeHTTP(w, r) })
}

// languages is intl.Site: locale URLs (/about, /de/about), the locale cookie, the viewer's time zone and clock,
// Content-Language, and noindex for a locale whose catalog is incomplete.
var languages = intl.Site{IsPage: pages.IsPage, ConnectionTimeZone: connectionTimeZone, Origin: web.Origin}

// search is seo.Site, read per request from the Worker's bindings: robots.txt; the sitemap, every page search engines
// may index in every locale with its alternates; noindex on whatever isn't such a page, and on everything in a
// preview (APP_PREVIEW); one address per page, https on APP_DOMAIN, which only production has.
func search() seo.Site {
	return seo.Site{
		Host:          bindings.Var("APP_DOMAIN"),
		Sitemap:       func(origin string) string { return intl.Sitemap(origin, pages.Sitemap()) },
		IsPage:        pages.Indexable,
		Preview:       bindings.Var("APP_PREVIEW") != "",
		ContentSignal: "search=yes, ai-input=yes, ai-train=yes",
		Origin:        web.Origin,
	}
}
