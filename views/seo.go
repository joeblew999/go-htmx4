package views

import (
	"context"
	"net/url"
	"strings"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
)

// SEO. Every page links its canonical URL and the same page in every locale (intl.Head: hreflang, reciprocal, plus
// x-default = the source locale), and /sitemap.xml lists every page with the same alternates (intl.Sitemap). This
// file is what is the app's own: which pages there are, and what makes a page different.

// SitemapPages are the pages /sitemap.xml lists, in every locale. Board topics other than the default are indexable
// but open-ended (anyone can create one), so the sitemap lists only /board.
var SitemapPages = []string{"/", "/board", "/formats", "/about"}

// pageTitle is a page's <title>: its title, then the app's name (a translated title is the message's argument).
func pageTitle(ctx context.Context, title intl.Msg) intl.Msg {
	return locales.PageTitle(intl.Text(ctx, title))
}

// pageKey is the request's page without locale prefix and without query parameters that don't make a different page:
// only ?topic= on /board survives, and not for the default topic. "/board?topic=lobby&utm_source=x" → "/board".
func pageKey(ctx context.Context) string {
	path, query := "/", ""
	if r, ok := intl.FromContext(ctx); ok && r.Path != "" {
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
