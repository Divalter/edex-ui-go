// Package terminal is the Go port of the "server" role of eDEX-UI's
// terminal.class.js: it spawns shells in pseudo-terminals, streams their
// output to a Sink (Wails events in the desktop app, a WebSocket in the
// browser development server) and reports the working directory and
// foreground process of each shell back to the UI.
package terminal

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"edex-ui-go/internal/pty"
)

// MaxExtraTTYs is the number of tabs that can be opened next to the main
// shell, like the original (4 extra terminals).
const MaxExtraTTYs = 4

// preConnectBufferLimit caps the output kept while no client is attached.
const preConnectBufferLimit = 256 * 1024

// Emitter delivers messages to the renderer, like Electron's
// webContents.send.
type Emitter interface {
	Emit(channel string, args ...any)
}

// Config describes how shells are started.
type Config struct {
	Shell    string
	Args     []string
	Cwd      string
	Env      []string
	BasePort int
}

// Manager owns the main shell and the extra tabs. TTYs are identified by a
// "port" number, kept from the original design where every terminal had its
// own WebSocket server.
type Manager struct {
	cfg    Config
	events Emitter

	mu   sync.Mutex
	ttys map[int]*TTY

	// OnMainExit is called when the main shell exits (the app should quit).
	OnMainExit func(code int)
}

// NewManager creates a manager. Call StartMain to spawn the main shell.
func NewManager(cfg Config, events Emitter) *Manager {
	return &Manager{cfg: cfg, events: events, ttys: map[int]*TTY{}}
}

// MainPort is the id of the main shell.
func (m *Manager) MainPort() int { return m.cfg.BasePort }

// StartMain spawns the main shell.
func (m *Manager) StartMain() error {
	t, err := m.start(m.cfg.BasePort, m.cfg.Cwd, true)
	if err != nil {
		return err
	}
	log.Printf("Terminal back-end initialized (pid %d)", t.pty.Pid())
	return nil
}

// Spawn opens a new terminal for an extra tab and returns its port. The new
// shell starts in the current directory of the main shell.
func (m *Manager) Spawn() (int, error) {
	m.mu.Lock()
	port := 0
	for i := 0; i < MaxExtraTTYs; i++ {
		p := m.cfg.BasePort + 2 + i
		if _, used := m.ttys[p]; !used {
			port = p
			m.ttys[p] = nil // reserve the slot
			break
		}
	}
	cwd := m.cfg.Cwd
	if main := m.ttys[m.cfg.BasePort]; main != nil {
		if c := main.CWD(); c != "" {
			cwd = c
		}
	}
	m.mu.Unlock()

	if port == 0 {
		return 0, errors.New("max number of ttys reached")
	}
	t, err := m.start(port, cwd, false)
	if err != nil {
		m.mu.Lock()
		delete(m.ttys, port)
		m.mu.Unlock()
		return 0, err
	}
	log.Printf("New terminal back-end initialized at %d (pid %d)", port, t.pty.Pid())
	return port, nil
}

func (m *Manager) start(port int, cwd string, main bool) (*TTY, error) {
	p, err := pty.Start(pty.Options{
		Shell: m.cfg.Shell,
		Args:  m.cfg.Args,
		Dir:   cwd,
		Env:   m.cfg.Env,
		Cols:  80,
		Rows:  24,
	})
	if err != nil {
		return nil, fmt.Errorf("spawn %s: %w", m.cfg.Shell, err)
	}
	t := &TTY{
		Port:        port,
		pty:         p,
		main:        main,
		fallbackCWD: cwd,
		events:      m.events,
		done:        make(chan struct{}),
		pumpDone:    make(chan struct{}),
	}
	m.mu.Lock()
	m.ttys[port] = t
	m.mu.Unlock()

	go t.pump()
	go t.track()
	go func() {
		code, _ := p.Wait()
		t.markClosed()
		// Let the remaining output reach the client before closing it.
		select {
		case <-t.pumpDone:
		case <-time.After(2 * time.Second):
		}
		t.closeConn()
		m.mu.Lock()
		delete(m.ttys, port)
		m.mu.Unlock()
		log.Printf("Terminal %d exited with code %d", port, code)
		if main && m.OnMainExit != nil {
			m.OnMainExit(code)
		}
	}()
	return t, nil
}

// Get returns the terminal with the given port, or nil.
func (m *Manager) Get(port int) *TTY {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ttys[port]
}

// CloseExtras kills the shells of the extra tabs, whose UI is gone after a
// reload.
func (m *Manager) CloseExtras() {
	m.mu.Lock()
	var ttys []*TTY
	for _, t := range m.ttys {
		if t != nil && !t.main {
			ttys = append(ttys, t)
		}
	}
	m.mu.Unlock()
	for _, t := range ttys {
		t.Close()
	}
}

// CloseAll kills every shell.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	ttys := make([]*TTY, 0, len(m.ttys))
	for _, t := range m.ttys {
		if t != nil {
			ttys = append(ttys, t)
		}
	}
	m.mu.Unlock()
	for _, t := range ttys {
		t.Close()
	}
}

// TTY is one running shell.
type TTY struct {
	Port int

	pty         pty.PTY
	main        bool
	fallbackCWD string
	events      Emitter

	// sendMu serializes output delivery; sinkMu guards sink and pending.
	sendMu  sync.Mutex
	sinkMu  sync.Mutex
	sink    Sink
	pending []byte

	stateMu    sync.Mutex
	cwd        string
	process    string
	noTracking bool

	dirty    atomic.Bool
	closed   atomic.Bool
	done     chan struct{}
	pumpDone chan struct{}
}

