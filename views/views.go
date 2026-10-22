// Package views holds the app's pages and fragments: gsx components composed from gsxui (ui/).
// Handlers in package app render them.
package views

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/gsxhq/gsx"
	"github.com/jackielii/structpages"
	"github.com/joeblew999/go-htmx4/intl"
)

// BoardState is one topic's shared state as the board page and its pushed fragment show it.
type BoardState struct {
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

// navVariant highlights the current page's header button (as gsxui's site layout does): structpages knows which
// page is being rendered.
func navVariant(ctx context.Context, nav any) (string, error) {
	in, err := structpages.Within(ctx, nav)
	return activeVariant(in), err
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

// htmxConfig is htmx's settings (the layout's htmx-config meta). hx-ws spreads reconnects over 1–3 s. With the request's
// CSP nonce (app/csp.go), htmx gives it to the scripts it swaps in, which the page's policy otherwise blocks.
func htmxConfig(ctx context.Context) string {
	config := "ws.reconnectDelay:2s ws.reconnectJitter:0.5"
	if nonce := gsx.NonceFromContext(ctx); nonce != "" {
		config += " inlineScriptNonce:" + nonce
	}
	return config
}

// homeURL is the home page's absolute URL in the request's language on the home page itself, "" elsewhere: seo.Head
// puts the site's name there for Google (WebSite structured data).
func homeURL(ctx context.Context) (string, error) {
	if home, err := structpages.Within(ctx, Home{}); !home || err != nil {
		return "", err
	}
	return intl.Canonical(ctx, "/"), nil
}
