package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"git-interact/internal/git"
	"git-interact/internal/tui"
	"git-interact/internal/validate"
)

// tagItem adapts a git.Tag to tui.Item. Column order mirrors branchItem:
// name, message (the tag's own annotation, or the commit's subject for a
// lightweight tag), relative date, author (tagger, or the commit's author for
// a lightweight tag).
type tagItem struct {
	t git.Tag
}

func (i tagItem) Columns() []string   { return []string{i.t.Name, i.t.Subject, i.t.Date, i.t.Author} }
func (i tagItem) FilterValue() string { return i.t.Name }
func (i tagItem) Current() bool       { return i.t.Head }

// createTagItem is the pinned "create a tag" row shown above the list, same
// pattern as createBranchItem: it participates in the list so "new" is
// reachable both from Shift+N and, via DefaultOp, from Enter on its own row.
type createTagItem struct{}

func (createTagItem) Columns() []string   { return []string{"+ new tag (Shift+N)", "", "", ""} }
func (createTagItem) FilterValue() string { return "" }
func (createTagItem) Current() bool       { return false }
func (createTagItem) DefaultOp() string   { return "new" }

func tagColumns() []tui.Column {
	return []tui.Column{
		{Title: "tag", MinWidth: 8, Flex: true, Density: tui.DensityShort},
		{Title: "message", MaxWidth: 50, Density: tui.DensityNormal},
		{Title: "date", MinWidth: 10, Density: tui.DensityNormal, Color: tui.ColorDate},
		{Title: "author", MinWidth: 10, Density: tui.DensityFull, Color: tui.ColorAuthor},
	}
}

// tagFilters is the parsed form of tags's filter flags. Tags have no upstream
// tracking or branch-merge concept, so unlike branchFilters this only covers
// author and creation date.
type tagFilters struct {
	author string
	since  string // "" | 1d | 3d | 1w | 1m | ytd | 1y
}

func (f tagFilters) validate() error {
	switch f.since {
	case "", "1d", "3d", "1w", "1m", "ytd", "1y":
	default:
		return fmt.Errorf("--since: unknown bucket %q (want 1d, 3d, 1w, 1m, ytd, or 1y)", f.since)
	}
	return nil
}

// filterTags applies tagFilters over tags.
func filterTags(tags []git.Tag, f tagFilters) []git.Tag {
	cutoff := sinceCutoff(f.since)
	out := make([]git.Tag, 0, len(tags))
	for _, t := range tags {
		if f.author != "" && !strings.Contains(strings.ToLower(t.Author), strings.ToLower(f.author)) {
			continue
		}
		if f.since != "" && t.DateUnix < cutoff {
			continue
		}
		out = append(out, t)
	}
	return out
}

// sortTags reorders tags in place per mode. "off" leaves ListTags' own order
// (most-recently-created-first) untouched; "created" and the default
// "last-commit" both sort by creation date since, unlike a branch's tip, a
// tag's own creation date is exactly what's wanted either way.
func sortTags(tags []git.Tag, mode string) {
	switch mode {
	case "author":
		sort.SliceStable(tags, func(i, j int) bool {
			return strings.ToLower(tags[i].Author) < strings.ToLower(tags[j].Author)
		})
	case "off":
		// no-op: keep git's own ordering.
	default: // "created", "last-commit"
		sort.SliceStable(tags, func(i, j int) bool { return tags[i].DateUnix > tags[j].DateUnix })
	}
}

// loadTagItems fetches, filters, and sorts tags, and adapts them to tui.Items;
// includeCreateRow pins the create-tag row at index 0 (used for the
// interactive list, not for -I output or direct-menu mode).
func loadTagItems(ctx context.Context, r *git.Runner, f tagFilters, sortMode string, includeCreateRow bool) ([]tui.Item, error) {
	tags, err := git.ListTags(ctx, r)
	if err != nil {
		return nil, err
	}
	tags = filterTags(tags, f)
	sortTags(tags, sortMode)

	items := make([]tui.Item, 0, len(tags)+1)
	if includeCreateRow {
		items = append(items, createTagItem{})
	}
	for _, t := range tags {
		items = append(items, tagItem{t: t})
	}
	return items, nil
}

