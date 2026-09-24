//go:build windows

package pty

import (
	"context"
	"errors"
	"strings"
	"syscall"

	"github.com/UserExistsError/conpty"
)

type winPTY struct {
	c *conpty.ConPty
}

// Start launches the shell described by opts using ConPTY.
func Start(opts Options) (PTY, error) {
	if !conpty.IsConPtyAvailable() {
		return nil, errors.New("ConPTY is not available (Windows 10 1809 or newer is required)")
	}
	parts := make([]string, 0, len(opts.Args)+1)
	parts = append(parts, syscall.EscapeArg(opts.Shell))
	for _, a := range opts.Args {
		parts = append(parts, syscall.EscapeArg(a))
	}
	c, err := conpty.Start(strings.Join(parts, " "),
		conpty.ConPtyDimensions(int(opts.Cols), int(opts.Rows)),
		conpty.ConPtyWorkDir(opts.Dir),
		conpty.ConPtyEnv(opts.Env),
	)
	if err != nil {
		return nil, err
	}
	return &winPTY{c: c}, nil
}

func (p *winPTY) Read(b []byte) (int, error)  { return p.c.Read(b) }
func (p *winPTY) Write(b []byte) (int, error) { return p.c.Write(b) }
func (p *winPTY) Pid() int                    { return p.c.Pid() }
func (p *winPTY) Close() error                { return p.c.Close() }

func (p *winPTY) Resize(cols, rows uint16) error {
	return p.c.Resize(int(cols), int(rows))
}

func (p *winPTY) Wait() (int, error) {
	code, err := p.c.Wait(context.Background())
	return int(code), err
}

func (p *winPTY) ForegroundPid() (int, error) {
	return 0, errors.New("foreground process tracking is not supported on Windows")
}
