//go:build windows
// +build windows

package terminal

import (
	"errors"
	"os"
	"os/exec"
)

var ErrNotImplemented = errors.New("not implemented")

func startPTY(shell string, args []string, env []string, dir string, cols, rows uint16) (*os.File, *exec.Cmd, error) {
	return nil, nil, ErrNotImplemented
}

func resizePTY(f *os.File, cols, rows uint16) error {
	return ErrNotImplemented
}
