package hotkey

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	procRegisterHotKey    = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey  = user32.NewProc("UnregisterHotKey")
	procGetMessage        = user32.NewProc("GetMessageW")
	procPostThreadMessage = user32.NewProc("PostThreadMessageW")
)

const (
	modAlt      = 0x1
	modControl  = 0x2
	modShift    = 0x4
	modWin      = 0x8
	modNoRepeat = 0x4000
	wmHotkey    = 0x0312
	wmQuit      = 0x0012
	hotkeyID    = 1
)

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

type winHotkey struct {
	threadID uint32
	once     sync.Once
}

// Register registers the accelerator with RegisterHotKey on a dedicated OS
// thread running a message loop.
func Register(accel string, fn func(Event)) (Hotkey, error) {
	a, err := Parse(accel)
	if err != nil {
		return nil, err
	}
	vk := virtualKey(a.Key)
	if vk == 0 {
		return nil, fmt.Errorf("hotkey %q: unsupported key", accel)
	}
	var mods uintptr = modNoRepeat
	if a.Alt {
		mods |= modAlt
	}
	if a.Ctrl {
		mods |= modControl
	}
	if a.Shift {
		mods |= modShift
	}
	if a.Super {
		mods |= modWin
	}

	h := &winHotkey{}
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		h.threadID = windows.GetCurrentThreadId()
		if r, _, e := procRegisterHotKey.Call(0, hotkeyID, mods, uintptr(vk)); r == 0 {
			ready <- fmt.Errorf("%s is already used by another application: %w", accel, e)
			return
		}
		defer procUnregisterHotKey.Call(0, hotkeyID)
		ready <- nil
		var m msg
		for {
			r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if int32(r) <= 0 {
				return
			}
			if m.message == wmHotkey && m.wParam == hotkeyID {
				go fn(Event{})
			}
		}
	}()
	if err := <-ready; err != nil {
		return nil, err
	}
	return h, nil
}

// Activate is a no-op: a process that received a hotkey may take the focus.
func (h *winHotkey) Activate(Event) {}

func (h *winHotkey) Close() {
	h.once.Do(func() {
		procPostThreadMessage.Call(uintptr(h.threadID), wmQuit, 0, 0)
	})
}

func virtualKey(key string) uint32 {
	if n := fNumber(key); n > 0 {
		return 0x70 + uint32(n-1) // VK_F1
	}
	if len(key) == 1 {
		c := key[0]
		if c >= 'a' && c <= 'z' {
			return uint32(c - 'a' + 'A')
		}
		return uint32(c) // '0'..'9'
	}
	switch key {
	case "space":
		return 0x20
	case "grave":
		return 0xc0 // VK_OEM_3 on US layouts
	case "tab":
		return 0x09
	case "escape":
		return 0x1b
	case "enter":
		return 0x0d
	case "pause":
		return 0x13
	case "scrolllock":
		return 0x91
	}
	return 0
}
