package cli

import (
	"fmt"
	"sort"
	"strings"

	"git-interact/internal/tui"
)

type branchGroupItem struct {
	prefix    string
	label     string
	depth     int
	count     int
	collapsed bool
}

func (g branchGroupItem) Columns() []string {
	marker := "▾"
	if g.collapsed {
		marker = "▸"
	}
	indent := strings.Repeat("  ", g.depth)
	return []string{fmt.Sprintf("%s%s %s (%d)", indent, marker, g.label, g.count), "", "", ""}
}

func (g branchGroupItem) FilterValue() string { return "" }
func (g branchGroupItem) Current() bool       { return false }
func (g branchGroupItem) DefaultOp() string   { return "toggle group" }

type treeNode struct {
	name     string
	dirs     []*treeNode
	branches []branchItem
	count    int
}

func (n *treeNode) insert(path string, b branchItem) {
	n.count++
	idx := strings.Index(path, "/")
	if idx < 0 {
		n.branches = append(n.branches, branchItem{b: b.b, merged: b.merged})
		return
	}
	dirName := path[:idx]
	rest := path[idx+1:]
	for _, child := range n.dirs {
		if child.name == dirName {
			child.insert(rest, b)
			return
		}
	}
	child := &treeNode{name: dirName}
	n.dirs = append(n.dirs, child)
	child.insert(rest, b)
}

func (n *treeNode) sortNode() {
	sort.Slice(n.dirs, func(i, j int) bool { return n.dirs[i].name < n.dirs[j].name })
	for _, d := range n.dirs {
		d.sortNode()
	}
}

func leafName(full string) string {
	if idx := strings.LastIndex(full, "/"); idx >= 0 {
		return full[idx+1:]
	}
	return full
}

func (n *treeNode) flatten(prefix string, depth int, collapsed map[string]bool, out *[]tui.Item) {
	indent := strings.Repeat("  ", depth)
	for _, d := range n.dirs {
		fp := prefix + d.name + "/"
		isCollapsed := collapsed[fp]
		*out = append(*out, branchGroupItem{
			prefix: fp, label: d.name + "/", depth: depth, count: d.count, collapsed: isCollapsed,
		})
		if !isCollapsed {
			d.flatten(fp, depth+1, collapsed, out)
		}
	}
	for _, b := range n.branches {
		b.displayName = indent + leafName(b.b.Name)
		*out = append(*out, b)
	}
}

func applyGrouping(items []tui.Item, collapsed map[string]bool) []tui.Item {
	if len(items) == 0 {
		return items
	}
	out := make([]tui.Item, 0, len(items))
	var rest []tui.Item
	if _, ok := items[0].(createBranchItem); ok {
		out = append(out, items[0])
		rest = items[1:]
	} else {
		rest = items
	}

	root := &treeNode{}
	for _, it := range rest {
		if b, ok := it.(branchItem); ok {
			root.insert(b.b.Name, b)
		}
	}
	root.sortNode()
	root.flatten("", 0, collapsed, &out)
	return out
}
