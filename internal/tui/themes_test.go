package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestThemeByName verifies every theme name resolves and unknown returns false.
func TestThemeByName(t *testing.T) {
	for _, name := range ThemeNames() {
		if _, ok := ThemeByName(name); !ok {
			t.Errorf("ThemeByName(%q) = false, want true (registered name)", name)
		}
	}
	if _, ok := ThemeByName("nonexistent-theme"); ok {
		t.Error("ThemeByName(nonexistent) = true, want false")
	}
}

// TestThemeNamesIncludesDefault guards the revert option — "default" must stay
// in the list so users can roll back from a chosen theme via the settings menu.
func TestThemeNamesIncludesDefault(t *testing.T) {
	for _, n := range ThemeNames() {
		if n == "default" {
			return
		}
	}
	t.Error(`ThemeNames() = [...] missing "default" — users can't roll back`)
}

// TestThemeColorsComplete asserts every theme variant sets every slot. A nil
// slot would render transparent or fallback to the terminal default, breaking
// the palette; we want a compile/runtime check that nothing was missed.
func TestThemeColorsComplete(t *testing.T) {
	for _, theme := range themes {
		t.Run(theme.Name+"/light", func(t *testing.T) {
			checkThemeColors(t, theme.Light)
		})
		t.Run(theme.Name+"/dark", func(t *testing.T) {
			checkThemeColors(t, theme.Dark)
		})
	}
}

func checkThemeColors(t *testing.T, c ThemeColors) {
	t.Helper()
	slots := []struct {
		name string
		val  lipgloss.Color
	}{
		{"Accent", c.Accent},
		{"Current", c.Current},
		{"Faint", c.Faint},
		{"Danger", c.Danger},
		{"Invert", c.Invert},
		{"Amber", c.Amber},
		{"Blue", c.Blue},
		{"Violet", c.Violet},
		{"Teal", c.Teal},
		{"Background", c.Background},
		{"Foreground", c.Foreground},
	}
	for _, s := range slots {
		if string(s.val) == "" {
			t.Errorf("slot %s is empty", s.name)
		}
	}
}

// TestResolveAppearancePassthrough ensures light/dark are returned unchanged —
// only "" and "system" trigger detection.
func TestResolveAppearancePassthrough(t *testing.T) {
	if got, want := resolveAppearance("light"), "light"; got != want {
		t.Errorf(`resolveAppearance("light") = %q, want %q`, got, want)
	}
	if got, want := resolveAppearance("dark"), "dark"; got != want {
		t.Errorf(`resolveAppearance("dark") = %q, want %q`, got, want)
	}
}

// TestResolveAppearanceSystemFallback ensures "system" never panics and always
// resolves to a concrete variant, regardless of OS detection availability.
func TestResolveAppearanceSystemFallback(t *testing.T) {
	for _, in := range []string{"system", ""} {
		got := resolveAppearance(in)
		if got != "light" && got != "dark" {
			t.Errorf(`resolveAppearance(%q) = %q, want "light" or "dark"`, in, got)
		}
	}
}

// TestDetectOSAppearanceNoCrash runs the OS-specific detection on the current
// platform and asserts it returns a safe value (light/dark) or ("", false),
// never panic, never empty-with-true.
func TestDetectOSAppearanceNoCrash(t *testing.T) {
	appearance, ok := detectOSAppearance()
	if ok {
		if appearance != "light" && appearance != "dark" {
			t.Errorf("detectOSAppearance() = (%q, true), want light or dark", appearance)
		}
	}
	// When !ok, the caller falls back to terminal-background detection — no
	// invariant to check here, just "didn't panic".
}

// TestSetPaletteUpdatesGlobals verifies setPalette writes through to every
// package-level color var the styles and column tints read from.
// TestSetPaletteUpdatesGlobals verifies setPalette writes through to every
// package-level color var the styles and column tints read from. Two
// TerminalColor values are equal iff their dynamic types + underlying values
// match, so interface comparison suffices.
func TestSetPaletteUpdatesGlobals(t *testing.T) {
	defer setPalette("default", "dark")
	setPalette("gruvbox", "dark")

	want := gruvboxTheme.Dark
	if colorAccent != want.Accent {
		t.Errorf("colorAccent = %v, want %v", colorAccent, want.Accent)
	}
	if colorCurrent != want.Current {
		t.Errorf("colorCurrent = %v, want %v", colorCurrent, want.Current)
	}
	if colorFaint != want.Faint {
		t.Errorf("colorFaint = %v, want %v", colorFaint, want.Faint)
	}
	if ColorSHA != want.Amber {
		t.Errorf("ColorSHA = %v, want %v (theme.Amber)", ColorSHA, want.Amber)
	}
	if ColorAuthor != want.Blue {
		t.Errorf("ColorAuthor = %v, want %v (theme.Blue)", ColorAuthor, want.Blue)
	}
	if len(graphLaneColors) != 7 {
		t.Errorf("len(graphLaneColors) = %d, want 7", len(graphLaneColors))
	}
}

// TestSetPaletteFreezesLight ensures "light" freezes the Light variant into the
// globals — colorAccent should equal the theme's Light Accent, not Dark.
func TestSetPaletteFreezesLight(t *testing.T) {
	defer setPalette("default", "dark")
	setPalette("solarized", "light")
	if colorAccent != solarizedTheme.Light.Accent {
		t.Errorf("colorAccent = %v, want solarized Light Accent %v",
			colorAccent, solarizedTheme.Light.Accent)
	}
	if ActiveResolvedAppearance != "light" {
		t.Errorf("ActiveResolvedAppearance = %q, want %q",
			ActiveResolvedAppearance, "light")
	}
}

// TestSetPaletteUnknownThemeFallsBack checks that an invalid theme name reverts
// to the default theme rather than leaving the palette half-set.
func TestSetPaletteUnknownThemeFallsBack(t *testing.T) {
	defer setPalette("default", "dark")
	setPalette("does-not-exist", "dark")
	if ActiveTheme.Name != "default" {
		t.Errorf("ActiveTheme.Name = %q, want %q", ActiveTheme.Name, "default")
	}
}

// TestPaletteForIsPure verifies PaletteFor doesn't mutate active state — it's
// used by the settings menu for swatches and must not preempt live preview.
func TestPaletteForIsPure(t *testing.T) {
	setPalette("default", "dark")
	beforeTheme, beforeAppearance, beforeResolved :=
		ActiveTheme.Name, ActiveAppearance, ActiveResolvedAppearance

	// Compute palettes for several theme/appearance pairs — none should
	// mutate the active state.
	_ = PaletteFor("gruvbox", "light")
	_ = PaletteFor("solarized", "system")
	_ = PaletteFor("nord", "dark")

	if ActiveTheme.Name != beforeTheme {
		t.Errorf("PaletteFor mutated ActiveTheme: got %q, want %q",
			ActiveTheme.Name, beforeTheme)
	}
	if ActiveAppearance != beforeAppearance {
		t.Errorf("PaletteFor mutated ActiveAppearance: got %q, want %q",
			ActiveAppearance, beforeAppearance)
	}
	if ActiveResolvedAppearance != beforeResolved {
		t.Errorf("PaletteFor mutated ActiveResolvedAppearance: got %q, want %q",
			ActiveResolvedAppearance, beforeResolved)
	}
}