func (t *TTY) channel() string { return fmt.Sprintf("terminal_channel-%d", t.Port) }

// CWD returns the last known working directory of the shell.
func (t *TTY) CWD() string {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	return t.cwd
}

// pump copies the shell output to the attached client, buffering what is
// produced before a client connects.
func (t *TTY) pump() {
	defer close(t.pumpDone)
	buf := make([]byte, 32*1024)
	for {
		n, err := t.pty.Read(buf)
		if n > 0 {
			t.dirty.Store(true)
			t.send(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

func (t *TTY) send(data []byte) {
	t.sendMu.Lock()
	defer t.sendMu.Unlock()
	t.sinkMu.Lock()
	sink := t.sink
	if sink == nil {
		t.bufferLocked(data)
		t.sinkMu.Unlock()
		return
	}
	t.sinkMu.Unlock()

	// Send may block for flow control, so it runs without sinkMu held.
	if err := sink.Send(data); err != nil {
		t.sinkMu.Lock()
		if t.sink == sink {
			t.sink = nil
		}
		t.bufferLocked(data)
		t.sinkMu.Unlock()
	}
}

func (t *TTY) bufferLocked(data []byte) {
	t.pending = append(t.pending, data...)
	if over := len(t.pending) - preConnectBufferLimit; over > 0 {
		t.pending = t.pending[over:]
	}
}

// Attach makes s the destination of the terminal output, replacing (and
// closing) the previous one, e.g. after a UI reload. Output produced while
// no client was attached is delivered first.
func (t *TTY) Attach(s Sink) {
	t.sinkMu.Lock()
	old := t.sink
	t.sink = nil
	t.sinkMu.Unlock()
	if old != nil {
		old.Close() // unblocks a pending Send
	}

	t.sendMu.Lock()
	defer t.sendMu.Unlock()
	t.sinkMu.Lock()
	pending := t.pending
	t.pending = nil
	t.sink = s
	t.sinkMu.Unlock()
	if len(pending) > 0 {
		_ = s.Send(pending)
	}
}

// Detach removes s if it is still the current sink and reports whether it
// was. When the client of an extra tab goes away, the caller kills the
// shell, like the original ondisconnected handler.
func (t *TTY) Detach(s Sink) bool {
	t.sinkMu.Lock()
	current := t.sink == s
	if current {
		t.sink = nil
	}
	t.sinkMu.Unlock()
	if current {
		s.Close()
	}
	return current
}

// IsMain reports whether this is the main shell.
func (t *TTY) IsMain() bool { return t.main }

// Write sends input to the shell.
func (t *TTY) Write(p []byte) error {
	_, err := t.pty.Write(p)
	return err
}

// track polls the working directory and the foreground process once per
// second after the shell produced output, like the original _tick loop.
func (t *TTY) track() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.done:
			return
		case <-ticker.C:
		}
		if !t.dirty.Swap(false) {
			continue
		}
		t.updateCWD()
		t.updateProcess()
	}
}

func (t *TTY) updateCWD() {
	t.stateMu.Lock()
	if t.noTracking {
		t.stateMu.Unlock()
		return
	}
	t.stateMu.Unlock()

	cwd, err := processCWD(t.pty.Pid())
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	if err != nil {
		if t.closed.Load() {
			return
		}
		log.Printf("Error while tracking TTY working directory: %v", err)
		t.noTracking = true
		t.events.Emit(t.channel(), "Fallback cwd", t.fallbackCWD)
		return
	}
	if cwd == t.cwd {
		return
	}
	t.cwd = cwd
	t.events.Emit(t.channel(), "New cwd", cwd)
}

func (t *TTY) updateProcess() {
	name, err := foregroundProcess(t.pty)
	if err != nil {
		return
	}
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	if name == t.process {
		return
	}
	t.process = name
	t.events.Emit(t.channel(), "New process", name)
}

// RendererStartup answers the "Renderer startup" message of a client by
// resending the known working directory.
func (t *TTY) RendererStartup() {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	if t.noTracking {
		t.events.Emit(t.channel(), "Fallback cwd", t.fallbackCWD)
	} else if t.cwd != "" {
		t.events.Emit(t.channel(), "New cwd", t.cwd)
	}
	if t.process != "" {
		t.events.Emit(t.channel(), "New process", t.process)
	}
	// Force a refresh on the next tick in case nothing was known yet.
	t.dirty.Store(true)
}

// Resize resizes the pseudo-terminal.
func (t *TTY) Resize(cols, rows int) {
	if cols <= 0 || rows <= 0 {
		return
	}
	_ = t.pty.Resize(uint16(cols), uint16(rows))
}

// Close kills the shell.
func (t *TTY) Close() {
	t.markClosed()
	_ = t.pty.Close()
}

func (t *TTY) markClosed() {
	if t.closed.CompareAndSwap(false, true) {
		close(t.done)
	}
}

func (t *TTY) closeConn() {
	t.sinkMu.Lock()
	sink := t.sink
	t.sink = nil
	t.sinkMu.Unlock()
	if sink != nil {
		sink.Close()
	}
	t.events.Emit(fmt.Sprintf("tty_exit-%d", t.Port))
}
