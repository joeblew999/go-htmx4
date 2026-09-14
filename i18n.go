package main

import (
	"net/http"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"github.com/joeblew999/go-htmx4/views"
)

// Locale URLs (plan: .plans/2026-09-14_1106_full-i18n.md, decision 2):
//
//	/about        the default locale (cldr.Data.Locales[0], "en"): unprefixed, and hreflang x-default
//	/de/about     every other shipped locale, prefixed with its lowercase id ("/pt-br/", "/zh-hant/")
//	/en/about     301 → /about;  /DE/about, /pt-BR/ → 301 to the lowercase prefix;  /de → 301 /de/
//
// There are no redirects from Accept-Language (Google: separate URLs per language, no automatic redirects,
// and Googlebot sends no Accept-Language). A full page view remembers its locale in a cookie, and only the
// bare "/" follows it, so a returning reader lands on their language's home page.

// localeCookie remembers the locale of the last full page a browser viewed.
const localeCookie = "locale"

// translated lists locales whose UI text is translated (message catalogs, plan Phase 5). Pages in other
// locales show English text with localized formatting, so they carry X-Robots-Tag: noindex: an English page
// under /de/ must not be indexed as German or as a duplicate.
var translated = map[string]bool{"en": true}

// withLocale resolves the request's locale from its URL prefix, strips the prefix, and puts
// i18n.Request in the context for handlers and views.
func withLocale(next http.Handler) http.Handler {
	data := cldr.Data
	def := data.Locales[0]
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		seg, rest := path, ""
		if len(path) > 1 {
			if i := strings.IndexByte(path[1:], '/'); i >= 0 {
				seg, rest = path[1:i+1], path[i+1:]
			} else {
				seg = path[1:]
			}
		}
		ld := def
		if len(path) > 1 {
			if found := localeByPrefix(seg); found != nil {
				canonical := views.LocalePrefix(found)
				switch {
				case found == def:
					redirect(w, r, cmpOr(rest, "/"), http.StatusMovedPermanently)
					return
				case "/"+seg != canonical || rest == "":
					redirect(w, r, canonical+cmpOr(rest, "/"), http.StatusMovedPermanently)
					return
				}
				ld = found
				path = rest
			}
		}
		if path == "/" && ld == def && r.Method == http.MethodGet && !isHTMX(r) {
			// An explicit choice of the default locale from the language list ("/?locale=en"): remember it and
			// redirect to the clean URL (a redirect, so the parameter URL is never indexed as a duplicate).
			if chosen := r.URL.Query().Get(localeCookie); chosen != "" {
				if found := localeByPrefix(chosen); found != nil {
					remember(w, r, found)
					http.Redirect(w, r, views.LocalePrefix(found)+"/", http.StatusFound)
					return
				}
			}
			if c, err := r.Cookie(localeCookie); err == nil {
				if remembered := localeByPrefix(c.Value); remembered != nil && remembered != def {
					w.Header().Add("Vary", "Cookie")
					redirect(w, r, views.LocalePrefix(remembered)+"/", http.StatusFound)
					return
				}
			}
		}
		loc, _ := data.Locale(i18n.MustParseTag(ld.ID))
		if r.Method == http.MethodGet && !isHTMX(r) && isPage(path) {
			remember(w, r, ld)
		}
		w.Header().Set("Content-Language", ld.ID)
		if !translated[ld.ID] {
			w.Header().Set("X-Robots-Tag", "noindex")
		}
		r2 := r.Clone(i18n.WithRequest(r.Context(), i18n.Request{Locale: loc, Path: pathWithQuery(path, r.URL.RawQuery)}))
		r2.URL.Path, r2.URL.RawPath = path, ""
		next.ServeHTTP(w, r2)
	})
}

// localeByPrefix finds a shipped locale by URL segment or cookie value, case-insensitively.
func localeByPrefix(seg string) *i18n.LocaleData {
	if seg == "" || len(seg) > 16 {
		return nil
	}
	for _, ld := range cldr.Data.Locales {
		if strings.EqualFold(ld.ID, seg) {
			return ld
		}
	}
	return nil
}

func remember(w http.ResponseWriter, r *http.Request, ld *i18n.LocaleData) {
	if c, err := r.Cookie(localeCookie); err == nil && c.Value == ld.ID {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: localeCookie, Value: ld.ID, Path: "/", MaxAge: 365 * 24 * 3600,
		SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})
}

// isPage reports whether path is a page (not an API, fragment, asset or health check).
func isPage(path string) bool {
	for _, p := range []string{"/fragments/", "/board/", "/greet", "/healthz", "/static/", "/assets/", "/gsxui/", "/live/"} {
		if strings.HasPrefix(path, p) {
			return false
		}
	}
	return true
}

func isHTMX(r *http.Request) bool { return r.Header.Get("HX-Request") != "" }

func redirect(w http.ResponseWriter, r *http.Request, to string, code int) {
	http.Redirect(w, r, pathWithQuery(to, r.URL.RawQuery), code)
}

func pathWithQuery(path, query string) string {
	if query == "" {
		return path
	}
	return path + "?" + query
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
