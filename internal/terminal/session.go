package terminal

import (
	"io"
	"os"
	"os/exec"
	"sync"
)

type Session struct {
	ID  string
	pty *os.File
	cmd *exec.Cmd
	mu  sync.Mutex
}

func (s *Session) Read(p []byte) (n int, err error) {
	if s.pty == nil {
		return 0, io.EOF
	}
	return s.pty.Read(p)
}

func (s *Session) Write(p []byte) (n int, err error) {
	if s.pty == nil {
		return 0, io.EOF
	}
	return s.pty.Write(p)
}

func (s *Session) Resize(cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pty == nil {
		return nil
	}
	return resizePTY(s.pty, cols, rows)
}

func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	if s.pty != nil {
		return s.pty.Close()
	}
	return nil
}
