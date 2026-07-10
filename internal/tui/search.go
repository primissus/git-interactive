package tui

import (
	"strings"

	"github.com/sahilm/fuzzy"
)

// filterItems returns the indices of items whose FilterValue fuzzy-matches
// query, ordered by match quality. An empty (or whitespace) query returns every
// index in original order, so the caller can treat "no search" and "search that
// matches everything" identically.
func filterItems(query string, items []Item) []int {
	q := strings.TrimSpace(query)
	if q == "" {
		idx := make([]int, len(items))
		for i := range items {
			idx[i] = i
		}
		return idx
	}

	values := make([]string, len(items))
	for i, it := range items {
		values[i] = it.FilterValue()
	}

	matches := fuzzy.Find(q, values)
	out := make([]int, len(matches))
	for i, m := range matches {
		out[i] = m.Index
	}
	return out
}
