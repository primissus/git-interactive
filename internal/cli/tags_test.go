package cli

import (
	"context"
	"testing"
	"time"

	"git-interact/internal/git"
	"git-interact/internal/tui"
)

func TestFilterTagsAuthor(t *testing.T) {
	tags := []git.Tag{
		{Name: "a", Author: "Alice"},
		{Name: "b", Author: "Bob"},
	}
	out := filterTags(tags, tagFilters{author: "ali"})
	if len(out) != 1 || out[0].Name != "a" {
		t.Fatalf("filterTags(author) = %+v, want just 'a'", out)
	}
}

func TestFilterTagsSince(t *testing.T) {
	now := time.Now()
	tags := []git.Tag{
		{Name: "recent", DateUnix: now.Unix()},
		{Name: "old", DateUnix: now.AddDate(0, 0, -10).Unix()},
	}
	out := filterTags(tags, tagFilters{since: "3d"})
	if len(out) != 1 || out[0].Name != "recent" {
		t.Fatalf("filterTags(since=3d) = %+v, want just 'recent'", out)
	}
}

func TestTagFiltersValidate(t *testing.T) {
	if err := (tagFilters{since: "bogus"}).validate(); err == nil {
		t.Errorf("unknown since bucket should error")
	}
	if err := (tagFilters{since: "1w"}).validate(); err != nil {
		t.Errorf("valid since bucket should not error: %v", err)
	}
}

func TestSortTags(t *testing.T) {
	tags := []git.Tag{
		{Name: "b", Author: "Bob", DateUnix: 1},
		{Name: "a", Author: "Alice", DateUnix: 2},
	}

	byAuthor := append([]git.Tag(nil), tags...)
	sortTags(byAuthor, "author")
	if byAuthor[0].Name != "a" {
		t.Errorf("sort by author: got %+v, want 'a' first", byAuthor)
	}

	byCreated := append([]git.Tag(nil), tags...)
	sortTags(byCreated, "created")
	if byCreated[0].Name != "a" {
		t.Errorf("sort by created: got %+v, want 'a' first (higher DateUnix)", byCreated)
	}

	byDefault := append([]git.Tag(nil), tags...)
	sortTags(byDefault, "last-commit")
	if byDefault[0].Name != "a" {
		t.Errorf("sort by last-commit: got %+v, want 'a' first (higher DateUnix)", byDefault)
	}
}

func TestTargetTagAndTargetTags(t *testing.T) {
	create := createTagItem{}
	a := tagItem{t: git.Tag{Name: "a"}}
	b := tagItem{t: git.Tag{Name: "b"}}

	if _, ok := targetTag(nil); ok {
		t.Fatalf("targetTag on empty selection should not resolve")
	}

	if got, ok := targetTag([]tui.Item{a}); !ok || got.t.Name != "a" {
		t.Fatalf("targetTag single item = %+v, %v; want 'a', true", got, ok)
	}
	if _, ok := targetTag([]tui.Item{a, b}); ok {
		t.Fatalf("targetTag with 2 items should not resolve")
	}

	out := targetTags([]tui.Item{create, a, b})
	if len(out) != 2 || out[0].t.Name != "a" || out[1].t.Name != "b" {
		t.Fatalf("targetTags = %+v, want [a, b] (create row dropped)", out)
	}
}

func TestLoadTagItemsIncludesCreateRow(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	commitFile(t, r, "a.txt", "hello\n", "initial commit")
	mustGit(t, r, "tag", "v1.0.0")

	items, err := loadTagItems(ctx, r, tagFilters{}, "", true)
	if err != nil {
		t.Fatalf("loadTagItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("loadTagItems = %d items, want 2 (create row + v1.0.0): %+v", len(items), items)
	}
	if _, ok := items[0].(createTagItem); !ok {
		t.Errorf("items[0] = %T, want createTagItem", items[0])
	}
	tg, ok := items[1].(tagItem)
	if !ok || tg.t.Name != "v1.0.0" {
		t.Errorf("items[1] = %+v, want tagItem{Name: v1.0.0}", items[1])
	}

	// Without the create row (direct-menu / -I mode), only the real tag remains.
	items, err = loadTagItems(ctx, r, tagFilters{}, "", false)
	if err != nil {
		t.Fatalf("loadTagItems: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("loadTagItems(no create row) = %d items, want 1", len(items))
	}
}
