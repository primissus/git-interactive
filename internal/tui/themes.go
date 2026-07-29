package tui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ThemeColors holds the full color palette gint uses, in one named-role shape.
// Every theme supplies both a Light and a Dark variant; setPalette freezes one
// of them into the package-level color vars at startup (or on settings change).
//
// Slots map to the existing names like this:
//
//	Accent  → colorAccent  (purple in default)
//	Current → colorCurrent (green — current branch/HEAD marker)
//	Faint   → colorFaint   (gray — gutter, de-emphasized text)
//	Danger  → colorDanger  (red — destructive prompts)
//	Invert  → colorInvert  (selected row's foreground on Accent bg)
//	Amber   → ColorSHA tint
//	Blue    → ColorAuthor tint + graph lane
//	Violet  → graph lane
//	Teal    → graph lane
//	Background / Foreground are kept for swatch previews and future use; they
//	are not applied to any style today (gint stays transparent-backgrounded).
type ThemeColors struct {
	Accent     lipgloss.Color
	Current    lipgloss.Color
	Faint      lipgloss.Color
	Danger     lipgloss.Color
	Invert     lipgloss.Color
	Amber      lipgloss.Color
	Blue       lipgloss.Color
	Violet     lipgloss.Color
	Teal       lipgloss.Color
	Background lipgloss.Color
	Foreground lipgloss.Color
}

// Theme is one selectable entry in the settings menu. Each carries both
// appearance variants so the user can flip Light/Dark independently of the
// chosen theme.
type Theme struct {
	Name  string
	Light ThemeColors
	Dark  ThemeColors
}

// defaultTheme mirrors gint's pre-theming palette exactly, so a user with no
// settings.json sees no visual change.
var defaultTheme = Theme{
	Name: "default",
	Light: ThemeColors{
		Accent:     "#7D56F4",
		Current:    "#1A7F37",
		Faint:      "#6E7781",
		Danger:     "#CF222E",
		Invert:     "#FFFFFF",
		Amber:      "#9A6700",
		Blue:       "#0550AE",
		Violet:     "#8250DF",
		Teal:       "#1B7C83",
		Background: "#FFFFFF",
		Foreground: "#24292F",
	},
	Dark: ThemeColors{
		Accent:     "#A78BFA",
		Current:    "#3FB950",
		Faint:      "#8B949E",
		Danger:     "#F85149",
		Invert:     "#0D1117",
		Amber:      "#E3B341",
		Blue:       "#58A6FF",
		Violet:     "#D2A8FF",
		Teal:       "#39C5CF",
		Background: "#0D1117",
		Foreground: "#E6EDF3",
	},
}

// gruvbox: Light = light soft, Dark = medium dark (the two variants the user
// named — single theme entry, selected by appearance).
var gruvboxTheme = Theme{
	Name: "gruvbox",
	Light: ThemeColors{
		Accent:     "#8F3F71", // purple
		Current:    "#79740E", // green
		Faint:      "#7C6F64", // gray4
		Danger:     "#9D0006", // red
		Invert:     "#F2E5BC", // bg0_soft
		Amber:      "#B57614", // yellow
		Blue:       "#076678", // aqua
		Violet:     "#8F3F71",
		Teal:       "#427B58",
		Background: "#F2E5BC",
		Foreground: "#3C3836",
	},
	Dark: ThemeColors{
		Accent:     "#D3869B", // purple
		Current:    "#B8BB26", // green
		Faint:      "#928374", // gray0
		Danger:     "#FB4934", // red
		Invert:     "#282828", // bg0
		Amber:      "#FABD2F", // yellow
		Blue:       "#83A598", // aqua
		Violet:     "#D3869B",
		Teal:       "#8EC07C",
		Background: "#282828",
		Foreground: "#EBDBB2",
	},
}

var solarizedTheme = Theme{
	Name: "solarized",
	Light: ThemeColors{
		Accent:     "#268BD2", // blue
		Current:    "#859900", // green
		Faint:      "#93A1A1", // base2
		Danger:     "#DC322F", // red
		Invert:     "#FDF6E3", // base3
		Amber:      "#B58900", // yellow/orange
		Blue:       "#268BD2",
		Violet:     "#6C71C4", // violet
		Teal:       "#2AA198", // cyan
		Background: "#FDF6E3",
		Foreground: "#657B83",
	},
	Dark: ThemeColors{
		Accent:     "#268BD2",
		Current:    "#859900",
		Faint:      "#586E75", // base01
		Danger:     "#DC322F",
		Invert:     "#002B36", // base03
		Amber:      "#B58900",
		Blue:       "#268BD2",
		Violet:     "#6C71C4",
		Teal:       "#2AA198",
		Background: "#002B36",
		Foreground: "#839496",
	},
}

