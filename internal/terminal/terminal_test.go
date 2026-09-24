//go:build !windows

package terminal

import (
	"bytes"
	"encoding/base64"
	"strings"
	"sync"
	"testing"
	"time"
)

type recordEmitter struct {
	mu     sync.Mutex
	events []string
}

func (r *recordEmitter) Emit(channel string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, channel)
}

type bufSink struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *bufSink) Send(p []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.Write(p)
	return nil
}
func (s *bufSink) Close() {}
func (s *bufSink) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func newTestManager(t *testing.T) (*Manager, *recordEmitter) {
	events := &recordEmitter{}
	m := NewManager(Config{Shell: "/bin/sh", Cwd: t.TempDir(), Env: []string{"PS1=$ ", "TERM=dumb"}, BasePort: 3000}, events)
	if err := m.StartMain(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.CloseAll)
	return m, events
}

func TestShellRoundTrip(t *testing.T) {
	m, _ := newTestManager(t)
	tty := m.Get(3000)
	sink := &bufSink{}
	tty.Attach(sink)
	if err := tty.Write([]byte("echo edex-$((6*7))\n")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return strings.Contains(sink.String(), "edex-42") })
}

func TestOutputBufferedUntilAttach(t *testing.T) {
	m, _ := newTestManager(t)
	tty := m.Get(3000)
	_ = tty.Write([]byte("echo before-attach\n"))
	time.Sleep(300 * time.Millisecond)
	sink := &bufSink{}
	tty.Attach(sink)
	waitFor(t, func() bool { return strings.Contains(sink.String(), "before-attach") })
}

func TestSpawnLimitAndCloseExtras(t *testing.T) {
	m, _ := newTestManager(t)
	var ports []int
	for i := 0; i < MaxExtraTTYs; i++ {
		p, err := m.Spawn()
		if err != nil {
			t.Fatal(err)
		}
		ports = append(ports, p)
	}
	if ports[0] != 3002 || ports[3] != 3005 {
		t.Errorf("unexpected ports %v", ports)
	}
	if _, err := m.Spawn(); err == nil {
		t.Error("expected the fifth extra tty to be refused")
	}
	m.CloseExtras()
	waitFor(t, func() bool { return m.Get(3002) == nil && m.Get(3005) == nil })
	if m.Get(3000) == nil {
		t.Error("main tty must survive CloseExtras")
	}
}

func TestExitEvent(t *testing.T) {
	m, events := newTestManager(t)
	exited := make(chan struct{})
	m.OnMainExit = func(int) { close(exited) }
	_ = m.Get(3000).Write([]byte("exit\n"))
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatal("main shell exit not reported")
	}
	events.mu.Lock()
	defer events.mu.Unlock()
	if !strings.Contains(strings.Join(events.events, ","), "tty_exit-3000") {
		t.Errorf("missing tty_exit event, got %v", events.events)
	}
}

func TestEventSinkFlowControl(t *testing.T) {
	var mu sync.Mutex
	var got []byte
	sink := NewEventSink(func(b64 string) {
		data, _ := base64.StdEncoding.DecodeString(b64)
		mu.Lock()
		got = append(got, data...)
		mu.Unlock()
	})
	chunk := bytes.Repeat([]byte("x"), eventMaxBatch)

	// Fill up to the high water mark without acknowledging.
	for i := 0; i < eventHighWater/eventMaxBatch; i++ {
		if err := sink.Send(chunk); err != nil {
			t.Fatal(err)
		}
	}
	blocked := make(chan error)
	go func() { blocked <- sink.Send([]byte("more")) }()
	select {
	case <-blocked:
		t.Fatal("Send should block above the high water mark")
	case <-time.After(100 * time.Millisecond):
	}
	sink.Ack(eventHighWater)
	select {
	case err := <-blocked:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Send still blocked after Ack")
	}
	sink.Close()
	mu.Lock()
	defer mu.Unlock()
	if len(got) != eventHighWater+4 {
		t.Errorf("delivered %d bytes, want %d", len(got), eventHighWater+4)
	}
	if err := sink.Send([]byte("late")); err != ErrSinkClosed {
		t.Errorf("Send after Close = %v", err)
	}
}

func TestMergeEnv(t *testing.T) {
	env := MergeEnv([]string{"A=1", "TERM=dumb", "B=2"}, map[string]string{"TERM": "xterm-256color"})
	joined := strings.Join(env, " ")
	if strings.Contains(joined, "TERM=dumb") || !strings.Contains(joined, "TERM=xterm-256color") || !strings.Contains(joined, "A=1") {
		t.Errorf("unexpected env %v", env)
	}
}
