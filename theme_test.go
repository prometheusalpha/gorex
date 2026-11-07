package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

const ayuMirage = `palette = 0=#171b24
palette = 1=#ed8274
palette = 15=#ffffff
background = #1f2430
foreground = #cccac2
cursor-color = #ffcc66
selection-background = #409fff
`

func TestThemes(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	themes := filepath.Join(dir, "ghostty", "themes")
	os.MkdirAll(themes, 0o755)
	os.WriteFile(filepath.Join(themes, "Ayu Mirage"), []byte(ayuMirage), 0o644)
	os.WriteFile(filepath.Join(themes, "Paper"), []byte("background = fbfbf0\nforeground = 222222\n"), 0o644)

	light, dark, err := loadThemes("Ayu Mirage")
	if err != nil || light != dark {
		t.Fatalf("one theme: %v", err)
	}
	if dark.Background != ui.Hex("#1f2430") || dark.Palette[1] != ui.Hex("#ed8274") || dark.Palette[15] != ui.Hex("#ffffff") || dark.Cursor != ui.Hex("#ffcc66") {
		t.Errorf("Ayu Mirage %+v", dark)
	}
	light, dark, err = loadThemes("light:Paper, dark:Ayu Mirage")
	if err != nil || light.Background != ui.Hex("#fbfbf0") || dark.Background != ui.Hex("#1f2430") {
		t.Fatalf("light and dark: %+v %+v %v", light, dark, err)
	}
	if _, _, err := loadThemes("No Such Theme"); err == nil {
		t.Error("a missing theme loaded")
	}
	// A dark theme makes the window dark, whatever the system's appearance.
	if k := colorsAround(dark); k.text != darkColors.text || k.cardFocused.R != 0x1f {
		t.Errorf("colors around a dark theme %+v", k)
	}
	if k := colorsAround(light); k.text != lightColors.text {
		t.Error("colors around a light theme are dark")
	}
}

// TestGhosttyThemes reads every theme Ghostty ships, where it is installed.
func TestGhosttyThemes(t *testing.T) {
	dir := "/Applications/Ghostty.app/Contents/Resources/ghostty/themes"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skip("no Ghostty")
	}
	for _, e := range entries {
		if _, err := loadTheme(filepath.Join(dir, e.Name())); err != nil {
			t.Errorf("%s: %v", e.Name(), err)
		}
	}
	t.Logf("%d themes", len(entries))
}
