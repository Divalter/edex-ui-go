package occlusion

import (
	"errors"
	"os"
	"sync"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// Watch polls the X11 window stack every interval and calls fn when the
// window of this process becomes covered or visible again. Covered means
// minimized, on another workspace, or entirely below other windows (their
// transparency, if any, is ignored). On Wayland it returns ErrUnsupported:
// compositors stop the frame callbacks of surfaces nobody sees.
func Watch(interval time.Duration, fn func(covered bool)) (stop func(), err error) {
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" || os.Getenv("DISPLAY") == "" {
		return nil, ErrUnsupported
	}
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}
	w := &watcher{conn: conn, root: xproto.Setup(conn).DefaultScreen(conn).Root, atoms: map[string]xproto.Atom{}}
	done := make(chan struct{})
	var once sync.Once
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		last := false
		for {
			select {
			case <-done:
				return
			case <-t.C:
			}
			covered, err := w.covered()
			if err != nil {
				continue // the window may not be mapped yet
			}
			if covered != last {
				last = covered
				fn(covered)
			}
		}
	}()
	return func() {
		once.Do(func() {
			close(done)
			conn.Close()
		})
	}, nil
}

type watcher struct {
	conn  *xgb.Conn
	root  xproto.Window
	atoms map[string]xproto.Atom
	own   xproto.Window
}

func (w *watcher) atom(name string) xproto.Atom {
	if a, ok := w.atoms[name]; ok {
		return a
	}
	r, err := xproto.InternAtom(w.conn, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0
	}
	w.atoms[name] = r.Atom
	return r.Atom
}

func (w *watcher) prop32(win xproto.Window, name string, typ xproto.Atom) []uint32 {
	r, err := xproto.GetProperty(w.conn, false, win, w.atom(name), typ, 0, 4096).Reply()
	if err != nil || r.Format != 32 {
		return nil
	}
	v := make([]uint32, len(r.Value)/4)
	for i := range v {
		v[i] = xgb.Get32(r.Value[i*4:])
	}
	return v
}

const allDesktops = 0xFFFFFFFF

func (w *watcher) covered() (bool, error) {
	stack := w.prop32(w.root, "_NET_CLIENT_LIST_STACKING", xproto.AtomWindow) // bottom to top
	idx := -1
	for i, win := range stack {
		if xproto.Window(win) == w.own {
			idx = i
		}
	}
	if idx < 0 {
		// Find the top-level window whose _NET_WM_PID is this process.
		pid := uint32(os.Getpid())
		for i, win := range stack {
			if p := w.prop32(xproto.Window(win), "_NET_WM_PID", xproto.AtomCardinal); len(p) == 1 && p[0] == pid {
				w.own, idx = xproto.Window(win), i
			}
		}
		if idx < 0 {
			return false, errors.New("window not found")
		}
	}

	if w.hidden(w.own) {
		return true, nil
	}
	current := w.prop32(w.root, "_NET_CURRENT_DESKTOP", xproto.AtomCardinal)
	onDesktop := func(win xproto.Window) bool {
		d := w.prop32(win, "_NET_WM_DESKTOP", xproto.AtomCardinal)
		return len(current) != 1 || len(d) != 1 || d[0] == allDesktops || d[0] == current[0]
	}
	if !onDesktop(w.own) {
		return true, nil
	}

	target, err := w.rect(w.own)
	if err != nil {
		return false, err
	}
	if root, err := xproto.GetGeometry(w.conn, xproto.Drawable(w.root)).Reply(); err == nil {
		target = target.intersect(Rect{0, 0, int(root.Width), int(root.Height)})
	}
	var above []Rect
	for _, win := range stack[idx+1:] {
		win := xproto.Window(win)
		if w.hidden(win) || !onDesktop(win) {
			continue
		}
		if r, err := w.rect(win); err == nil {
			above = append(above, r)
		}
	}
	return Covered(target, above), nil
}

func (w *watcher) hidden(win xproto.Window) bool {
	if a, err := xproto.GetWindowAttributes(w.conn, win).Reply(); err != nil || a.MapState != xproto.MapStateViewable {
		return true
	}
	hidden := w.atom("_NET_WM_STATE_HIDDEN")
	for _, s := range w.prop32(win, "_NET_WM_STATE", xproto.AtomAtom) {
		if xproto.Atom(s) == hidden {
			return true
		}
	}
	return false
}

// rect is the visible area of a top-level window in root coordinates,
// without the invisible shadow margins of client-side decorations.
func (w *watcher) rect(win xproto.Window) (Rect, error) {
	g, err := xproto.GetGeometry(w.conn, xproto.Drawable(win)).Reply()
	if err != nil {
		return Rect{}, err
	}
	t, err := xproto.TranslateCoordinates(w.conn, win, w.root, 0, 0).Reply()
	if err != nil {
		return Rect{}, err
	}
	r := Rect{int(t.DstX), int(t.DstY), int(g.Width), int(g.Height)}
	if e := w.prop32(win, "_GTK_FRAME_EXTENTS", xproto.AtomCardinal); len(e) == 4 { // left, right, top, bottom
		r = Rect{r.X + int(e[0]), r.Y + int(e[2]), r.W - int(e[0]) - int(e[1]), r.H - int(e[2]) - int(e[3])}
	}
	return r, nil
}
