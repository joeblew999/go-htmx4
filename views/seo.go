package views

import (
	"context"
	"net/url"
	"strings"

	"github.com/joeblew999/go-htmx4/internal/i18n"
	"github.com/joeblew999/go-htmx4/locales"
)

// SEO. Every page links its canonical URL and the same page in every locale (hreflang,
// reciprocal, plus x-default = the default locale), and /sitemap.xml lists every page with the same alternates.
// URLs are absolute, built from the request's origin.

// SitemapPages are the pages /sitemap.xml lists, in every locale. Board topics other than the default are indexable
// but open-ended (anyone can create one), so the sitemap lists only /board.
var SitemapPages = []string{"/", "/board", "/formats", "/about"}

// pageKey is the request's page without locale prefix and without query parameters that don't make a different page:
// only ?topic= on /board survives, and not for the default topic. "/board?topic=lobby&utm_source=x" → "/board".
func pageKey(ctx context.Context) string {
	path, query := "/", ""
	if r, ok := i18n.FromContext(ctx); ok && r.Path != "" {
		path, query, _ = strings.Cut(r.Path, "?")
	}
	if path == "/board" && query != "" {
		if q, err := url.ParseQuery(query); err == nil {
			if topic := q.Get("topic"); topic != "" && topic != DefaultTopic {
				return path + "?topic=" + url.QueryEscape(topic)
			}
		}
	}
	return path
}

func origin(ctx context.Context) string {
	r, _ := i18n.FromContext(ctx)
	return r.Origin
}

// CanonicalURL is the request's page in its own locale, absolute.
func CanonicalURL(ctx context.Context) string {
	return origin(ctx) + LocalePrefix(Loc(ctx)) + pageKey(ctx)
}

// Alternate is one hreflang link.
type Alternate struct {
	Lang string // BCP 47 id, or "x-default"
	Href string
}

// Alternates are the request's page in every shipped locale plus x-default (the default locale's URL), in data order.
// Every locale's version of a page gets the same list, so the links are reciprocal.
func Alternates(ctx context.Context) []Alternate {
	return alternates(origin(ctx), pageKey(ctx))
}

func alternates(origin, page string) []Alternate {
	out := make([]Alternate, 0, len(locales.Shipped)+1)
	for _, ld := range locales.Shipped {
		out = append(out, Alternate{Lang: ld.ID(), Href: origin + LocalePrefix(ld) + page})
	}
	return append(out, Alternate{Lang: "x-default", Href: origin + page})
}

// Sitemap is /sitemap.xml: every SitemapPages page in every locale, each with all its alternates
// (https://developers.google.com/search/docs/specialty/international/localized-versions#sitemap).
func Sitemap(origin string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">` + "\n")
	for _, page := range SitemapPages {
		alts := alternates(origin, page)
		for _, self := range alts[:len(alts)-1] {
			b.WriteString("  <url>\n    <loc>" + xmlEscape(self.Href) + "</loc>\n")
			for _, a := range alts {
				b.WriteString(`    <xhtml:link rel="alternate" hreflang="` + a.Lang + `" href="` + xmlEscape(a.Href) + `"/>` + "\n")
			}
			b.WriteString("  </url>\n")
		}
	}
	b.WriteString("</urlset>\n")
	return b.String()
}

var xmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")

func xmlEscape(s string) string { return xmlEscaper.Replace(s) }

// Open Graph: link previews in chat apps and social sites. og:url is the canonical URL,
// og:title/og:description are the page's translated title and description, og:locale is the page's locale and
// og:locale:alternate every other shipped locale.

// OGLocale is a locale in Open Graph's language_TERRITORY form, from its likely subtags (Intl.Locale maximize):
// "en" (en-Latn-US) → "en_US", "pt-BR" → "pt_BR", "zh-Hant" (zh-Hant-TW) → "zh_TW".
func OGLocale(ld *i18n.Locale) string {
	tags := strings.Split(ld.Maximal(), "-")
	for _, t := range tags[1:] {
		if len(t) == 2 || len(t) == 3 && t[0] >= '0' && t[0] <= '9' {
			return tags[0] + "_" + t
		}
	}
	return tags[0]
}

// OGLocaleAlternates are every shipped locale except the request's, in data order.
func OGLocaleAlternates(ctx context.Context) []string {
	self := Loc(ctx)
	out := make([]string, 0, len(locales.Shipped)-1)
	for _, ld := range locales.Shipped {
		if ld != self {
			out = append(out, OGLocale(ld))
		}
	}
	return out
}
