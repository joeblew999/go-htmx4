package app

import (
	"net/http"
	"strings"

	"github.com/joeblew999/go-htmx4/internal/httpx"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/seo"
	"github.com/joeblew999/go-htmx4/views"
)

// The app on the web: its languages (intl.Site) and how it answers search engines (seo.Site), around every route.
// What is the app's own is here: which paths are pages, which pages the sitemap lists, Cloudflare's guess of the
// viewer's time zone (platform_js.go) and the origin behind the proxy.

// withSite puts the app behind both: search engines first, at the paths crawlers ask for, then the locale.
func withSite(mux http.Handler) http.Handler {
	h := languages.Handler(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { search().Handler(h).ServeHTTP(w, r) })
}

// languages is intl.Site: locale URLs (/about, /de/about), the locale cookie, the viewer's time zone and clock,
// Content-Language, and noindex for a locale whose catalog is incomplete.
var languages = intl.Site{IsPage: isPage, ConnectionTimeZone: connectionTimeZone, Origin: httpx.Origin}

// search is seo.Site, read per request from the Worker's bindings: robots.txt; the sitemap, every page of
// sitemapPages in every locale with its alternates; noindex on whatever isn't a page, and on everything in a preview
// (APP_PREVIEW); one address per page, https on APP_DOMAIN, which only production has.
func search() seo.Site {
	return seo.Site{
		Host:          getenv("APP_DOMAIN"),
		Sitemap:       func(origin string) string { return intl.Sitemap(origin, sitemapPages) },
		IsPage:        indexable,
		Preview:       getenv("APP_PREVIEW") != "",
		ContentSignal: "search=yes, ai-input=yes, ai-train=yes",
		Origin:        httpx.Origin,
	}
}

// sitemapPages are the pages the sitemap lists, in every locale: a new page goes here.
var sitemapPages = []string{"/", "/board", "/formats", "/components", "/about"}

// isPage reports whether path (without its locale prefix) is a page: not an API, fragment, asset or health check.
func isPage(path string) bool {
	for _, p := range []string{"/fragments/", "/board/", "/greet", "/healthz", "/static/", "/assets/", "/gsxui/", "/live/", "/preferences", "/sitemap.xml", "/robots.txt", "/components/"} {
		if strings.HasPrefix(path, p) {
			return false
		}
	}
	return true
}

// indexable reports whether a request is for a page Google may index: a page, and of the board only its default
// topic. Any other topic is someone's scratch space, and anyone can make another: there is no end to them.
func indexable(r *http.Request) bool {
	path := intl.Unprefixed(r.URL.Path)
	if t := r.URL.Query().Get("topic"); path == "/board" && t != "" && t != views.DefaultTopic {
		return false
	}
	return isPage(path)
}
