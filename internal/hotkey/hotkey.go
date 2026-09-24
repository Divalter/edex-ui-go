// Package hotkey registers system-wide keyboard shortcuts, used for the
// drop-down mode (show/hide eDEX-UI-GO from anywhere, like Guake or Quake's
// console).
//
// It is implemented for X11 and Windows. Wayland and macOS do not let an
// application grab keys globally: there, the desktop's own shortcut settings
// can run `edex-ui-go --toggle` instead.
package hotkey

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnsupported is returned where global shortcuts cannot be registered.
var ErrUnsupported = errors.New("global shortcuts are not supported on this system")

// Event describes a hotkey press.
type Event struct {
	// Time is the X server timestamp of the key press (X11 only), needed to
	// be allowed to take the focus.
	Time uint32
}

// Hotkey is a registered global shortcut.
type Hotkey interface {
	// Activate gives the keyboard focus to this process' window, using the
	// timestamp of ev (no-op where the platform does it by itself).
	Activate(ev Event)
	// Close unregisters the shortcut.
	Close()
}

// Accelerator is a parsed shortcut such as "F12" or "Ctrl+Alt+T".
type Accelerator struct {
	Ctrl, Shift, Alt, Super bool
	// Key is the key name in lowercase: "f1".."f24", "a".."z", "0".."9",
	// "space", "grave", "tab", "escape", "enter", "pause", "scrolllock".
	Key string
}

var keyAliases = map[string]string{
	"`": "grave", "backquote": "grave", "esc": "escape", "return": "enter",
	"scroll": "scrolllock", "scroll_lock": "scrolllock",
}

func validKey(k string) bool {
	switch {
	case len(k) == 1 && (k[0] >= 'a' && k[0] <= 'z' || k[0] >= '0' && k[0] <= '9'):
		return true
	case len(k) >= 2 && k[0] == 'f':
		var n int
		_, err := fmt.Sscanf(k[1:], "%d", &n)
		return err == nil && n >= 1 && n <= 24 && fmt.Sprint(n) == k[1:]
	}
	switch k {
	case "space", "grave", "tab", "escape", "enter", "pause", "scrolllock":
		return true
	}
	return false
}

// Parse parses an accelerator like the ones of shortcuts.json.
func Parse(s string) (Accelerator, error) {
	var a Accelerator
	parts := strings.Split(strings.TrimSpace(s), "+")
	for i, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if i < len(parts)-1 {
			switch p {
			case "ctrl", "control":
				a.Ctrl = true
			case "shift":
				a.Shift = true
			case "alt":
				a.Alt = true
			case "super", "meta", "win", "cmd", "command":
				a.Super = true
			default:
				return a, fmt.Errorf("hotkey %q: unknown modifier %q", s, p)
			}
			continue
		}
		if alias, ok := keyAliases[p]; ok {
			p = alias
		}
		if !validKey(p) {
			return a, fmt.Errorf("hotkey %q: unsupported key %q", s, p)
		}
		a.Key = p
	}
	return a, nil
}

// fNumber returns n for "fN" keys, or 0.
func fNumber(key string) int {
	var n int
	if len(key) >= 2 && key[0] == 'f' {
		fmt.Sscanf(key[1:], "%d", &n)
	}
	return n
}
