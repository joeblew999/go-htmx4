// Package views holds the app's pages and fragments: gsx components composed from gsxui (ui/).
// Handlers in package main render them.
package views

import (
	"maps"
	"slices"
)

// Board is one topic's shared state as the board page and its pushed fragment show it.
type Board struct {
	Topic   string
	Value   int64
	Version int64
	Notes   []Note // newest first, at most maxNotes (package main)
}

// Note is one posted note.
type Note struct {
	ID        int64
	Body      string
	CreatedAt string
}

// StackItem is one row on the About page.
type StackItem struct {
	Name, Role, URL string
}

// ServerInfo is what the home page's dialog shows about the process that rendered it.
type ServerInfo struct {
	GoVersion string
	Uptime    string
	Requests  int64
	Now       string
	Note      string // platform caveat, e.g. per-request state on Workers
}

// navVariant highlights the current page's header button (as gsxui's site layout does).
func navVariant(path, href string) string {
	if path == href {
		return "secondary"
	}
	return "ghost"
}

func sortedKeys(m map[string]int) []string {
	return slices.Sorted(maps.Keys(m))
}
