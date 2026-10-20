package classmerge

import (
	"testing"

	"github.com/joeblew999/go-htmx4/ui/merge"
)

func TestSameAsGsxui(t *testing.T) {
	for _, c := range [][]string{
		{"px-2 py-1", "px-4"},
		{"text-sm text-red-500", "text-lg"},
		{"bg-primary hover:bg-primary/90", ""},
		{"px-2", "px-4"}, // again: from the cache
	} {
		if got, want := Merge(c), merge.Merge(c); got != want {
			t.Errorf("Merge(%q) = %q, gsxui's merger says %q", c, got, want)
		}
	}
}
