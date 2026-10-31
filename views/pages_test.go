package views_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/joeblew999/go-htmx4/intl"
	_ "github.com/joeblew999/go-htmx4/locales" // the app's languages, for the locale prefix
	"github.com/joeblew999/go-htmx4/views"
)

// TestPages: All is every page in the tree (Pages' fields), in its order, and each page's address comes from its route.
func TestPages(t *testing.T) {
	// The tree's pages are its types with a Page method; the rest are the routes pages call (routes.go).
	var tree, all []reflect.Type
	for i := range reflect.TypeOf(views.Pages{}).NumField() {
		if t := reflect.TypeOf(views.Pages{}).Field(i).Type; func() bool { _, ok := t.MethodByName("Page"); return ok }() {
			tree = append(tree, t)
		}
	}
	for _, p := range views.All {
		all = append(all, reflect.TypeOf(p))
	}
	if !slices.Equal(tree, all) {
		t.Errorf("All is %v, the tree has %v", all, tree)
	}
	if got, want := views.Sitemap(), []string{"/", "/board", "/formats", "/components", "/live", "/map", "/about"}; !slices.Equal(got, want) {
		t.Errorf("Sitemap() = %v, want %v", got, want)
	}
	for topic, want := range map[string]string{"": "", views.DefaultTopic: "", "a b&c": "?topic=a+b%26c"} {
		if got := (views.Board{Topic: topic}).Query(); got != want {
			t.Errorf("Board{%q}.Query() = %q, want %q", topic, got, want)
		}
	}
}

// TestHooks: intl's IsPage takes a path without its locale; seo's takes the request's path and query, locale and all.
// A board topic anyone made is a page, but not one Google may index.
func TestHooks(t *testing.T) {
	for path, want := range map[string]bool{"/": true, "/board": true, "/about": true, "/fragments/stats": false, "/components/sidebar": false, "/greet": false} {
		if got := views.IsPage(path); got != want {
			t.Errorf("IsPage(%q) = %v, want %v", path, got, want)
		}
	}
	for path, want := range map[string]bool{
		"/": true, "/de/": true, "/board": true, "/board?topic=lobby": true, "/de/board?utm_source=x": true,
		"/board?topic=games": false, "/de/board?topic=games": false, "/fragments/stats": false, "/de/fragments/stats": false,
	} {
		if got := views.Indexable(path); got != want {
			t.Errorf("Indexable(%q) = %v, want %v", path, got, want)
		}
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
		if got := views.Key(ctx); got != want {
			t.Errorf("Key(%q) = %q, want %q", path, got, want)
		}
	}
}