// targetTag resolves a single-tag operation's target, rejecting the
// create-row placeholder (or an empty/mixed selection).
func targetTag(items []tui.Item) (tagItem, bool) {
	if len(items) != 1 {
		return tagItem{}, false
	}
	t, ok := items[0].(tagItem)
	return t, ok
}

// targetTags resolves a bulk operation's targets, dropping the create-row
// placeholder if it was somehow included in the selection.
func targetTags(items []tui.Item) []tagItem {
	out := make([]tagItem, 0, len(items))
	for _, it := range items {
		if t, ok := it.(tagItem); ok {
			out = append(out, t)
		}
	}
	return out
}

func newTagsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tags [tag-name]",
		Aliases: []string{"tag"},
		Short:   "Interactive tabular list of tags",
		Args:    cobra.MaximumNArgs(1),
	}
	attachCommonFlags(cmd)
	cmd.Flags().StringP("new", "b", "", "create tag (at HEAD)")
	cmd.Flags().BoolP("delete", "D", false, "delete tag")
	cmd.Flags().String("author", "", "filter: tags by this tagger/author")
	cmd.Flags().String("since", "", "filter: created within 1d|3d|1w|1m|ytd|1y")
	cmd.RunE = runTags
	return cmd
}

func runTags(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	runner := git.NewRunner("")

	newName, _ := cmd.Flags().GetString("new")
	del, _ := cmd.Flags().GetBool("delete")

	switch {
	case newName != "":
		if err := git.CreateTag(ctx, runner, newName, ""); err != nil {
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "created tag %s\n", newName)
		return err
	case del:
		if len(args) != 1 {
			return fmt.Errorf("tags -D/--delete requires exactly one tag name")
		}
		if err := git.DeleteTag(ctx, runner, args[0]); err != nil {
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "deleted tag %s\n", args[0])
		return err
	}

	f, err := tagFiltersFromFlags(cmd)
	if err != nil {
		return err
	}
	flags := registeredFlags[cmd]
	sortMode := sortModeFromFlag(flags.sort)

	if len(args) == 1 {
		return runTagsDirectMenu(cmd, runner, args[0], f, sortMode)
	}

	interactive := flags.resolveInteractive()
	items, err := loadTagItems(ctx, runner, f, sortMode, interactive)
	if err != nil {
		return err
	}

	if !interactive {
		return tui.RenderTable(cmd.OutOrStdout(), tagColumns(), items, tui.TableOptions{
			Density: densityFromFlags(flags),
			Header:  true,
			Marker:  true,
		})
	}

	list := tui.New(tui.Config{
		Title:         "gint tags",
		Columns:       tagColumns(),
		Items:         items,
		Operations:    buildTagOperations(ctx, runner, f, sortMode, true),
		Density:       densityFromFlags(flags),
		Sort:          flags.sort,
		InitialCursor: 1, // focus stays on the first real tag, past the create row
	})
	p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err = p.Run()
	return err
}

func tagFiltersFromFlags(cmd *cobra.Command) (tagFilters, error) {
	author, _ := cmd.Flags().GetString("author")
	since, _ := cmd.Flags().GetString("since")
	f := tagFilters{author: author, since: since}
	if err := f.validate(); err != nil {
		return tagFilters{}, err
	}
	return f, nil
}

