package main

import (
	"net/http"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/httpx"
	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"github.com/joeblew999/go-htmx4/locales"
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

// Date and time preferences (plan decision 5): the viewer's explicit choice (these cookies, set by POST
// /preferences) > Cloudflare's guess from the connection (request.cf.timezone) > UTC.
const (
	timeZoneCookie  = "tz" // an Intl time zone id
	hourCycleCookie = "hc" // "h12" or "h23"
)

// preferences resolves the viewer's time zone and hour cycle for i18n.Request.
func preferences(r *http.Request) (tz string, chosen bool, hc i18n.HourCycle) {
	if c, err := r.Cookie(hourCycleCookie); err == nil && (c.Value == "h12" || c.Value == "h23") {
		hc, _ = i18n.ParseHourCycle(c.Value)
	}
	if c, err := r.Cookie(timeZoneCookie); err == nil {
		if id, ok := cldr.Data.TimeZone(c.Value); ok {
			return id, true, hc
		}
	}
	if id, ok := cldr.Data.TimeZone(connectionTimeZone(r)); ok {
		return id, false, hc
	}
	return "UTC", false, hc
}

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
		// A locale whose catalog isn't complete (locales/*.toml) shows some English: noindex, so an English page
		// under /de/ isn't indexed as German or as a duplicate.
		if !locales.Complete(ld.ID) {
			w.Header().Set("X-Robots-Tag", "noindex")
		}
		tz, chosen, hc := preferences(r)
		r2 := r.Clone(i18n.WithRequest(r.Context(), i18n.Request{
			Locale: loc, Path: pathWithQuery(path, r.URL.RawQuery), Origin: httpx.Origin(r),
			TimeZone: tz, TimeZoneChosen: chosen, HourCycle: hc,
		}))
		r2.URL.Path, r2.URL.RawPath = path, ""
		next.ServeHTTP(w, r2)
	})
}

// savePreferences answers POST /preferences: it stores the time zone and hour cycle cookies (an empty value
// clears one, back to automatic) and redirects to the page the form was on, which re-renders with them.
func savePreferences(w http.ResponseWriter, r *http.Request) {
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	set := func(name, value string) {
		c := &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: 365 * 24 * 3600, SameSite: http.SameSiteLaxMode, Secure: secure}
		if value == "" {
			c.MaxAge = -1
		}
		http.SetCookie(w, c)
	}
	tz := r.FormValue("tz")
	if tz != "" {
		id, ok := cldr.Data.TimeZone(tz)
		if !ok {
			http.Error(w, "unknown time zone", http.StatusBadRequest)
			return
		}
		tz = id
	}
	hc := r.FormValue("hc")
	if hc != "" && hc != "h12" && hc != "h23" {
		http.Error(w, "unknown hour cycle", http.StatusBadRequest)
		return
	}
	set(timeZoneCookie, tz)
	set(hourCycleCookie, hc)
	http.Redirect(w, r, safeReturn(r.FormValue("return")), http.StatusSeeOther)
}

// safeReturn keeps a same-site redirect target: the path and query of an absolute or root-relative URL,
// else the default locale's home page (never another host).
func safeReturn(u string) string {
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
		if j := strings.IndexByte(u, '/'); j >= 0 {
			u = u[j:]
		} else {
			u = "/"
		}
	}
	if u == "" || u[0] != '/' || strings.HasPrefix(u, "//") || strings.ContainsAny(u, "\\\r\n") {
		return "/"
	}
	return u
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
	for _, p := range []string{"/fragments/", "/board/", "/greet", "/healthz", "/static/", "/assets/", "/gsxui/", "/live/", "/preferences", "/sitemap.xml", "/robots.txt"} {
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
