package pages_test

import (
	"context"
	"slices"
	"testing"

	"github.com/joeblew999/go-htmx4/intl"
	_ "github.com/joeblew999/go-htmx4/locales" // the app's languages, for the locale prefix
	"github.com/joeblew999/go-htmx4/pages"
)

// TestPaths: a page's address carries only what makes it a different page, and its route is the address's path.
func TestPaths(t *testing.T) {
	for _, tc := range []struct {
		page        pages.Page
		path, route string
	}{
		{pages.Home{}, "/", "/"},
		{pages.Board{}, "/board", "/board"},
		{pages.Board{Topic: pages.DefaultTopic}, "/board", "/board"},
		{pages.Board{Topic: "a b&c"}, "/board?topic=a+b%26c", "/board"},
		{pages.About{}, "/about", "/about"},
	} {
		if got := tc.page.Path(); got != tc.path {
			t.Errorf("%#v.Path() = %q, want %q", tc.page, got, tc.path)
		}
		if got := pages.Route(tc.page); got != tc.route {
			t.Errorf("Route(%#v) = %q, want %q", tc.page, got, tc.route)
		}
	}
}

// TestHooks: intl's IsPage takes a path without its locale; seo's takes the request's path and query, locale and all.
// A board topic anyone made is a page, but not one Google may index.
func TestHooks(t *testing.T) {
	for path, want := range map[string]bool{"/": true, "/board": true, "/about": true, "/fragments/stats": false, "/components/sidebar": false, "/greet": false} {
		if got := pages.IsPage(path); got != want {
			t.Errorf("IsPage(%q) = %v, want %v", path, got, want)
		}
	}
	for path, want := range map[string]bool{
		"/": true, "/de/": true, "/board": true, "/board?topic=lobby": true, "/de/board?utm_source=x": true,
		"/board?topic=games": false, "/de/board?topic=games": false, "/fragments/stats": false, "/de/fragments/stats": false,
	} {
		if got := pages.Indexable(path); got != want {
			t.Errorf("Indexable(%q) = %v, want %v", path, got, want)
		}
	}
	if got, want := pages.Sitemap(), []string{"/", "/board", "/formats", "/components", "/about"}; !slices.Equal(got, want) {
		t.Errorf("Sitemap() = %v, want %v", got, want)
	}
}

// TestKey: the canonical page drops what doesn't make a different page; what isn't a page is its path.
func TestKey(t *testing.T) {
	for path, want := range map[string]string{
		"/board?topic=lobby&utm_source=x": "/board",
		"/board?topic=games&x=1":          "/board?topic=games",
		"/about?ref=y":                    "/about",
		"/components/sidebar":             "/components/sidebar",
	} {
		ctx := intl.WithRequest(context.Background(), intl.Request{Locale: intl.Default(), Path: path})
		if got := pages.Key(ctx); got != want {
			t.Errorf("Key(%q) = %q, want %q", path, got, want)
		}
	}
}
