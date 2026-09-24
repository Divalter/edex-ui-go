// Package config manages the eDEX-UI-GO user data directory.
//
// It mirrors what eDEX-UI's _boot.js did inside Electron's userData folder:
// default settings.json / shortcuts.json / lastWindowState.json files, a
// mirror of the bundled themes, keyboard layouts and fonts (so users can edit
// or add their own), and the versions_log.json history.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// AppDirName is the name of the user data directory, the equivalent of
// Electron's userData folder for the original eDEX-UI.
const AppDirName = "eDEX-UI-GO"

// Paths holds every well-known location inside the user data directory.
type Paths struct {
	Dir             string `json:"settingsDir"`
	Themes          string `json:"themesDir"`
	Keyboards       string `json:"keyboardsDir"`
	Fonts           string `json:"fontsDir"`
	Settings        string `json:"settingsFile"`
	Shortcuts       string `json:"shortcutsFile"`
	LastWindowState string `json:"lastWindowStateFile"`
	VersionsLog     string `json:"versionsLogFile"`
	GeoIPCache      string `json:"geoIPCacheDir"`
}

// DefaultDir returns the platform user data directory.
func DefaultDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, AppDirName)
}

// NewPaths computes all paths relative to dir.
func NewPaths(dir string) Paths {
	return Paths{
		Dir:             dir,
		Themes:          filepath.Join(dir, "themes"),
		Keyboards:       filepath.Join(dir, "keyboards"),
		Fonts:           filepath.Join(dir, "fonts"),
		Settings:        filepath.Join(dir, "settings.json"),
		Shortcuts:       filepath.Join(dir, "shortcuts.json"),
		LastWindowState: filepath.Join(dir, "lastWindowState.json"),
		VersionsLog:     filepath.Join(dir, "versions_log.json"),
		GeoIPCache:      filepath.Join(dir, "geoIPcache"),
	}
}

// defaultSettings keeps the exact keys and order of the original settings.json.
type defaultSettings struct {
	Shell                     string  `json:"shell"`
	ShellArgs                 string  `json:"shellArgs"`
	Cwd                       string  `json:"cwd"`
	Keyboard                  string  `json:"keyboard"`
	Theme                     string  `json:"theme"`
	TermFontSize              int     `json:"termFontSize"`
	Audio                     bool    `json:"audio"`
	AudioVolume               float64 `json:"audioVolume"`
	DisableFeedbackAudio      bool    `json:"disableFeedbackAudio"`
	ClockHours                int     `json:"clockHours"`
	PingAddr                  string  `json:"pingAddr"`
	Port                      int     `json:"port"`
	Nointro                   bool    `json:"nointro"`
	Nocursor                  bool    `json:"nocursor"`
	ForceFullscreen           bool    `json:"forceFullscreen"`
	AllowWindowed             bool    `json:"allowWindowed"`
	ExcludeThreadsFromToplist bool    `json:"excludeThreadsFromToplist"`
	HideDotfiles              bool    `json:"hideDotfiles"`
	FsListView                bool    `json:"fsListView"`
	HideKeyboard              bool    `json:"hideKeyboard"`
	ExperimentalGlobeFeatures bool    `json:"experimentalGlobeFeatures"`
	ExperimentalFeatures      bool    `json:"experimentalFeatures"`
}

func newDefaultSettings(cwd string) defaultSettings {
	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "powershell.exe"
	}
	return defaultSettings{
		Shell:                     shell,
		Cwd:                       cwd,
		Keyboard:                  "en-US",
		Theme:                     "tron",
		TermFontSize:              15,
		Audio:                     true,
		AudioVolume:               1.0,
		ClockHours:                24,
		PingAddr:                  "1.1.1.1",
		Port:                      3000,
		ForceFullscreen:           true,
		ExcludeThreadsFromToplist: true,
	}
}

// Shortcut is one entry of shortcuts.json.
type Shortcut struct {
	Type      string `json:"type"`
	Trigger   string `json:"trigger"`
	Action    string `json:"action"`
	Linebreak bool   `json:"linebreak,omitempty"`
	Enabled   bool   `json:"enabled"`
}

