package theme_test

import (
	"edex-ui-go/internal/theme"
	"testing"
)

func TestListThemes(t *testing.T) {
	themes := theme.ListThemes()
	if len(themes) == 0 {
		t.Fatal("Expected at least one bundled theme")
	}

	foundTron := false
	for _, name := range themes {
		if name == "tron" {
			foundTron = true
			break
		}
	}

	if !foundTron {
		t.Errorf("Expected bundled themes to include 'tron'")
	}
}

func TestLoadTheme(t *testing.T) {
	th, err := theme.LoadTheme("tron")
	if err != nil {
		t.Fatalf("Failed to load 'tron' theme: %v", err)
	}

	if th == nil {
		t.Fatal("Expected non-nil theme")
	}

	if th.Colors.Black == "" {
		t.Errorf("Expected black color to be specified in theme")
	}
}
