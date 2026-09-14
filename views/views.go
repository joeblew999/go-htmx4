// Package views holds the app's pages and fragments: gsx components composed from gsxui (ui/).
// Handlers in package main render them with writeNode.
package views

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

// navVariant highlights the current page's header button (as gsxui's site layout does).
func navVariant(path, href string) string {
	if path == href {
		return "secondary"
	}
	return "ghost"
}
