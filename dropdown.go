package main

import (
	"encoding/json"
	"errors"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"edex-ui-go/internal/bridge"
	"edex-ui-go/internal/hotkey"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// defaultDropdownHotkey toggles the window from anywhere, like Guake.
const defaultDropdownHotkey = "F12"

// dropdown implements the drop-down mode: a global hotkey (or
// `edex-ui-go --toggle`) hides the window when it is focused and brings it
// back on top otherwise.
type dropdown struct {
	mu                sync.Mutex
	hk                hotkey.Hotkey
	visible           bool
	focused           bool
	restoreFullscreen bool
}

// startDropdown registers the global hotkey configured by dropdownHotkey
// (an empty value disables it).
func (h *Host) startDropdown() {
	accel := defaultDropdownHotkey
	if v, ok := h.backend.Settings()["dropdownHotkey"].(string); ok {
		accel = strings.TrimSpace(v)
	}
	h.drop.visible, h.drop.focused, h.drop.restoreFullscreen = true, true, h.fullscreen
	h.backend.Bridge().Handle("window.focus", func(args []json.RawMessage) (any, error) {
		var focused bool
		_ = bridge.Arg(args, 0, &focused)
		h.drop.mu.Lock()
		h.drop.focused = focused
		h.drop.mu.Unlock()
		return nil, nil
	})
	if accel == "" {
		return
	}
	hk, err := hotkey.Register(accel, h.toggleDropdown)
	if err != nil {
		hint := ""
		if errors.Is(err, hotkey.ErrUnsupported) {
			hint = " — bind `edex-ui-go --toggle` to a shortcut in your desktop settings instead"
		}
		log.Printf("Drop-down hotkey %s unavailable: %v%s", accel, err, hint)
		return
	}
	h.drop.hk = hk
	log.Printf("Drop-down hotkey: %s", accel)
}

func (h *Host) stopDropdown() {
	if h.drop.hk != nil {
		h.drop.hk.Close()
	}
}

// toggleDropdown hides the window when it is visible and focused, and shows
// and focuses it otherwise.
func (h *Host) toggleDropdown(ev hotkey.Event) {
	ctx := h.context()
	if ctx == nil {
		return
	}
	h.drop.mu.Lock()
	defer h.drop.mu.Unlock()

	if h.drop.visible && h.drop.focused {
		h.drop.restoreFullscreen = runtime.WindowIsFullscreen(ctx)
		runtime.WindowHide(ctx)
		h.drop.visible, h.drop.focused = false, false
		return
	}

	wasHidden := !h.drop.visible
	runtime.WindowShow(ctx)
	runtime.WindowUnminimise(ctx) // raises the window (gtk_window_present)
	if wasHidden && h.drop.restoreFullscreen {
		runtime.WindowFullscreen(ctx)
	}
	h.drop.visible = true
	if hk := h.drop.hk; hk != nil {
		// The window is only listed by the window manager once mapped.
		go func() {
			for i := 0; i < 5; i++ {
				time.Sleep(60 * time.Millisecond)
				hk.Activate(ev)
			}
		}()
	}
	h.emit("dropdown", "show")
}

// secondInstance handles a new launch while eDEX-UI-GO is running.
func (h *Host) secondInstance(args []string) {
	if slices.ContainsFunc(args, func(a string) bool { return strings.EqualFold(a, "--toggle") }) {
		h.toggleDropdown(hotkey.Event{})
		return
	}
	if ctx := h.context(); ctx != nil {
		runtime.WindowShow(ctx)
		runtime.WindowUnminimise(ctx)
	}
}
