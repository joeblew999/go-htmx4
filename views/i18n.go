package views

import (
	"context"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
)

// Loc is the request's locale (set by the app's locale middleware), or the default locale when a view is
// rendered without one (tests, fragments rendered outside a request).
func Loc(ctx context.Context) *i18n.Locale {
	if r, ok := i18n.FromContext(ctx); ok {
		return r.Locale
	}
	return cldr.Data.MustLocale(cldr.Data.Locales[0].ID)
}

// LocalePrefix is a locale's URL prefix: "" for the default locale, "/de", "/pt-br", "/zh-hant" otherwise.
func LocalePrefix(ld *i18n.LocaleData) string {
	if ld == cldr.Data.Locales[0] {
		return ""
	}
	return "/" + strings.ToLower(ld.ID)
}

// LocaleKey is a locale's id as URL prefixes, the Room's socket tags and live.Localized use it: "de", "pt-br".
func LocaleKey(ld *i18n.LocaleData) string { return strings.ToLower(ld.ID) }

// URL localizes an in-app path for the request's locale: URL(ctx, "/about") is "/about" in English and
// "/de/about" in German. Every in-app href and hx-* URL goes through it; /static/, /assets/, /gsxui/ and
// /live/ are not locale-specific and must not.
func URL(ctx context.Context, path string) string {
	return LocalePrefix(Loc(ctx).Data) + path
}

// LocaleLink is the current page in one shipped locale, for the language list.
type LocaleLink struct {
	Lang    string // BCP 47, for hreflang and lang
	Name    string // the locale's name in its own language
	Href    string
	Current bool
}

// LocaleLinks lists the current page in every shipped locale, in data order.
func LocaleLinks(ctx context.Context) []LocaleLink {
	path := "/"
	if r, ok := i18n.FromContext(ctx); ok && r.Path != "" {
		path = r.Path
	}
	cur := Loc(ctx).Data
	out := make([]LocaleLink, 0, len(cldr.Data.Locales))
	for _, ld := range cldr.Data.Locales {
		href := LocalePrefix(ld) + path
		if href == "/" {
			// The bare "/" follows a remembered locale; choosing the default locale must override it.
			href = "/?locale=" + ld.ID
		}
		out = append(out, LocaleLink{Lang: ld.ID, Name: ld.NativeName, Href: href, Current: ld == cur})
	}
	return out
}

// LocaleLinkHref is the current page's own href in its locale, to highlight it in the language list.
func LocaleLinkHref(ctx context.Context) string {
	for _, l := range LocaleLinks(ctx) {
		if l.Current {
			return l.Href
		}
	}
	return ""
}

// now is the clock relative times are rendered against (a variable so tests can fix it).
var now = time.Now

// noteTime parses a note's created_at ("2006-01-02 15:04:05", SQLite datetime('now'), UTC).
func noteTime(createdAt string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02 15:04:05", createdAt)
	return t, err == nil
}

// NoteISO is a note's timestamp for <time datetime>, e.g. "2026-09-14T12:34:56Z".
func NoteISO(createdAt string) string {
	if t, ok := noteTime(createdAt); ok {
		return t.UTC().Format(time.RFC3339)
	}
	return createdAt
}

// RelativeSince renders a timestamp relative to now in the request's locale ("3 minutes ago", "yesterday"),
// with the same unit choice as static/relative-time.js, which keeps it fresh in the browser.
func RelativeSince(ctx context.Context, createdAt string) string {
	t, ok := noteTime(createdAt)
	if !ok {
		return createdAt
	}
	v, unit := i18n.ElapsedUnit(t.Sub(now()).Seconds())
	s, err := Loc(ctx).RelativeTimeFormat(i18n.RelativeTimeOptions{NumericAuto: true}).Format(v, unit)
	if err != nil {
		return createdAt
	}
	return s
}