// catppuccin: Light = Latte, Dark = Mocha.
var catppuccinTheme = Theme{
	Name: "catppuccin",
	Light: ThemeColors{
		Accent:     "#8839EF", // mauve
		Current:    "#40A02B", // green
		Faint:      "#ACB0BE", // surface2
		Danger:     "#D20F39", // red
		Invert:     "#EFF1F5", // base
		Amber:      "#DF8E1D", // yellow
		Blue:       "#1E66F5", // blue
		Violet:     "#8839EF",
		Teal:       "#179299", // teal
		Background: "#EFF1F5",
		Foreground: "#4C4F69",
	},
	Dark: ThemeColors{
		Accent:     "#CBA6F7", // mauve
		Current:    "#A6E3A1", // green
		Faint:      "#6C7086", // overlay0
		Danger:     "#F38BA8", // red
		Invert:     "#1E1E2E", // base
		Amber:      "#F9E2AF", // yellow
		Blue:       "#89B4FA", // blue
		Violet:     "#CBA6F7",
		Teal:       "#94E2D5", // teal
		Background: "#1E1E2E",
		Foreground: "#CDD6F4",
	},
}

var githubTheme = Theme{
	Name: "github",
	Light: ThemeColors{
		Accent:     "#0969DA", // blue
		Current:    "#1A7F37", // green
		Faint:      "#6E7781", // neutral
		Danger:     "#CF222E", // red
		Invert:     "#FFFFFF",
		Amber:      "#9A6700",
		Blue:       "#0550AE",
		Violet:     "#8250DF",
		Teal:       "#1B7C83",
		Background: "#FFFFFF",
		Foreground: "#24292F",
	},
	Dark: ThemeColors{
		Accent:     "#58A6FF",
		Current:    "#3FB950",
		Faint:      "#8B949E",
		Danger:     "#F85149",
		Invert:     "#0D1117",
		Amber:      "#E3B341",
		Blue:       "#58A6FF",
		Violet:     "#D2A8FF",
		Teal:       "#39C5CF",
		Background: "#0D1117",
		Foreground: "#E6EDF3",
	},
}

var nordTheme = Theme{
	Name: "nord",
	Light: ThemeColors{
		Accent:     "#5E81AC", // frost (fjord)
		Current:    "#A3BE8C", // aurora green
		Faint:      "#D8DEE9", // nord4
		Danger:     "#BF616A", // aurora red
		Invert:     "#ECEFF4", // nord5 (snow storm)
		Amber:      "#EBCB8B", // aurora yellow
		Blue:       "#81A1C1", // frost
		Violet:     "#B48EAD", // aurora purple
		Teal:       "#8FBCBB", // frost (ice)
		Background: "#ECEFF4",
		Foreground: "#2E3440",
	},
	Dark: ThemeColors{
		Accent:     "#88C0D0", // frost (ice)
		Current:    "#A3BE8C",
		Faint:      "#4C566A", // nord3
		Danger:     "#BF616A",
		Invert:     "#2E3440", // nord0
		Amber:      "#EBCB8B",
		Blue:       "#81A1C1", // frost
		Violet:     "#B48EAD",
		Teal:       "#8FBCBB",
		Background: "#2E3440",
		Foreground: "#D8DEE9",
	},
}

// rose-pine: Light = Dawn, Dark = base Rose Pine.
var rosePineTheme = Theme{
	Name: "rose-pine",
	Light: ThemeColors{
		Accent:     "#907AA9", // iris
		Current:    "#286983", // pine
		Faint:      "#9893A5", // subtle
		Danger:     "#B4637A", // love
		Invert:     "#FAF4ED", // base
		Amber:      "#EA9D34", // gold
		Blue:       "#56949F", // foam
		Violet:     "#907AA9",
		Teal:       "#56949F",
		Background: "#FAF4ED",
		Foreground: "#575279",
	},
	Dark: ThemeColors{
		Accent:     "#C4A7E7", // iris
		Current:    "#31748F", // pine
		Faint:      "#908CAA", // subtle
		Danger:     "#EB6F92", // love
		Invert:     "#191724", // base
		Amber:      "#F6C177", // gold
		Blue:       "#9CCFD8", // foam
		Violet:     "#C4A7E7",
		Teal:       "#9CCFD8",
		Background: "#191724",
		Foreground: "#E0DEF4",
	},
}

// themes is the registration order — what the settings menu lists top to bottom.
// default leads so it's the natural revert target.
var themes = []Theme{
	defaultTheme,
	gruvboxTheme,
	solarizedTheme,
	catppuccinTheme,
	githubTheme,
	nordTheme,
	rosePineTheme,
}

// ThemeNames returns every registered theme's name in list order. "default" is
// always present so the user can roll back from a chosen theme.
func ThemeNames() []string {
	out := make([]string, len(themes))
	for i, t := range themes {
		out[i] = t.Name
	}
	return out
}

// ThemeByName looks up a theme by its Name. Unknown → default + false.
func ThemeByName(name string) (Theme, bool) {
	for _, t := range themes {
		if t.Name == name {
			return t, true
		}
	}
	return defaultTheme, false
}

// --- active theme state ---------------------------------------------------

