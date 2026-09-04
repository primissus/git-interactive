package tui

import (
	"errors"
	"testing"
)

// searchableRow is a minimal Searchable item: FilterValue is its short label
// (as used by batch-op summaries), SearchValue widens the fuzzy filter to
// extra text FilterValue never carries.
type searchableRow struct {
	label string
	extra string
}

func (r searchableRow) Columns() []string   { return []string{r.label} }
func (r searchableRow) FilterValue() string { return r.label }
func (r searchableRow) Current() bool       { return false }
func (r searchableRow) SearchValue() string { return r.label + " " + r.extra }

func TestFilterItemsPrefersSearchValue(t *testing.T) {
	items := []Item{
		searchableRow{label: "feature/x", extra: "~/src/wt/foo"},
		searchableRow{label: "main", extra: "~/src/wt/bar"},
	}
	got := filterItems("wtfoo", items)
	if len(got) != 1 || items[got[0]].FilterValue() != "feature/x" {
		t.Fatalf("filterItems(SearchValue-only match) = %v, want just the 'feature/x' row", got)
	}
}

func TestFilterItemsFallsBackToFilterValue(t *testing.T) {
	items := []Item{demoRow{name: "feature/login"}, demoRow{name: "main"}}
	got := filterItems("login", items)
	if len(got) != 1 || items[got[0]].FilterValue() != "feature/login" {
		t.Fatalf("filterItems(non-Searchable) = %v, want just 'feature/login'", got)
	}
}

// TestBatchFailureLabelUsesFilterValueNotSearchValue verifies a resilient bulk
// operation's failure summary still labels rows with FilterValue even when the
// item implements Searchable — SearchValue only widens the fuzzy filter, it
// must never leak into the human-readable label batch.go reports on failure.
func TestBatchFailureLabelUsesFilterValueNotSearchValue(t *testing.T) {
	item := searchableRow{label: "feature/x", extra: "should not appear in the label"}
	l := New(Config{Columns: DemoColumns(), Items: []Item{item}})
	op := Operation{
		Name: "fail-op", Scope: ScopeItem,
		Batch: &BatchSpec{
			Verb: "did",
			Step: func(it Item) error { return errors.New("boom") },
		},
	}
	l.startBatch(op, []Item{item})
	if len(l.batch.fails) != 1 {
		t.Fatalf("fails = %v, want 1 entry", l.batch.fails)
	}
	if l.batch.fails[0].label != "feature/x" {
		t.Errorf("batch failure label = %q, want FilterValue %q (not SearchValue)", l.batch.fails[0].label, "feature/x")
	}
}
