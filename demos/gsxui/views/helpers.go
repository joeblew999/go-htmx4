package views

import (
	"maps"
	"slices"
)

func sortedKeys(m map[string]int) []string {
	return slices.Sorted(maps.Keys(m))
}
