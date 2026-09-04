package cli

import (
	"testing"

	"git-interact/internal/gh"
	"git-interact/internal/git"
	"git-interact/internal/tui"
)

func mkBranch(name string) branchItem {
	return branchItem{b: git.Branch{Name: name}}
}

// itemsFromBranches turns []branchItem into []tui.Item.
func itemsFromBranches(bs []branchItem) []tui.Item {
	items := make([]tui.Item, len(bs))
	for i, b := range bs {
		items[i] = b
	}
	return items
}

// names extracts the first column from each item.
func names(items []tui.Item) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Columns()[0])
	}
	return out
}

// assertDisplay checks that the visible display names match want.
func assertDisplay(t *testing.T, items []tui.Item, want []string) {
	t.Helper()
	got := names(items)
	if len(got) != len(want) {
		t.Fatalf("names = %d items, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("names[%d] = %q, want %q\nall: %v", i, got[i], w, got)
		}
	}
}

func TestApplyGroupingFlat(t *testing.T) {
	branches := []branchItem{
		mkBranch("main"),
		mkBranch("develop"),
		mkBranch("feature"),
	}
	items := itemsFromBranches(branches)
	grouped := applyGrouping(items, nil)

	assertDisplay(t, grouped, []string{
		"main",
		"develop",
		"feature",
	})
}

func TestApplyGroupingTree(t *testing.T) {
	branches := []branchItem{
		mkBranch("main"),
		mkBranch("feature/api"),
		mkBranch("feature/login"),
		mkBranch("feature/api/v2"),
		mkBranch("feature/api/config"),
		mkBranch("bugfix/urgent"),
	}
	items := itemsFromBranches(branches)
	grouped := applyGrouping(items, nil)

	assertDisplay(t, grouped, []string{
		"▾ bugfix/ (1)",
		"  urgent",
		"▾ feature/ (4)",
		"  ▾ api/ (2)",
		"    v2",
		"    config",
		"  api",
		"  login",
		"main",
	})
}

func TestApplyGroupingCollapsed(t *testing.T) {
	branches := []branchItem{
		mkBranch("feature/api"),
		mkBranch("feature/api/v2"),
		mkBranch("feature/login"),
	}
	items := itemsFromBranches(branches)
	collapsed := map[string]bool{"feature/api/": true}
	grouped := applyGrouping(items, collapsed)

	assertDisplay(t, grouped, []string{
		"▾ feature/ (3)",
		"  ▸ api/ (1)",
		"  api",
		"  login",
	})
}

func TestApplyGroupingBranchIsAlsoDir(t *testing.T) {
	branches := []branchItem{
		mkBranch("feature"),
		mkBranch("feature/api"),
		mkBranch("feature/api/v2"),
	}
	items := itemsFromBranches(branches)
	grouped := applyGrouping(items, nil)

	assertDisplay(t, grouped, []string{
		"▾ feature/ (2)",
		"  ▾ api/ (1)",
		"    v2",
		"  api",
		"feature",
	})
}

func TestApplyGroupingDirsAlphaSorted(t *testing.T) {
	branches := []branchItem{
		mkBranch("z/one"),
		mkBranch("a/two"),
		mkBranch("m/three"),
	}
	items := itemsFromBranches(branches)
	grouped := applyGrouping(items, nil)

	// Dirs should appear alphabetically: a/, m/, z/
	assertDisplay(t, grouped, []string{
		"▾ a/ (1)",
		"  two",
		"▾ m/ (1)",
		"  three",
		"▾ z/ (1)",
		"  one",
	})
}

func TestApplyGroupingBranchesKeepSortOrder(t *testing.T) {
	// Branches in a specific order should keep that order within their group.
	branches := []branchItem{
		mkBranch("feature/zzz"),
		mkBranch("feature/aaa"),
		mkBranch("feature/bbb"),
	}
	items := itemsFromBranches(branches)
	grouped := applyGrouping(items, nil)

	// After the group header ("▾ feature/ (3)"), branches should be in
	// original order: zzz, aaa, bbb (indented).
	assertDisplay(t, grouped, []string{
		"▾ feature/ (3)",
		"  zzz",
		"  aaa",
		"  bbb",
	})
}

func TestApplyGroupingWithCreateRow(t *testing.T) {
	branches := []branchItem{
		mkBranch("main"),
		mkBranch("feature/x"),
	}
	items := []tui.Item{createBranchItem{}}
	for _, b := range branches {
		items = append(items, b)
	}
	grouped := applyGrouping(items, nil)

	assertDisplay(t, grouped, []string{
		"+ new branch (Shift+N)",
		"▾ feature/ (1)",
		"  x",
		"main",
	})
}

