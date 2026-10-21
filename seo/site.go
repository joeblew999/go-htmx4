// Package seo is how a site answers search engines: Site, the middleware that serves robots.txt and the sitemap,
// marks what isn't a page noindex and gives every page one address; and SnippetWidth, how wide Google shows a title.
// It is small enough for a Worker. The tools are beside it: seo/audit (the site as Googlebot reads it),
// seo/searchconsole (Google's Search Console API) and go tool seo (README.md).
package seo

import (
	"io"
	"net/http"
	"strings"
)

// Site is how an app answers search engines, as middleware around the rest of it (outside a locale middleware such
// as intl.Site, so it sees the paths crawlers ask for):
//
//   - /robots.txt: one group that lets every crawler read everything, and where the sitemap is;
//   - /sitemap.xml: the app's Sitemap for the request's origin;
//   - X-Robots-Tag: noindex on every response that isn't a page (IsPage) or one of those two files, and on
//     everything under Preview;
//   - one address: a GET of a page or of those files on another host, or over plain http, is redirected (301) to
//     https://Host with the same path and query.
//
// Both files are only at the site's root, as crawlers ask for them: /de/robots.txt goes on to the app.
type Site struct {
	// Host is the canonical host ("app.example.com"). Empty: nothing redirects (a preview, local, a test). A
	// loopback host never redirects.
	Host string
	// Sitemap writes the sitemap for origin ("https://app.example.com"), e.g. intl.Sitemap. nil: no /sitemap.xml.
	Sitemap func(origin string) string
	// IsPage reports whether a GET or HEAD request is for a page people read and Google may index; everything else
	// gets X-Robots-Tag: noindex. nil: every GET and HEAD is a page.
	IsPage func(r *http.Request) bool
	// Preview marks every response noindex: a preview must never compete with production.
	Preview bool
	// ContentSignal is robots.txt's Content-Signal line (Cloudflare's content signals: what crawlers may use the
	// content for), e.g. "search=yes, ai-input=yes, ai-train=yes". Empty: none.
	ContentSignal string
	// Origin is the request's scheme and host ("https://app.example.com"). nil: from the request, and
	// X-Forwarded-Proto behind a proxy.
	Origin func(r *http.Request) string
}

// Handler wraps next.
func (s Site) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		read := r.Method == http.MethodGet || r.Method == http.MethodHead
		robots := read && r.URL.Path == "/robots.txt"
		sitemap := read && r.URL.Path == "/sitemap.xml" && s.Sitemap != nil
		page := read && !robots && !sitemap && (s.IsPage == nil || s.IsPage(r))
		origin := s.origin(r)
		if (page || robots || sitemap) && s.Host != "" && !loopback(r) && origin != "https://"+s.Host {
			target := "https://" + s.Host + r.URL.EscapedPath()
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		if s.Preview || !page && !robots && !sitemap {
			w.Header().Set("X-Robots-Tag", "noindex")
		}
		switch {
		case robots:
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			io.WriteString(w, s.robots(origin))
		case sitemap:
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			io.WriteString(w, s.Sitemap(origin))
		default:
			next.ServeHTTP(w, r)
		}
	})
}

// robots is robots.txt: one * group that allows everything (a named group, say Googlebot, would make that crawler
// ignore *), and the sitemap's absolute URL.
func (s Site) robots(origin string) string {
	body := "User-agent: *\n"
	if s.ContentSignal != "" {
		body += "Content-Signal: " + s.ContentSignal + "\n"
	}
	body += "Allow: /\n"
	if s.Sitemap != nil {
		body += "\nSitemap: " + origin + "/sitemap.xml\n"
	}
	return body
}

func (s Site) origin(r *http.Request) string {
	if s.Origin != nil {
		return s.Origin(r)
	}
	scheme, host := r.URL.Scheme, r.URL.Host
	if host == "" {
		host = r.Host
	}
	if scheme == "" {
		scheme = "http"
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
	}
	return scheme + "://" + host
}

// loopback reports whether the request is for localhost or a loopback address: a local server never redirects. (Plain
// string handling: the net package is thin under TinyGo.)
func loopback(r *http.Request) bool {
	host := r.URL.Host
	if host == "" {
		host = r.Host
	}
	if strings.HasPrefix(host, "[") { // [::1]:8913
		host, _, _ = strings.Cut(host[1:], "]")
	} else if h, _, ok := strings.Cut(host, ":"); ok { // 127.0.0.1:8913
		host = h
	}
	return host == "localhost" || host == "::1" || strings.HasPrefix(host, "127.")
}
