package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReloadSettings applies settings.json twice, as Reload Settings
// does: a key taken out goes back to its default.
func TestReloadSettings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOREX_DIR", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	themes := filepath.Join(dir, "ghostty", "themes")
	os.MkdirAll(themes, 0o755)
	os.WriteFile(filepath.Join(themes, "Ayu Mirage"), []byte(ayuMirage), 0o644)
	t.Cleanup(func() { applySettings(settings{}) })

	write := func(s string) {
		os.WriteFile(filepath.Join(dir, "settings.json"), []byte(s), 0o600)
	}
	write(`{"fontFamily": "Iosevka", "lineHeight": 1.2, "theme": "Ayu Mirage"}`)
	for range 2 {
		s, err := readSettings()
		if err != nil {
			t.Fatal(err)
		}
		if err := applySettings(s); err != nil {
			t.Fatal(err)
		}
	}
	if termFont.Family != "Iosevka, "+baseFont.Family || termFont.LineHeight != 1.2 || darkTerm == baseDarkTerm || themedDark == nil {
		t.Errorf("settings applied twice: %+v", termFont)
	}

	write(`{}`)
	s, _ := readSettings()
	applySettings(s)
	if termFont.Family != baseFont.Family || termFont.LineHeight != baseFont.LineHeight || darkTerm != baseDarkTerm || themedDark != nil {
		t.Errorf("settings taken out: %+v", termFont)
	}

	broken := `{"lineHeight": 1.2,}`
	write(broken)
	if _, err := readSettings(); err == nil {
		t.Error("a file that is not JSON read")
	}
	saveSetting("fontSize", float32(15))
	if b, _ := os.ReadFile(filepath.Join(dir, "settings.json")); string(b) != broken {
		t.Errorf("a file that is not JSON saved over: %s", b)
	}
}

// TestSaveSetting changes a key of settings.json and keeps the others,
// which the app has not read.
func TestSaveSetting(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOREX_DIR", dir)
	path := filepath.Join(dir, "settings.json")

	saveSetting("fontSize", float32(15))
	if s, err := readSettings(); err != nil || s.FontSize != 15 {
		t.Errorf("saved without a file %+v, %v", s, err)
	}
	os.WriteFile(path, []byte(`{"fontSize": 15, "lineHeight": 1.2, "hideSidebar": true}`), 0o600)
	saveSetting("fontSize", float32(16))
	saveSetting("hideSidebar", false)
	s, err := readSettings()
	if err != nil || s.FontSize != 16 || s.LineHeight != 1.2 || s.HideSidebar {
		t.Errorf("saved %+v, %v", s, err)
	}
	if b, _ := os.ReadFile(path); strings.Contains(string(b), "hideSidebar") {
		t.Errorf("false saved: %s", b)
	}

	os.WriteFile(path, nil, 0o600)
	saveSetting("appearance", "dark")
	if s, err := readSettings(); err != nil || s.Appearance != "dark" {
		t.Errorf("saved into a blank file %+v, %v", s, err)
	}
}
