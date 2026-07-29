package cli

import (
	"testing"

	"git-interact/internal/tui"
)

func TestShortAuthor(t *testing.T) {
	cases := map[string]string{
		"Test User":        "Test U.",
		"Ada Lovelace":     "Ada L.",
		"Cher":             "Cher",
		"Mary Jane Watson": "Mary Jane W.",
	}
	for in, want := range cases {
		if got := tui.ShortAuthor(in); got != want {
			t.Errorf("ShortAuthor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFirstRef(t *testing.T) {
	if got, ok := firstRef(nil); ok || got != "" {
		t.Errorf("firstRef(nil) = %q, %v; want \"\", false", got, ok)
	}
	if got, ok := firstRef([]string{"main", "other"}); !ok || got != "main" {
		t.Errorf("firstRef([main, other]) = %q, %v; want main, true", got, ok)
	}
}

func TestPlural2(t *testing.T) {
	if got := plural2(1); got != "1 commit" {
		t.Errorf("plural2(1) = %q, want %q", got, "1 commit")
	}
	if got := plural2(3); got != "3 commits" {
		t.Errorf("plural2(3) = %q, want %q", got, "3 commits")
	}
}
