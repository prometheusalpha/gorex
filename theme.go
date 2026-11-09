package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// themeDirs are where Ghostty keeps its themes: the user's, then those it
// ships.
func themeDirs() []string {
	var dirs []string
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		home, _ := os.UserHomeDir()
		config = filepath.Join(home, ".config")
	}
	dirs = append(dirs, filepath.Join(config, "ghostty", "themes"))
	return append(dirs,
		"/Applications/Ghostty.app/Contents/Resources/ghostty/themes",
		"/usr/share/ghostty/themes",
	)
}

// loadThemes reads a theme as Ghostty's theme setting takes it: a name,
// a path, or "light:A,dark:B", one for each appearance.
func loadThemes(spec string) (light, dark *terminal.Theme, err error) {
	if l, d, ok := strings.Cut(spec, ","); ok {
		for _, part := range []string{l, d} {
			kind, name, _ := strings.Cut(strings.TrimSpace(part), ":")
			t, err := loadTheme(strings.TrimSpace(name))
			if err != nil {
				return nil, nil, err
			}
			switch kind {
			case "light":
				light = t
			case "dark":
				dark = t
			default:
				return nil, nil, fmt.Errorf("theme %q: %q is neither light: nor dark:", spec, part)
			}
		}
		if light == nil || dark == nil {
			return nil, nil, fmt.Errorf("theme %q wants a light and a dark theme", spec)
		}
		return light, dark, nil
	}
	t, err := loadTheme(spec)
	return t, t, err
}

func loadTheme(name string) (*terminal.Theme, error) {
	paths := []string{name}
	if !filepath.IsAbs(name) {
		paths = nil
		for _, d := range themeDirs() {
			paths = append(paths, filepath.Join(d, name))
		}
	}
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		defer f.Close()
		return parseTheme(bufio.NewScanner(f))
	}
	return nil, fmt.Errorf("no theme %q in %s", name, strings.Join(themeDirs(), ", "))
}

// parseTheme reads the colors of a Ghostty theme: lines of key = value.
func parseTheme(sc *bufio.Scanner) (*terminal.Theme, error) {
	t := &terminal.Theme{}
	color := func(v string) ui.Color {
		if !strings.HasPrefix(v, "#") {
			v = "#" + v
		}
		return ui.Hex(v)
	}
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if !ok || strings.HasPrefix(strings.TrimSpace(k), "#") {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "background":
			t.Background = color(v)
		case "foreground":
			t.Foreground = color(v)
		case "cursor-color":
			t.Cursor = color(v)
		case "cursor-text":
			t.CursorText = color(v)
		case "selection-background":
			t.Selection = color(v)
		case "selection-foreground":
			t.SelectionText = color(v)
		case "palette":
			n, c, _ := strings.Cut(v, "=")
			if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil && i >= 0 && i < len(t.Palette) {
				t.Palette[i] = color(strings.TrimSpace(c))
			}
		}
	}
	if t.Background.A == 0 || t.Foreground.A == 0 {
		return nil, fmt.Errorf("a theme without a background and a foreground")
	}
	return t, sc.Err()
}

// themed are the window's colors around terminals of a theme: a theme's
// dark or light, whatever the system's, with the panes' cards of its
// background, as Ghostty's window-theme = auto.
var themedLight, themedDark *colors

func applyTheme(light, dark *terminal.Theme) {
	lightTerm, darkTerm = light, dark
	themedLight, themedDark = colorsAround(light), colorsAround(dark)
}

func colorsAround(t *terminal.Theme) *colors {
	k, shade := lightColors, 0.07
	if luminance(t.Background) < 0.5 {
		k, shade = darkColors, 0.25
	}
	k.cardFocused = t.Background.Alpha(0.96)
	k.card = t.Background.Alpha(0.78)
	// The window's gradient, shaped as GoRex's own but of the theme's
	// colors: its background a shade darker than the cards', toward its
	// magenta at the top and its yellow at the bottom.
	black := ui.RGB(0, 0, 0)
	k.bgMid = mix(t.Background, black, shade)
	k.bgLow = mix(t.Background, black, shade*1.2)
	k.bgTop, k.bgBottom, k.bgTint = k.bgMid, k.bgLow, ui.Color{}
	if magenta := t.Palette[5]; magenta.A != 0 {
		k.bgTop = mix(k.bgMid, magenta, 0.1)
		k.bgTint = magenta.Alpha(0.1)
	}
	if yellow := t.Palette[3]; yellow.A != 0 {
		k.bgBottom = mix(k.bgLow, yellow, 0.07)
	}
	return &k
}

// mix returns a, moved by t toward b.
func mix(a, b ui.Color, t float64) ui.Color {
	m := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return ui.RGB(m(a.R, b.R), m(a.G, b.G), m(a.B, b.B))
}

func luminance(c ui.Color) float64 {
	return (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
}
