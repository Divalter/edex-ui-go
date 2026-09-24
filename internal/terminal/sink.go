package terminal

import (
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

// Sink receives the output of a terminal.
type Sink interface {
	// Send delivers data. It may block to apply backpressure.
	Send(data []byte) error
	// Close releases the sink and unblocks a pending Send.
	Close()
}

// ErrSinkClosed is returned by Send after Close.
var ErrSinkClosed = errors.New("terminal sink closed")

const (
	eventFlushDelay = 5 * time.Millisecond
	eventMaxBatch   = 64 * 1024
	// eventHighWater is the amount of unacknowledged output after which the
	// shell output stops being read, so that a program flooding the terminal
	// cannot exhaust the webview memory.
	eventHighWater = 1024 * 1024
)

// EventSink delivers output as base64 encoded messages through an event
// emitter (the Wails IPC). Messages are coalesced for a few milliseconds
// and the UI acknowledges the bytes it consumed with Ack.
type EventSink struct {
	emit func(b64 string)

	mu       sync.Mutex
	cond     *sync.Cond
	buf      []byte
	inflight int
	timer    *time.Timer
	closed   bool
}

// NewEventSink creates an EventSink calling emit for every batch.
func NewEventSink(emit func(b64 string)) *EventSink {
	s := &EventSink{emit: emit}
	s.cond = sync.NewCond(&s.mu)
	return s
}

// Send queues data, blocking while too much output is unacknowledged.
func (s *EventSink) Send(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for !s.closed && s.inflight >= eventHighWater {
		s.cond.Wait()
	}
	if s.closed {
		return ErrSinkClosed
	}
	s.buf = append(s.buf, data...)
	if len(s.buf) >= eventMaxBatch {
		s.flushLocked()
	} else if s.timer == nil {
		s.timer = time.AfterFunc(eventFlushDelay, s.flush)
	}
	return nil
}

func (s *EventSink) flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushLocked()
}

func (s *EventSink) flushLocked() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if len(s.buf) == 0 || s.closed {
		return
	}
	s.inflight += len(s.buf)
	s.emit(base64.StdEncoding.EncodeToString(s.buf))
	s.buf = nil
}

// Ack marks n bytes as consumed by the UI.
func (s *EventSink) Ack(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inflight -= n
	if s.inflight < 0 {
		s.inflight = 0
	}
	s.cond.Broadcast()
}

// Close implements Sink.
func (s *EventSink) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushLocked()
	s.closed = true
	s.cond.Broadcast()
}
