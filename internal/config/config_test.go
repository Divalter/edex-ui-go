package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func testAssets() fstest.MapFS {
	return fstest.MapFS{
		"themes/tron.json":      {Data: []byte(`{"colors":{"r":170}}`)},
		"kb_layouts/en-US.json": {Data: []byte(`{}`)},
		"fonts/fira_mono.woff2": {Data: []byte("font")},
	}
}

func TestInitWritesDefaultsAndMirrors(t *testing.T) {
	p := NewPaths(t.TempDir())
	if err := Init(p, testAssets(), "0.1.0"); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSettings(p)
	if err != nil {
		t.Fatal(err)
	}
	if s.String("theme", "") != "tron" || s.Int("port", 0) != 3000 || s.String("cwd", "") != p.Dir {
		t.Errorf("unexpected defaults: %v", s)
	}
	for _, f := range []string{filepath.Join(p.Themes, "tron.json"), filepath.Join(p.Keyboards, "en-US.json"), filepath.Join(p.Fonts, "fira_mono.woff2")} {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("asset not mirrored: %v", err)
		}
	}
	raw, err := LoadShortcuts(p)
	if err != nil {
		t.Fatal(err)
	}
	var cuts []Shortcut
	if err := json.Unmarshal(raw, &cuts); err != nil || len(cuts) != len(DefaultShortcuts()) {
		t.Errorf("bad shortcuts: %v %v", err, cuts)
	}
}

func TestInitKeepsUserSettings(t *testing.T) {
	p := NewPaths(t.TempDir())
	if err := os.WriteFile(p.Settings, []byte(`{"theme":"matrix","custom":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Init(p, testAssets(), "0.1.0"); err != nil {
		t.Fatal(err)
	}
	s, _ := LoadSettings(p)
	if s.String("theme", "") != "matrix" || s["custom"] == nil {
		t.Errorf("user settings overwritten: %v", s)
	}
}

func TestDefaultSettingsKeyOrder(t *testing.T) {
	data, _ := json.Marshal(newDefaultSettings("/x"))
	if !strings.HasPrefix(string(data), `{"shell":`) {
		t.Errorf("settings.json should start with shell like the original: %s", data)
	}
}

func TestShellArgsAndEnv(t *testing.T) {
	s := Settings{"shellArgs": "-l -i", "env": map[string]any{"FOO": "bar", "N": 1.0}}
	if got := s.ShellArgs(); len(got) != 2 || got[1] != "-i" {
		t.Errorf("ShellArgs = %v", got)
	}
	s["shellArgs"] = []any{"--login"}
	if got := s.ShellArgs(); len(got) != 1 {
		t.Errorf("ShellArgs(array) = %v", got)
	}
	if env := s.Env(); env["FOO"] != "bar" || env["N"] != "1" {
		t.Errorf("Env = %v", env)
	}
	if (Settings{"env": "undefined"}).Env()["x"] != "" {
		t.Error("string env must be ignored")
	}
}
