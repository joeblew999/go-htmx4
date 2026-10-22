// Package views holds the app's pages and fragments: gsx components composed from gsxui (ui/).
// Handlers in package app render them.
package views

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/pages"
)

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
func navVariant(page, nav pages.Page) string {
	return activeVariant(pages.Route(page) == pages.Route(nav))
}

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

// homeURL is the home page's absolute URL in the request's language on the home page itself, "" elsewhere: seo.Head
// puts the site's name there for Google (WebSite structured data).
func homeURL(ctx context.Context, page pages.Page) string {
	if pages.Route(page) != "/" {
		return ""
	}
	return intl.Canonical(ctx, "/")
}
