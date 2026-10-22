// Package pages is the app's page table (#79): every page people read is a Go type here, whose Path is its address
// without the locale prefix. Everything else comes from it: the routes (app/server.go registers each page's Route),
// the links ({ pages.About{} |> url }, intl's url filter), the sitemap (the pages search engines may index), which
// requests are page views (intl.Site's IsPage) and which Google may index (seo.Site's IsPage), and each page's
// canonical address (Key, for intl.Head). A new page is a type with a Path, an entry in All, and its handler.
//
// No reflection: TinyGo builds it as it is.
package pages

import (
	"context"
	"net/url"
	"strings"

	"github.com/joeblew999/go-htmx4/intl"
)

// Page is one of the app's pages.
type Page interface {
	// Path is its address without the locale prefix, with what makes it a different page: "/board?topic=games".
	Path() string
}

// A page with parameters makes its value from a request's query.
type parameterized interface {
	Page
	FromQuery(url.Values) Page
}

// A page that search engines must not index says so.
type unindexed interface {
	Page
	Noindex() bool
}

// All is every page, each with no parameters: the routes, and the sitemap's pages (those search engines may index).
var All = []Page{Home{}, Board{}, Formats{}, Components{}, About{}}

// Home is the home page.
type Home struct{}

func (Home) Path() string { return "/" }

// DefaultTopic is the board's topic without ?topic=: plain /board.
const DefaultTopic = "lobby"

// Board is the shared board of one topic.
type Board struct{ Topic string }

func (b Board) Path() string {
	if b.Topic == "" || b.Topic == DefaultTopic {
		return "/board"
	}
	return "/board?topic=" + url.QueryEscape(b.Topic)
}

func (Board) FromQuery(q url.Values) Page { return Board{Topic: q.Get("topic")} }

// Noindex: any topic but the default is someone's scratch space, and anyone can make another: there is no end to them.
func (b Board) Noindex() bool { return b.Path() != "/board" }

// Formats is what Intl does with dates, numbers and text.
type Formats struct{}

func (Formats) Path() string { return "/formats" }

// Components is every gsxui component, with Intl.
type Components struct{}

func (Components) Path() string { return "/components" }

// About is the stack.
type About struct{}

func (About) Path() string { return "/about" }

// Route is p's path without its query: what the router registers.
func Route(p Page) string {
	path, _, _ := strings.Cut(p.Path(), "?")
	return path
}

// At is the page at path (without the locale prefix) for a request with query, false when path is no page's.
func At(path string, query url.Values) (Page, bool) {
	for _, p := range All {
		if Route(p) == path {
			if pp, ok := p.(parameterized); ok {
				return pp.FromQuery(query), true
			}
			return p, true
		}
	}
	return nil, false
}

// IsPage is intl.Site's IsPage: whether path (without the locale prefix) is a page, so a GET of it is a page view.
func IsPage(path string) bool {
	_, ok := At(path, nil)
	return ok
}

// Indexable is seo.Site's IsPage: whether a request for path ("/de/board?topic=games") is for a page Google may index.
func Indexable(path string) bool {
	path, query, _ := strings.Cut(path, "?")
	q, _ := url.ParseQuery(query)
	p, ok := At(intl.Unprefixed(path), q)
	if u, ok := p.(unindexed); ok && u.Noindex() {
		return false
	}
	return ok
}

// Sitemap is the paths the sitemap lists, in every locale: every page search engines may index.
func Sitemap() []string {
	var out []string
	for _, p := range All {
		if u, ok := p.(unindexed); !ok || !u.Noindex() {
			out = append(out, p.Path())
		}
	}
	return out
}

// Key is the request's page, as its canonical address and its alternates name it (intl.Head): its path without the
// locale prefix or the query parameters that don't make a different page. "/board?topic=lobby&utm_source=x" is
// "/board". What isn't a page is its path.
func Key(ctx context.Context) string {
	path, query := "/", ""
	if r, ok := intl.FromContext(ctx); ok && r.Path != "" {
		path, query, _ = strings.Cut(r.Path, "?")
	}
	q, _ := url.ParseQuery(query)
	if p, ok := At(path, q); ok {
		return p.Path()
	}
	return path
}
