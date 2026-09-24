package hotkey

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// Register grabs the accelerator on the X11 root window. fn runs in a
// dedicated goroutine for every press (auto-repeat is ignored).
func Register(accel string, fn func(Event)) (Hotkey, error) {
	a, err := Parse(accel)
	if err != nil {
		return nil, err
	}
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" || (os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") != "") {
		return nil, fmt.Errorf("%w (Wayland session)", ErrUnsupported)
	}
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("connect to X server: %w", err)
	}
	h := &x11Hotkey{conn: conn, root: xproto.Setup(conn).DefaultScreen(conn).Root}
	h.keycode, err = h.keycodeFor(keysym(a.Key))
	if err != nil {
		conn.Close()
		return nil, err
	}
	h.mods = modMask(a)
	// Grab with every combination of Caps Lock and Num Lock, which the X
	// server treats as modifiers.
	for _, extra := range []uint16{0, xproto.ModMaskLock, xproto.ModMask2, xproto.ModMaskLock | xproto.ModMask2} {
		err := xproto.GrabKeyChecked(conn, true, h.root, h.mods|extra, h.keycode, xproto.GrabModeAsync, xproto.GrabModeAsync).Check()
		if err != nil {
			h.Close()
			var access xproto.AccessError
			if errors.As(err, &access) {
				return nil, fmt.Errorf("%s is already used by another application", accel)
			}
			return nil, fmt.Errorf("grab %s: %w", accel, err)
		}
	}
	go h.loop(fn)
	return h, nil
}

type x11Hotkey struct {
	conn    *xgb.Conn
	root    xproto.Window
	keycode xproto.Keycode
	mods    uint16

	once sync.Once
}

func (h *x11Hotkey) loop(fn func(Event)) {
	var last time.Time
	for {
		ev, err := h.conn.WaitForEvent()
		if ev == nil && err == nil {
			return // connection closed
		}
		press, ok := ev.(xproto.KeyPressEvent)
		if !ok || press.Detail != h.keycode {
			continue
		}
		// Holding the key sends repeated presses.
		if time.Since(last) < 300*time.Millisecond {
			last = time.Now()
			continue
		}
		last = time.Now()
		fn(Event{Time: uint32(press.Time)})
	}
}

func (h *x11Hotkey) Close() {
	h.once.Do(func() { h.conn.Close() })
}

// Activate asks the window manager to focus the window of this process with
// _NET_ACTIVE_WINDOW, marked as coming from a pager and carrying the key
// press timestamp so that focus stealing prevention allows it.
func (h *x11Hotkey) Activate(ev Event) {
	win, err := h.ownWindow()
	if err != nil {
		return
	}
	atom, err := h.atom("_NET_ACTIVE_WINDOW")
	if err != nil {
		return
	}
	msg := xproto.ClientMessageEvent{
		Format: 32,
		Window: win,
		Type:   atom,
		Data:   xproto.ClientMessageDataUnionData32New([]uint32{2, ev.Time, 0, 0, 0}),
	}
	xproto.SendEvent(h.conn, false, h.root,
		xproto.EventMaskSubstructureRedirect|xproto.EventMaskSubstructureNotify, string(msg.Bytes()))
}

func (h *x11Hotkey) atom(name string) (xproto.Atom, error) {
	r, err := xproto.InternAtom(h.conn, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, err
	}
	return r.Atom, nil
}

// ownWindow finds the top-level window whose _NET_WM_PID is this process.
func (h *x11Hotkey) ownWindow() (xproto.Window, error) {
	clientList, err := h.atom("_NET_CLIENT_LIST")
	if err != nil {
		return 0, err
	}
	pidAtom, err := h.atom("_NET_WM_PID")
	if err != nil {
		return 0, err
	}
	r, err := xproto.GetProperty(h.conn, false, h.root, clientList, xproto.AtomWindow, 0, 4096).Reply()
	if err != nil {
		return 0, err
	}
	pid := uint32(os.Getpid())
	for i := 0; i+4 <= len(r.Value); i += 4 {
		win := xproto.Window(xgb.Get32(r.Value[i:]))
		p, err := xproto.GetProperty(h.conn, false, win, pidAtom, xproto.AtomCardinal, 0, 1).Reply()
		if err == nil && len(p.Value) >= 4 && xgb.Get32(p.Value) == pid {
			return win, nil
		}
	}
	return 0, errors.New("window not found")
}

func (h *x11Hotkey) keycodeFor(sym uint32) (xproto.Keycode, error) {
	setup := xproto.Setup(h.conn)
	first, last := setup.MinKeycode, setup.MaxKeycode
	r, err := xproto.GetKeyboardMapping(h.conn, first, byte(last-first+1)).Reply()
	if err != nil {
		return 0, err
	}
	per := int(r.KeysymsPerKeycode)
	for i := 0; i*per < len(r.Keysyms); i++ {
		for j := 0; j < per; j++ {
			if uint32(r.Keysyms[i*per+j]) == sym {
				return first + xproto.Keycode(i), nil
			}
		}
	}
	return 0, fmt.Errorf("no key produces keysym %#x on this keyboard", sym)
}

func modMask(a Accelerator) uint16 {
	var m uint16
	if a.Shift {
		m |= xproto.ModMaskShift
	}
	if a.Ctrl {
		m |= xproto.ModMaskControl
	}
	if a.Alt {
		m |= xproto.ModMask1
	}
	if a.Super {
		m |= xproto.ModMask4
	}
	return m
}

// keysym maps a key name to its X keysym.
func keysym(key string) uint32 {
	if n := fNumber(key); n > 0 {
		return 0xffbe + uint32(n-1) // XK_F1
	}
	if len(key) == 1 {
		return uint32(key[0]) // Latin-1 keysyms: lowercase letters and digits
	}
	switch key {
	case "space":
		return 0x20
	case "grave":
		return 0x60
	case "tab":
		return 0xff09
	case "escape":
		return 0xff1b
	case "enter":
		return 0xff0d
	case "pause":
		return 0xff13
	case "scrolllock":
		return 0xff14
	}
	return 0
}
