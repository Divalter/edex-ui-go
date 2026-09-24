//go:build !windows

package pty

import (
	"errors"
	"os"
	"os/exec"
	"syscall"

	cpty "github.com/creack/pty"
	"golang.org/x/sys/unix"
)

type unixPTY struct {
	f   *os.File
	cmd *exec.Cmd
}

// Start launches the shell described by opts.
func Start(opts Options) (PTY, error) {
	cmd := exec.Command(opts.Shell, opts.Args...)
	cmd.Dir = opts.Dir
	cmd.Env = opts.Env
	f, err := cpty.StartWithSize(cmd, &cpty.Winsize{Cols: opts.Cols, Rows: opts.Rows})
	if err != nil {
		return nil, err
	}
	return &unixPTY{f: f, cmd: cmd}, nil
}

func (p *unixPTY) Read(b []byte) (int, error)  { return p.f.Read(b) }
func (p *unixPTY) Write(b []byte) (int, error) { return p.f.Write(b) }
func (p *unixPTY) Pid() int                    { return p.cmd.Process.Pid }

func (p *unixPTY) Resize(cols, rows uint16) error {
	return cpty.Setsize(p.f, &cpty.Winsize{Cols: cols, Rows: rows})
}

func (p *unixPTY) Wait() (int, error) {
	err := p.cmd.Wait()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	if err != nil {
		return -1, err
	}
	return 0, nil
}

func (p *unixPTY) ForegroundPid() (int, error) {
	return unix.IoctlGetInt(int(p.f.Fd()), unix.TIOCGPGRP)
}

func (p *unixPTY) Close() error {
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Signal(syscall.SIGHUP)
	}
	return p.f.Close()
}
