package tui

import (
	"strings"
	"testing"
)

func renderDemoTable(t *testing.T, density Density) string {
	t.Helper()
	var b strings.Builder
	if err := RenderTable(&b, DemoColumns(), DemoItems(), TableOptions{
		Density: density,
		Header:  true,
		Marker:  true,
	}); err != nil {
		t.Fatalf("RenderTable: %v", err)
	}
	return b.String()
}

func TestRenderTableDensityGatesColumns(t *testing.T) {
	normal := renderDemoTable(t, DensityNormal)
	if !strings.Contains(normal, "last commit") || !strings.Contains(normal, "date") {
		t.Errorf("normal view missing expected columns:\n%s", normal)
	}
	if strings.Contains(normal, "author") {
		t.Errorf("normal view should hide the author column:\n%s", normal)
	}

	full := renderDemoTable(t, DensityFull)
	if !strings.Contains(full, "author") || !strings.Contains(full, "Ada Lovelace") {
		t.Errorf("full view missing the author column:\n%s", full)
	}

	short := renderDemoTable(t, DensityShort)
	if strings.Contains(short, "last commit") || strings.Contains(short, "date") {
		t.Errorf("short view should show only always-on columns:\n%s", short)
	}
	if !strings.Contains(short, "branch") {
		t.Errorf("short view missing the branch column:\n%s", short)
	}
}

func TestRenderTableMarksCurrentRow(t *testing.T) {
	out := renderDemoTable(t, DensityNormal)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// lines[0] is the header; the current item ("main") must carry the "* "
	// marker while others get two spaces.
	var mainLine, otherLine string
	for _, ln := range lines[1:] {
		if strings.Contains(ln, "main") {
			mainLine = ln
		}
		if strings.Contains(ln, "feat/tui-framework") {
			otherLine = ln
		}
	}
	if !strings.HasPrefix(mainLine, "* ") {
		t.Errorf("current row not marked: %q", mainLine)
	}
	if !strings.HasPrefix(otherLine, "  ") || strings.HasPrefix(otherLine, "* ") {
		t.Errorf("non-current row wrongly marked: %q", otherLine)
	}
}

func TestFitCell(t *testing.T) {
	if got := fitCell("hi", 5); got != "hi   " {
		t.Errorf("pad: got %q, want %q", got, "hi   ")
	}
	if got := fitCell("hello world", 6); got != "hello…" {
		t.Errorf("truncate: got %q, want %q", got, "hello…")
	}
	if got := fitCell("abc", 3); got != "abc" {
		t.Errorf("exact: got %q, want %q", got, "abc")
	}
}