// DefaultShortcuts are the original eDEX-UI keymap, plus KB_TOGGLE which
// shows or hides the on-screen keyboard.
func DefaultShortcuts() []Shortcut {
	return []Shortcut{
		{Type: "app", Trigger: "Ctrl+Shift+C", Action: "COPY", Enabled: true},
		{Type: "app", Trigger: "Ctrl+Shift+V", Action: "PASTE", Enabled: true},
		{Type: "app", Trigger: "Ctrl+Tab", Action: "NEXT_TAB", Enabled: true},
		{Type: "app", Trigger: "Ctrl+Shift+Tab", Action: "PREVIOUS_TAB", Enabled: true},
		{Type: "app", Trigger: "Ctrl+X", Action: "TAB_X", Enabled: true},
		{Type: "app", Trigger: "Ctrl+Shift+S", Action: "SETTINGS", Enabled: true},
		{Type: "app", Trigger: "Ctrl+Shift+K", Action: "SHORTCUTS", Enabled: true},
		{Type: "app", Trigger: "Ctrl+Shift+F", Action: "FUZZY_SEARCH", Enabled: true},
		{Type: "app", Trigger: "Ctrl+Shift+L", Action: "FS_LIST_VIEW", Enabled: true},
		{Type: "app", Trigger: "Ctrl+Shift+H", Action: "FS_DOTFILES", Enabled: true},
		{Type: "app", Trigger: "Ctrl+Shift+P", Action: "KB_PASSMODE", Enabled: true},
		{Type: "app", Trigger: "Ctrl+Shift+Alt+K", Action: "KB_TOGGLE", Enabled: true},
		{Type: "app", Trigger: "Ctrl+Shift+I", Action: "DEV_DEBUG", Enabled: false},
		{Type: "app", Trigger: "Ctrl+Shift+F5", Action: "DEV_RELOAD", Enabled: true},
		{Type: "shell", Trigger: "Ctrl+Shift+Alt+Space", Action: "neofetch", Linebreak: true, Enabled: false},
	}
}

// Init creates the user data directory, writes the default files that are
// missing and mirrors the bundled assets. assets must contain the "themes",
// "kb_layouts" and "fonts" directories.
func Init(p Paths, assets fs.FS, version string) error {
	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := writeJSONIfMissing(p.Settings, newDefaultSettings(p.Dir)); err != nil {
		return err
	}
	if err := writeJSONIfMissing(p.Shortcuts, DefaultShortcuts()); err != nil {
		return err
	}
	if err := writeJSONIfMissing(p.LastWindowState, map[string]bool{"useFullscreen": true}); err != nil {
		return err
	}

	// Copy default themes, keyboard layouts and fonts, overwriting older copies
	// like the original did on every launch.
	mirrors := map[string]string{"themes": p.Themes, "kb_layouts": p.Keyboards, "fonts": p.Fonts}
	for src, dst := range mirrors {
		if err := mirrorDir(assets, src, dst); err != nil {
			return fmt.Errorf("mirror %s: %w", src, err)
		}
	}

	return logVersion(p.VersionsLog, version)
}

func writeJSONIfMissing(path string, v any) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return writeJSON(path, v, "    ")
}

func writeJSON(path string, v any, indent string) error {
	data, err := json.MarshalIndent(v, "", indent)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func mirrorDir(assets fs.FS, src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := fs.ReadDir(assets, src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := fs.ReadFile(assets, src+"/"+e.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

type versionEntry struct {
	FirstSeen int64 `json:"firstSeen"`
	LastSeen  int64 `json:"lastSeen"`
}

func logVersion(path, version string) error {
	history := map[string]versionEntry{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &history)
	}
	now := time.Now().UnixMilli()
	entry, ok := history[version]
	if !ok {
		entry.FirstSeen = now
	}
	entry.LastSeen = now
	history[version] = entry
	return writeJSON(path, history, "  ")
}

// Settings is the raw content of settings.json. It is kept as a generic map
// so that unknown keys survive and the frontend sees exactly what is on disk.
type Settings map[string]any

// LoadSettings reads settings.json.
func LoadSettings(p Paths) (Settings, error) {
	var s Settings
	if err := readJSON(p.Settings, &s); err != nil {
		return nil, err
	}
	return s, nil
}

// LoadShortcuts reads shortcuts.json as raw JSON.
func LoadShortcuts(p Paths) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := readJSON(p.Shortcuts, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// LoadLastWindowState reads lastWindowState.json.
func LoadLastWindowState(p Paths) (map[string]any, error) {
	var s map[string]any
	if err := readJSON(p.LastWindowState, &s); err != nil {
		return nil, err
	}
	return s, nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return nil
}

// String returns the string value of key, or def when missing or empty.
func (s Settings) String(key, def string) string {
	if v, ok := s[key].(string); ok && v != "" && v != "undefined" {
		return v
	}
	return def
}

// Bool returns the boolean value of key, or def when missing.
func (s Settings) Bool(key string, def bool) bool {
	if v, ok := s[key].(bool); ok {
		return v
	}
	return def
}

// Int returns the numeric value of key, or def when missing.
func (s Settings) Int(key string, def int) int {
	if v, ok := s[key].(float64); ok {
		return int(v)
	}
	return def
}

// ShellArgs returns the shellArgs setting, which may be a string or an array.
func (s Settings) ShellArgs() []string {
	switch v := s["shellArgs"].(type) {
	case string:
		return strings.Fields(v)
	case []any:
		args := make([]string, 0, len(v))
		for _, a := range v {
			if str, ok := a.(string); ok {
				args = append(args, str)
			}
		}
		return args
	}
	return nil
}

// Env returns the custom environment override. The original accepted an
// object; anything else is ignored.
func (s Settings) Env() map[string]string {
	env := map[string]string{}
	if m, ok := s["env"].(map[string]any); ok {
		for k, v := range m {
			env[k] = fmt.Sprint(v)
		}
	}
	return env
}
