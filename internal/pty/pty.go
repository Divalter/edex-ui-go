// Package pty starts shells attached to a pseudo-terminal, replacing
// node-pty. Unix systems use creack/pty, Windows uses ConPTY.
package pty

import "io"

// PTY is a running shell attached to a pseudo-terminal.
type PTY interface {
	io.ReadWriteCloser
	// Resize changes the terminal window size.
	Resize(cols, rows uint16) error
	// Pid returns the shell process id.
	Pid() int
	// Wait blocks until the shell exits and returns its exit code.
	Wait() (int, error)
	// ForegroundPid returns the process group currently in the foreground
	// of the terminal (the running program), when supported.
	ForegroundPid() (int, error)
}

// Options describes the shell to start.
type Options struct {
	Shell string
	Args  []string
	Dir   string
	Env   []string
	Cols  uint16
	Rows  uint16
}
