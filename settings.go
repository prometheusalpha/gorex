package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"

	"gorex/internal/rex"
)

// settings are the user's choices, kept in settings.json beside the
// server's state.
type settings struct {
	// Appearance is "light", "dark", or "" to follow the system.
	Appearance string  `json:"appearance,omitempty"`
	FontSize   float32 `json:"fontSize,omitempty"`
	// FontFamily is a family, or a comma-separated list, that the
	// terminals try before the embedded JetBrains Mono.
	FontFamily string `json:"fontFamily,omitempty"`
	// FontFeatures are OpenType features as Ghostty's font-feature takes
	// them: "-calt" turns most coding fonts' ligatures off, "ss01" on.
	FontFeatures []string `json:"fontFeatures,omitempty"`
	HideSidebar  bool     `json:"hideSidebar,omitempty"`
	// LineHeight is the height of the terminals' rows, as a multiple of
	// the font's: 1.2 makes them 20% taller, as Ghostty's
	// adjust-cell-height = 20%.
	LineHeight float32 `json:"lineHeight,omitempty"`
	// Theme names one of Ghostty's themes, as its theme setting does:
	// "Ayu Mirage", or "light:Rose Pine Dawn,dark:Rose Pine".
	Theme string `json:"theme,omitempty"`
}

const defaultFontSize = 11.6

var prefs settings

func settingsPath() string { return filepath.Join(rex.Dir(), "settings.json") }

// The terminals' font and colors without settings, which reloading starts
// from again, as a key taken out of settings.json goes back to its default.
var (
	baseFont                    = termFont
	baseLightTerm, baseDarkTerm = lightTerm, darkTerm
)

func loadSettings() {
	s, err := readSettings()
	if err != nil {
		log.Printf("settings: %v", err)
	}
	if err := applySettings(s); err != nil {
		log.Printf("settings: %v", err)
	}
	applyAppearance()
}

func readSettings() (settings, error) {
	var s settings
	b, err := settingsFile()
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return settings{}, fmt.Errorf("%s: %w", settingsPath(), err)
	}
	return s, nil
}

// settingsFile reads settings.json, as {} when it is missing or blank.
func settingsFile() ([]byte, error) {
	b, err := os.ReadFile(settingsPath())
	if errors.Is(err, fs.ErrNotExist) || err == nil && len(bytes.TrimSpace(b)) == 0 {
		return []byte("{}"), nil
	}
	return b, err
}

// applySettings makes s the settings but for the appearance, keeping the
// default colors when its theme does not load.
func applySettings(s settings) error {
	prefs = s
	if prefs.FontSize < 6 || prefs.FontSize > 40 {
		prefs.FontSize = defaultFontSize
	}
	termFont = baseFont
	termFont.Size = prefs.FontSize
	if f := strings.TrimSpace(prefs.FontFamily); f != "" {
		termFont.Family = f + ", " + termFont.Family
	}
	termFont.Features = prefs.FontFeatures
	if prefs.LineHeight >= 0.8 && prefs.LineHeight <= 2 {
		termFont.LineHeight = prefs.LineHeight
	}
	lightTerm, darkTerm = baseLightTerm, baseDarkTerm
	themedLight, themedDark = nil, nil
	var err error
	if prefs.Theme != "" {
		var light, dark *terminal.Theme
		if light, dark, err = loadThemes(prefs.Theme); err == nil {
			applyTheme(light, dark)
		}
	}
	return err
}

// reloadSettings applies settings.json again, to the panes open too, as
// Ghostty's Reload Configuration. A file that is not JSON changes nothing.
func (a *App) reloadSettings() {
	s, err := readSettings()
	if err != nil {
		mygo.Dialog.Error("GoRex could not read its settings", err.Error())
		return
	}
	err = applySettings(s)
	applyAppearance()
	a.eachTerm(func(t *terminal.Terminal) {
		t.SetFont(termFont)
		t.SetTheme(lightTerm, darkTerm)
	})
	if err != nil {
		mygo.Dialog.Error("GoRex could not load the theme", err.Error())
	}
}

func (a *App) eachTerm(fn func(*terminal.Terminal)) {
	for _, t := range a.allTabs() {
		for _, p := range t.panes() {
			if p.term != nil {
				fn(p.term)
			}
		}
	}
}

// saveSetting writes one setting into settings.json, the zero value as
// none, keeping the others as the file has them: edits there that Reload
// Settings has not applied yet stay. It writes nothing over a file that
// is not JSON, the user's mid-edit.
func saveSetting(key string, v any) {
	b, err := settingsFile()
	var m map[string]json.RawMessage
	if err == nil {
		err = json.Unmarshal(b, &m)
	}
	if err != nil {
		log.Printf("settings: %s not saved: %v", key, err)
		return
	}
	if m == nil {
		m = map[string]json.RawMessage{} // the file is null
	}
	if reflect.ValueOf(v).IsZero() {
		delete(m, key)
	} else {
		m[key], _ = json.Marshal(v)
	}
	b, _ = json.MarshalIndent(m, "", "  ")
	os.MkdirAll(rex.Dir(), 0o700)
	os.WriteFile(settingsPath(), append(b, '\n'), 0o600)
}

func applyAppearance() {
	switch prefs.Appearance {
	case "light":
		mygo.Theme.SetSource(mygo.ThemeLight)
	case "dark":
		mygo.Theme.SetSource(mygo.ThemeDark)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
	// The menu checks the appearance chosen from it, but not from the
	// palette or settings.json.
	if m := mygo.App.Menu(); m != nil {
		if it := m.ItemByID("appearance:" + prefs.Appearance); it != nil {
			it.SetChecked(true)
		}
	}
}

func (a *App) toggleSidebar() {
	prefs.HideSidebar = !prefs.HideSidebar
	saveSetting("hideSidebar", prefs.HideSidebar)
}

func (a *App) setAppearance(v string) {
	prefs.Appearance = v
	applyAppearance()
	saveSetting("appearance", v)
}

// setFontSize changes the size of the terminals' text, in every pane.
func (a *App) setFontSize(size float32) {
	size = min(max(size, 8), 32)
	prefs.FontSize = size
	termFont.Size = size
	a.eachTerm(func(t *terminal.Terminal) { t.SetFont(termFont) })
	saveSetting("fontSize", size)
}
