package app

import (
	"strings"

	"github.com/joeblew999/go-htmx4/internal/httpx"
	"github.com/joeblew999/go-htmx4/intl"
)

// site is the app's languages on the web (intl.Site): locale URLs (/about, /de/about), the locale cookie, the
// viewer's time zone and clock, Content-Language and noindex for an incomplete locale. What is the app's own: which
// paths are pages, the connection's time zone (Cloudflare's, platform_js.go), and the origin behind the proxy.
var site = intl.Site{IsPage: isPage, ConnectionTimeZone: connectionTimeZone, Origin: httpx.Origin}

// isPage reports whether path is a page (not an API, fragment, asset or health check).
func isPage(path string) bool {
	for _, p := range []string{"/fragments/", "/board/", "/greet", "/healthz", "/static/", "/assets/", "/gsxui/", "/live/", "/preferences", "/sitemap.xml", "/robots.txt", "/components/"} {
		if strings.HasPrefix(path, p) {
			return false
		}
	}
	return true
}