func TestApplyGroupingEmpty(t *testing.T) {
	grouped := applyGrouping(nil, nil)
	if len(grouped) != 0 {
		t.Fatalf("empty input should produce empty output, got %v", names(grouped))
	}
}

func TestApplyGroupingNoBranches(t *testing.T) {
	// Only create row, no branches.
	grouped := applyGrouping([]tui.Item{createBranchItem{}}, nil)
	assertDisplay(t, grouped, []string{"+ new branch (Shift+N)"})
}

func TestBranchGroupItemDefaultOp(t *testing.T) {
	g := branchGroupItem{}
	if g.DefaultOp() != "toggle group" {
		t.Fatalf("DefaultOp = %q, want %q", g.DefaultOp(), "toggle group")
	}
}

func TestBranchGroupItemFilterValue(t *testing.T) {
	g := branchGroupItem{label: "test/", depth: 1}
	if g.FilterValue() != "" {
		t.Fatalf("FilterValue should be empty for groups, got %q", g.FilterValue())
	}
	if g.Current() {
		t.Fatalf("Current should be false for groups")
	}
}

func TestLeafName(t *testing.T) {
	tests := []struct{ full, leaf string }{
		{"main", "main"},
		{"feature/x", "x"},
		{"a/b/c", "c"},
	}
	for _, tc := range tests {
		if got := leafName(tc.full); got != tc.leaf {
			t.Errorf("leafName(%q) = %q, want %q", tc.full, got, tc.leaf)
		}
	}
}

func TestNextSortMode(t *testing.T) {
	order := []string{"last-commit", "created", "author", "off"}
	for i, m := range order {
		next := nextSortMode(m)
		want := order[(i+1)%len(order)]
		if next != want {
			t.Errorf("nextSortMode(%q) = %q, want %q", m, next, want)
		}
	}
	if nextSortMode("unknown") != "last-commit" {
		t.Errorf("unknown should default to last-commit")
	}
}

func TestBranchViewStateSortLabel(t *testing.T) {
	s := branchViewState{sort: "created"}
	if s.sortLabel() != "created" {
		t.Errorf("sortLabel = %q, want %q", s.sortLabel(), "created")
	}
	s.grouped = true
	if s.sortLabel() != "created · tree" {
		t.Errorf("grouped sortLabel = %q, want %q", s.sortLabel(), "created · tree")
	}
}

func TestBranchGroupItemCollapsedMarker(t *testing.T) {
	g := branchGroupItem{label: "x/", collapsed: false, depth: 0}
	if g.Columns()[0] != "▾ x/ (0)" {
		t.Errorf("expanded = %q, want %q", g.Columns()[0], "▾ x/ (0)")
	}
	g.collapsed = true
	if g.Columns()[0] != "▸ x/ (0)" {
		t.Errorf("collapsed = %q, want %q", g.Columns()[0], "▸ x/ (0)")
	}
	g.depth = 2
	if g.Columns()[0] != "    ▸ x/ (0)" {
		t.Errorf("deep collapsed = %q, want %q", g.Columns()[0], "    ▸ x/ (0)")
	}
}

func TestBranchGroupItemColumnsPadding(t *testing.T) {
	g := branchGroupItem{label: "test/", count: 5, depth: 0}
	cols := g.Columns()
	if len(cols) != 6 {
		t.Fatalf("Columns should return 6 entries (matching the 6-column branch table), got %d", len(cols))
	}
	for i := 1; i < 6; i++ {
		if cols[i] != "" {
			t.Errorf("cols[%d] = %q, want empty", i, cols[i])
		}
	}
}

// TestApplyGroupingPreservesWorktreeAndPR is the regression test for the p15
// bug where grouping rebuilt each leaf branchItem as branchItem{b, merged},
// silently dropping wtPath/cwd/pr and blanking those columns whenever tree
// view was on.
func TestApplyGroupingPreservesWorktreeAndPR(t *testing.T) {
	b := branchItem{b: git.Branch{Name: "feature/x"}, wtPath: "/home/u/wt", cwd: "/home/u", pr: gh.PR{Number: 7}}
	items := itemsFromBranches([]branchItem{b})
	grouped := applyGrouping(items, nil)

	if len(grouped) != 2 { // group header + leaf
		t.Fatalf("grouped = %d items, want 2", len(grouped))
	}
	leaf, ok := grouped[1].(branchItem)
	if !ok {
		t.Fatalf("grouped[1] = %T, want branchItem", grouped[1])
	}
	if leaf.wtPath != "/home/u/wt" || leaf.pr.Number != 7 {
		t.Errorf("grouped leaf lost wtPath/pr: %+v", leaf)
	}
}

