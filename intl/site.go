package intl

import (
	"cmp"
	"net/http"
	"strings"
)

// Site puts each request's language in its context (Handler), by the URL rules a site with languages needs:
//
//	/about        the source locale (intl.toml's first): unprefixed, and hreflang x-default
//	/de/about     every other shipped locale, under its lowercase id ("/pt-br/", "/zh-hant/")
//	/en/about     301 → /about;  /DE/about, /pt-BR/ → 301 to the lowercase prefix;  /de → 301 /de/
//
// There are no redirects from Accept-Language (Google: separate URLs per language and no automatic redirects;
// Googlebot sends none). A page view remembers its locale in a cookie, and only the bare "/" follows it, so a
// returning reader lands on their language's home page. A locale whose catalog is incomplete is served noindex. The
// viewer's time zone and clock come from the cookies Preferences sets, else the connection's zone.
type Site struct {
	// IsPage reports whether path (without its locale prefix) is a page people read; only a page view remembers
	// its locale. nil: every path.
	IsPage func(path string) bool
	// ConnectionTimeZone is the server's guess of the viewer's IANA time zone (Cloudflare's request.cf.timezone).
	// nil: none, "UTC".
	ConnectionTimeZone func(*http.Request) string
	// Origin is the request's scheme and host, for absolute URLs. nil: from the request (and X-Forwarded-Proto).
	Origin func(*http.Request) string
}

// The cookies Site reads and Preferences sets.
const (
	LocaleCookie    = "locale" // the locale of the last page viewed
	TimeZoneCookie  = "tz"     // the viewer's chosen IANA time zone
	HourCycleCookie = "hc"     // the viewer's chosen clock: "h12" or "h23"
)

// Handler resolves the request's locale from its URL prefix, strips the prefix, and puts Request in the context.
func (s Site) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		def := Default()
		path := r.URL.Path
		seg, rest := path, ""
		if len(path) > 1 {
			if i := strings.IndexByte(path[1:], '/'); i >= 0 {
				seg, rest = path[1:i+1], path[i+1:]
			} else {
				seg = path[1:]
			}
		}
		loc := def
		if len(path) > 1 {
			if found := byPrefix(seg); found != nil {
				switch {
				case found == def:
					redirect(w, r, cmp.Or(rest, "/"), http.StatusMovedPermanently)
					return
				case "/"+seg != Prefix(found) || rest == "":
					redirect(w, r, Prefix(found)+cmp.Or(rest, "/"), http.StatusMovedPermanently)
					return
				}
				loc, path = found, rest
			}
		}
		page := r.Method == http.MethodGet && r.Header.Get("HX-Request") == "" && (s.IsPage == nil || s.IsPage(path))
		if path == "/" && loc == def && page {
			// An explicit choice of the source locale from a language list ("/?locale=en"): remember it and redirect
			// to the clean URL, so the parameter URL is never indexed as a duplicate.
			if chosen := byPrefix(r.URL.Query().Get(LocaleCookie)); chosen != nil {
				remember(w, r, chosen)
				http.Redirect(w, r, Prefix(chosen)+"/", http.StatusFound)
				return
			}
			if c, err := r.Cookie(LocaleCookie); err == nil {
				if remembered := byPrefix(c.Value); remembered != nil && remembered != def {
					w.Header().Add("Vary", "Cookie")
					redirect(w, r, Prefix(remembered)+"/", http.StatusFound)
					return
				}
			}
		}
		if page {
			remember(w, r, loc)
		}
		w.Header().Set("Content-Language", loc.ID())
		if !Complete(loc.ID()) {
			// Part of the page would be in the source language: not indexed as this language, nor as a duplicate.
			w.Header().Set("X-Robots-Tag", "noindex")
		}
		req := Request{Locale: loc, Path: withQuery(path, r.URL.RawQuery), Origin: s.origin(r), TimeZone: "UTC"}
		if c, err := r.Cookie(HourCycleCookie); err == nil && (c.Value == "h12" || c.Value == "h23") {
			req.HourCycle = c.Value
		}
		if c, err := r.Cookie(TimeZoneCookie); err == nil && ValidTimeZone(c.Value) {
			req.TimeZone, req.TimeZoneChosen = c.Value, true
		} else {
			req.TimeZone = s.ConnectionZone(r)
		}
		r2 := r.Clone(WithRequest(r.Context(), req))
		r2.URL.Path, r2.URL.RawPath = path, ""
		next.ServeHTTP(w, r2)
	})
}

// Preferences answers the preferences form (POST tz, hc, return): it stores the time zone and clock cookies (an
// empty value clears one, back to automatic) and redirects to the page the form was on, which renders with them.
func (s Site) Preferences(w http.ResponseWriter, r *http.Request) {
	tz, hc := r.FormValue("tz"), r.FormValue("hc")
	if tz != "" && !ValidTimeZone(tz) {
		http.Error(w, "unknown time zone", http.StatusBadRequest)
		return
	}
	if hc != "" && hc != "h12" && hc != "h23" {
		http.Error(w, "unknown hour cycle", http.StatusBadRequest)
		return
	}
	for name, value := range map[string]string{TimeZoneCookie: tz, HourCycleCookie: hc} {
		c := &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: 365 * 24 * 3600, SameSite: http.SameSiteLaxMode, Secure: secure(r)}
		if value == "" {
			c.MaxAge = -1
		}
		http.SetCookie(w, c)
	}
	http.Redirect(w, r, SafeReturn(r.FormValue("return")), http.StatusSeeOther)
}

// ConnectionZone is the server's guess of the viewer's time zone, "UTC" without one: "Automatic (…)" in a
// preferences form.
func (s Site) ConnectionZone(r *http.Request) string {
	if s.ConnectionTimeZone != nil {
		if id := s.ConnectionTimeZone(r); ValidTimeZone(id) {
			return id
		}
	}
	return "UTC"
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
		if secure(r) {
			scheme = "https"
		}
	}
	return scheme + "://" + host
}

// SafeReturn keeps a same-site redirect target: the path and query of an absolute or root-relative URL, else "/"
// (never another host).
func SafeReturn(u string) string {
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

// byPrefix is the shipped locale a URL segment or cookie names, case-insensitively, or nil.
func byPrefix(seg string) *Locale {
	if seg == "" || len(seg) > 16 {
		return nil
	}
	return shipped.Find(seg)
}

func remember(w http.ResponseWriter, r *http.Request, l *Locale) {
	if c, err := r.Cookie(LocaleCookie); err == nil && c.Value == l.ID() {
		return
	}
	http.SetCookie(w, &http.Cookie{Name: LocaleCookie, Value: l.ID(), Path: "/", MaxAge: 365 * 24 * 3600,
		SameSite: http.SameSiteLaxMode, Secure: secure(r)})
}

func secure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func redirect(w http.ResponseWriter, r *http.Request, to string, code int) {
	http.Redirect(w, r, withQuery(to, r.URL.RawQuery), code)
}

func withQuery(path, query string) string {
	if query == "" {
		return path
	}
	return path + "?" + query
}
