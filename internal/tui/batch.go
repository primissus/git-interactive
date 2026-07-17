package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// BatchSpec drives a resilient bulk operation: each target item is processed on
// its own, and a failure pauses the List to ask whether to continue rather than
// aborting the whole run. Set it on an Operation instead of Run for destructive
// bulk actions (delete, force delete, archive) where one bad branch should not
// strand the rest.
type BatchSpec struct {
	// Verb labels the summary line, e.g. "deleted", "archived".
	Verb string
	// Order returns the target items in processing order (e.g. oldest branch
	// first). nil processes them as given.
	Order func([]Item) []Item
	// Step performs the action on one item, returning an error to surface.
	Step func(Item) error
	// Refresh, when non-nil, returns a command to reload the list after the run
	// completes (typically SetItems).
	Refresh func() tea.Cmd
}

// batchFail records one item that Step could not process.
type batchFail struct {
	label string
	err   string
}

// batchRun is the in-flight state of a BatchSpec: which items remain, what has
// succeeded or failed, and whether the user waved off further prompts with "a".
type batchRun struct {
	spec    BatchSpec
	items   []Item // ordered targets
	pos     int    // index of the next item to process
	okCount int
	fails   []batchFail
	all     bool // continue past every remaining error without asking
}

// startBatch orders the targets and runs the batch to its first pause (an error
// prompt) or to completion.
func (l *List) startBatch(op Operation, items []Item) tea.Cmd {
	ordered := items
	if op.Batch.Order != nil {
		ordered = op.Batch.Order(items)
	}
	l.batch = &batchRun{spec: *op.Batch, items: ordered}
	l.mode = modeList
	l.pending = nil
	if l.selectMode {
		l.exitSelectMode()
	}
	return l.advanceBatch()
}

// advanceBatch processes items until one fails (pausing for a continue prompt)
// or the run is done. Steps run synchronously, matching Operation.Run.
func (l *List) advanceBatch() tea.Cmd {
	br := l.batch
	for br.pos < len(br.items) {
		it := br.items[br.pos]
		err := br.spec.Step(it)
		br.pos++
		if err != nil {
			br.fails = append(br.fails, batchFail{label: it.FilterValue(), err: err.Error()})
			if !br.all {
				l.mode = modeBatchPrompt
				return nil
			}
			continue
		}
		br.okCount++
	}
	return l.finishBatch()
}

// finishBatch clears batch state, reports a summary, and triggers the refresh.
func (l *List) finishBatch() tea.Cmd {
	br := l.batch
	l.batch = nil
	l.mode = modeList

	summary := fmt.Sprintf(chrome().BatchSummary, br.spec.Verb, br.okCount)
	if len(br.fails) > 0 {
		reasons := make([]string, len(br.fails))
		for i, f := range br.fails {
			reasons[i] = f.label + ": " + f.err
		}
		summary += fmt.Sprintf(chrome().BatchFailSuffix, len(br.fails), strings.Join(reasons, "; "))
	}

	cmds := []tea.Cmd{Status(summary)}
	if br.spec.Refresh != nil {
		cmds = append(cmds, br.spec.Refresh())
	}
	return tea.Batch(cmds...)
}

// updateBatchPrompt handles the continue/stop keys shown after a failed step.
func (l *List) updateBatchPrompt(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "y", "enter":
		l.mode = modeList
		return l.advanceBatch()
	case "a":
		l.batch.all = true
		l.mode = modeList
		return l.advanceBatch()
	case "n", "esc", "ctrl+c":
		return l.finishBatch()
	}
	return nil
}

// batchPromptView renders the after-failure prompt: which item failed, why, how
// many remain, and the continue/stop keys.
func (l *List) batchPromptView() string {
	br := l.batch
	last := br.fails[len(br.fails)-1]
	remaining := len(br.items) - br.pos

	var b strings.Builder
	b.WriteString(l.styles.ConfirmPrompt.Render(br.spec.Verb + " failed: " + last.label))
	b.WriteByte('\n')
	b.WriteString(l.styles.ConfirmPhrase.Render(last.err))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, chrome().BatchContinuePrompt, remaining)
	b.WriteString(l.styles.Help.Render(chrome().BatchFooter))
	return l.styles.Overlay.Render(b.String())
}