func TestSortModesCycle(t *testing.T) {
	modes := sortModes
	if len(modes) != 4 {
		t.Fatalf("expected 4 sort modes, got %d", len(modes))
	}
	seen := map[string]bool{}
	for _, m := range modes {
		if seen[m] {
			t.Fatalf("duplicate sort mode %q", m)
		}
		seen[m] = true
	}
}

func TestTreeNodeCount(t *testing.T) {
	root := &treeNode{}
	root.insert("a/b/c", mkBranch("a/b/c"))
	root.insert("a/b/d", mkBranch("a/b/d"))
	root.insert("a/e", mkBranch("a/e"))
	root.insert("f", mkBranch("f"))

	if root.count != 4 {
		t.Errorf("root count = %d, want 4", root.count)
	}
}

func TestDisplayNameNotSetWithoutGrouping(t *testing.T) {
	b := mkBranch("feature/x")
	if b.displayName != "" {
		t.Errorf("displayName should be empty when not grouped, got %q", b.displayName)
	}
	cols := b.Columns()
	if cols[0] != "feature/x" {
		t.Errorf("Columns[0] = %q, want %q", cols[0], "feature/x")
	}
}

func TestDisplayNameSetInGroupedMode(t *testing.T) {
	bi := mkBranch("feature/api/v2")
	bi.displayName = "    v2"
	cols := bi.Columns()
	if cols[0] != "    v2" {
		t.Errorf("Columns[0] = %q, want %q", cols[0], "    v2")
	}
	if bi.FilterValue() != "feature/api/v2" {
		t.Errorf("FilterValue = %q, want %q", bi.FilterValue(), "feature/api/v2")
	}
}

func TestBranchItemCurrent(t *testing.T) {
	b := branchItem{b: git.Branch{Name: "main", Head: true}}
	if !b.Current() {
		t.Errorf("Current should be true when Head is true")
	}
	b.b.Head = false
	if b.Current() {
		t.Errorf("Current should be false when Head is false")
	}
}

// testCollapsedTreeDepth traverses the flattened output to verify that
// collapsed groups produce a branchGroupItem with the collapsed field set.
func TestCollapsedFlagInGroup(t *testing.T) {
	branches := []branchItem{
		mkBranch("a/b/c"),
	}
	items := itemsFromBranches(branches)
	collapsed := map[string]bool{"a/": true}
	grouped := applyGrouping(items, collapsed)

	if len(grouped) != 1 {
		t.Fatalf("expected 1 item (collapsed group), got %d", len(grouped))
	}
	g, ok := grouped[0].(branchGroupItem)
	if !ok {
		t.Fatalf("expected branchGroupItem, got %T", grouped[0])
	}
	if !g.collapsed {
		t.Errorf("group should be collapsed")
	}
}

func TestApplyGroupingWithCreateRowOnly(t *testing.T) {
	items := []tui.Item{createBranchItem{}}
	grouped := applyGrouping(items, nil)
	assertDisplay(t, grouped, []string{"+ new branch (Shift+N)"})
}

func TestTreeNodeSortStable(t *testing.T) {
	root := &treeNode{}
	names := []string{"c/z", "a/z", "b/z", "c/y", "a/y"}
	for _, n := range names {
		root.insert(n, mkBranch(n))
	}
	root.sortNode()

	// Verify alphabetical: a/, b/, c/
	if len(root.dirs) != 3 {
		t.Fatalf("expected 3 dirs, got %d", len(root.dirs))
	}
	if root.dirs[0].name != "a" || root.dirs[1].name != "b" || root.dirs[2].name != "c" {
		t.Errorf("dirs not sorted: %v", []string{root.dirs[0].name, root.dirs[1].name, root.dirs[2].name})
	}
}

func TestApplyGroupingWithCollapsedAllLevels(t *testing.T) {
	branches := []branchItem{
		mkBranch("a/b/c"),
		mkBranch("a/d"),
	}
	items := itemsFromBranches(branches)
	collapsed := map[string]bool{"a/": true, "a/b/": true}
	grouped := applyGrouping(items, collapsed)

	// Both dirs collapsed, only group rows shown.
	assertDisplay(t, grouped, []string{
		"▸ a/ (2)",
	})
}