var (
	// ActiveTheme is the currently selected theme. Defaults to default.
	ActiveTheme = defaultTheme
	// ActiveAppearance is the user's selection: "system", "light", "dark", or
	// "" (treated as "system"). Set once at startup from settings.json and again
	// whenever the user changes it inside the settings overlay.
	ActiveAppearance string
	// ActiveResolvedAppearance is what "system" resolved to ("light" or "dark")
	// at the last setPalette call — the variant actually driving the colors.
	// Exposed so the settings menu can show "(resolved: dark)" as a hint.
	ActiveResolvedAppearance string
)

// resolveAppearance turns "system" / "" into "light" or "dark" via OS + terminal
// detection. "light" and "dark" pass through unchanged.
func resolveAppearance(a string) string {
	if a != "system" && a != "" {
		return a
	}
	if detected, ok := detectOSAppearance(); ok {
		return detected
	}
	// Terminal-background detection — same source AdaptiveColor uses.
	if lipgloss.DefaultRenderer().HasDarkBackground() {
		return "dark"
	}
	return "light"
}

// detectOSAppearance returns ("light" or "dark", true) when the OS dark-mode
// preference is known, or ("", false) when it can't be determined (the caller
// then falls back to terminal-background detection). Shelling out matches
// gint's existing "shell out to git" convention — quick, dependency-free, and
// platform code stays contained in this one function.
func detectOSAppearance() (string, bool) {
	switch runtime.GOOS {
	case "darwin":
		// `defaults read -g AppleInterfaceStyle` prints "Dark" under dark mode
		// and exits non-zero (exit 1, no DangerousAppleInterfaceStyle key) under
		// light mode — distinguish that from "command failed".
		cmd := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle")
		out, err := cmd.Output()
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
				return "light", true // command ran, no Dark key → light mode
			}
			return "", false // command not found / unexpected → fall through
		}
		if strings.TrimSpace(string(out)) == "Dark" {
			return "dark", true
		}
		return "light", true
	case "linux":
		// GNOME / GTK4 via gsettings.
		out, err := exec.Command("gsettings", "get",
			"org.gnome.desktop.interface", "color-scheme").Output()
		if err == nil {
			v := strings.Trim(strings.TrimSpace(string(out)), "'")
			switch v {
			case "prefer-dark":
				return "dark", true
			case "prefer-light", "default":
				return "light", true
			}
		}
		// KDE: ~/.config/kdeglobals [General] ColorScheme contains "Dark" for
		// dark schemes. Cheap heuristic — avoids INI parsing for one field.
		home := os.Getenv("HOME")
		if home != "" {
			if data, err := os.ReadFile(filepath.Join(home, ".config", "kdeglobals")); err == nil {
				s := string(data)
				if strings.Contains(s, "ColorScheme=Dark") ||
					strings.Contains(s, "ColorScheme=Breeze Dark") {
					return "dark", true
				}
				return "light", true
			}
		}
		return "", false
	case "windows":
		// reg.exe reads the registry value; avoids syscall plumbing.
		out, err := exec.Command("reg", "query",
			`HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`,
			"/v", "AppsUseLightTheme").Output()
		if err == nil {
			s := strings.TrimSpace(string(out))
			switch {
			case strings.HasSuffix(s, "0x0"):
				return "dark", true
			case strings.HasSuffix(s, "0x1"):
				return "light", true
			}
		}
		return "", false
	}
	return "", false
}

// setPalette freezes the active theme's resolved Light or Dark variant into the
// package-level color vars that styles.go and render.go read. Called once at
// startup (after settings load) and again whenever the settings overlay changes
// the user's selection (live preview) or reverts (esc).
//
// We use plain lipgloss.Color for the frozen variants — NOT AdaptiveColor — so
// they don't re-query the terminal background at render time. The resolution
// happens once here and the result is locked in for the session.
func setPalette(themeName, appearance string) {
	theme, ok := ThemeByName(themeName)
	if !ok {
		theme = defaultTheme
	}
	resolved := resolveAppearance(appearance)

	var c ThemeColors
	if resolved == "light" {
		c = theme.Light
	} else {
		c = theme.Dark
	}

	colorAccent = c.Accent
	colorCurrent = c.Current
	colorFaint = c.Faint
	colorDanger = c.Danger
	colorInvert = c.Invert

	// Exported column-tint vars — set at startup so command constructors pick
	// them up. (They won't update mid-session; acceptable per the plan.)
	ColorName = c.Accent
	ColorSHA = c.Amber
	ColorDate = c.Faint
	ColorAuthor = c.Blue
	ColorRef = c.Current

	graphLaneColors = []lipgloss.TerminalColor{
		c.Accent,
		c.Blue,
		c.Current,
		c.Amber,
		c.Danger,
		c.Violet,
		c.Teal,
	}

	ActiveTheme = theme
	ActiveAppearance = appearance
	ActiveResolvedAppearance = resolved
}

// PaletteFor returns the ThemeColors that setPalette would freeze for the given
// theme + appearance. Used by the settings menu to render live swatch previews
// without mutating global state.
func PaletteFor(themeName, appearance string) ThemeColors {
	theme, ok := ThemeByName(themeName)
	if !ok {
		theme = defaultTheme
	}
	resolved := resolveAppearance(appearance)
	if resolved == "light" {
		return theme.Light
	}
	return theme.Dark
}