// runTagsDirectMenu implements `gint tags <name>`: skip the list, open the
// operations menu for that tag directly, with fuzzy op matching — same
// direct-menu mode as branch (.context/glossary.md).
func runTagsDirectMenu(cmd *cobra.Command, r *git.Runner, name string, f tagFilters, sortMode string) error {
	ctx := cmd.Context()
	items, err := loadTagItems(ctx, r, f, sortMode, false)
	if err != nil {
		return err
	}
	idx := -1
	for i, it := range items {
		if t, ok := it.(tagItem); ok && t.t.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("tags: no such tag %q", name)
	}

	list := tui.New(tui.Config{
		Title:           "gint tags " + name,
		Columns:         tagColumns(),
		Items:           items,
		Operations:      buildTagOperations(ctx, r, f, sortMode, false),
		InitialCursor:   idx,
		OpenMenuOnStart: true,
	})
	p := tea.NewProgram(list, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err = p.Run()
	return err
}

// buildTagOperations returns the tags view's operation registry.
// includeCreateRow must match the value used to build the list's initial
// items so a post-mutation refresh keeps the same shape.
func buildTagOperations(ctx context.Context, r *git.Runner, f tagFilters, sortMode string, includeCreateRow bool) []tui.Operation {
	refresh := func() tea.Cmd {
		items, err := loadTagItems(ctx, r, f, sortMode, includeCreateRow)
		if err != nil {
			return tui.Status(err.Error())
		}
		return tui.SetItems(items)
	}
	refreshWith := func(status string) tea.Cmd {
		return tea.Batch(tui.Status(status), refresh())
	}

	// oldestFirst orders bulk-delete targets from the oldest creation date to
	// the newest, mirroring branch's resilient-batch ordering.
	oldestFirst := func(items []tui.Item) []tui.Item {
		out := append([]tui.Item(nil), items...)
		sort.SliceStable(out, func(i, j int) bool {
			ti, iok := out[i].(tagItem)
			tj, jok := out[j].(tagItem)
			if !iok || !jok {
				return iok && !jok
			}
			return ti.t.DateUnix < tj.t.DateUnix
		})
		return out
	}

	return []tui.Operation{
		{
			Name: "checkout", Key: "C", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Check out this tag? (detached HEAD)"},
			Run: func(c tui.OpContext) tea.Cmd {
				t, ok := targetTag(c.Items)
				if !ok {
					return tui.Status("select a tag first")
				}
				if err := git.CheckoutTag(ctx, r, t.t.Name); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("checked out " + t.t.Name)
			},
		},
		{
			Name: "delete", Key: "D", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Delete this tag?"},
			Run: func(c tui.OpContext) tea.Cmd {
				t, ok := targetTag(c.Items)
				if !ok {
					return tui.Status("select a tag first")
				}
				if err := git.DeleteTag(ctx, r, t.t.Name); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("deleted " + t.t.Name)
			},
		},
		{
			Name: "push", Key: "P", Scope: tui.ScopeItem,
			Confirm: &tui.Confirm{Kind: tui.ConfirmYesNo, Prompt: "Push this tag to origin?"},
			Run: func(c tui.OpContext) tea.Cmd {
				t, ok := targetTag(c.Items)
				if !ok {
					return tui.Status("select a tag first")
				}
				if err := git.PushTag(ctx, r, t.t.Name); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("pushed " + t.t.Name)
			},
		},
		{
			Name: "copy sha", Key: "y", Scope: tui.ScopeItem,
			Run: func(c tui.OpContext) tea.Cmd {
				t, ok := targetTag(c.Items)
				if !ok {
					return tui.Status("select a tag first")
				}
				if err := copyToClipboard(t.t.SHA); err != nil {
					return tui.Status("sha " + t.t.SHA + " (clipboard unavailable: " + err.Error() + ")")
				}
				return tui.Status("copied sha " + t.t.SHA + " (" + t.t.Name + ")")
			},
		},
		{
			Name: "new", Key: "N", Scope: tui.ScopeList,
			Input: &tui.InputSpec{Prompt: "New tag", Placeholder: "tag name", Validate: validate.TagName},
			Run: func(c tui.OpContext) tea.Cmd {
				if err := git.CreateTag(ctx, r, c.Input, ""); err != nil {
					return tui.Status(err.Error())
				}
				return refreshWith("created tag " + c.Input)
			},
		},
		{
			Name: "delete", Scope: tui.ScopeItem, BulkOnly: true,
			Confirm: &tui.Confirm{Kind: tui.ConfirmTyped, Prompt: "Delete the selected tags?", Phrase: "delete all"},
			Batch: &tui.BatchSpec{
				Verb:  "deleted",
				Order: oldestFirst,
				Step: func(it tui.Item) error {
					return git.DeleteTag(ctx, r, it.(tagItem).t.Name)
				},
				Refresh: refresh,
			},
		},
	}
}
