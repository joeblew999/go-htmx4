// Package views holds the app's pages and fragments: gsx components composed from gsxui (ui/).
// Handlers in package app render them.
package views

import (
	"context"
	"maps"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/intl"
)

// DefaultTopic is the board's topic without ?topic= (its canonical URL is plain /board).
const DefaultTopic = "lobby"

// Board is one topic's shared state as the board page and its pushed fragment show it.
type Board struct {
	Topic   string
	Value   int64
	Version int64
	Notes   []Note // newest first, at most maxNotes (package app)
}

// Note is one posted note.
type Note struct {
	ID        int64
	Body      string
	CreatedAt string // SQLite datetime('now'), UTC: "2026-09-14 12:34:56"
}

// Created is when the note was posted.
func (n Note) Created() time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05", n.CreatedAt)
	return t
}

// StackItem is one row on the About page.
type StackItem struct {
	Name, URL string
	Role      intl.Msg // e.g. locales.AboutRolesGsx
}

// ServerInfo is what the home page's dialog shows about the process that rendered it.
type ServerInfo struct {
	GoVersion string
	Uptime    time.Duration
	Requests  int64
	Now       time.Time
	Note      string // platform caveat, e.g. per-request state on Workers
}

// navVariant highlights the current page's header button (as gsxui's site layout does).
func navVariant(path, href string) string { return activeVariant(path == href) }

// activeVariant is a nav button's gsxui variant: secondary for the current page, else ghost.
func activeVariant(active bool) string {
	if active {
		return "secondary"
	}
	return "ghost"
}

func sortedKeys(m map[string]int) []string {
	return slices.Sorted(maps.Keys(m))
}

// pageKey is the request's page for its canonical URL and alternates (intl.Head): without locale prefix and without query parameters that don't make a different page:
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

// homeURL is the home page's absolute URL in the request's language on the home page itself, "" elsewhere: seo.Head
// puts the site's name there for Google (WebSite structured data).
func homeURL(ctx context.Context, path string) string {
	if path != "/" {
		return ""
	}
	return intl.Canonical(ctx, "/")
}
