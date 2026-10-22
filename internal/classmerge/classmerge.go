// Package classmerge is gsx's class merger for this app (gsx.toml's class_merger): gsxui's Tailwind
// merger (ui/merge, a gsxui file we don't edit), with each result kept. The same classes always merge
// the same way, and a page merges the same few combinations on every button, so a page costs a map
// lookup per merge instead of a Tailwind parse. Measured on the home page: a quarter of what it
// allocated was merging. tailwind-merge-go keeps an LRU cache of its own, and this one is still worth having: on
// three of the app's merges, 87 ns and 1 allocation each against the LRU's 141 ns and 4 (go test -bench, Apple M-series,
// 2026-10-22).
package classmerge

import (
	"strings"
	"sync"

	"github.com/joeblew999/go-htmx4/ui/merge"
)

// merged holds every combination seen: the components' classes, a fixed set (no user input reaches
// a class attribute), so it stops growing after the first pages.
var merged sync.Map

// Merge is gsx's func([]string) string merger seam.
func Merge(classes []string) string {
	key := strings.Join(classes, "\x00")
	if v, ok := merged.Load(key); ok {
		return v.(string)
	}
	v := merge.Merge(classes)
	merged.Store(key, v)
	return v
}
