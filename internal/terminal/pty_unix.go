//go:build !windows
// +build !windows

package terminal

import (
	"os"
	"os/exec"
	"github.com/creack/pty"
)

func startPTY(shell string, args []string, env []string, dir string, cols, rows uint16) (*os.File, *exec.Cmd, error) {
	cmd := exec.Command(shell, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = env
	}

	sz := &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	}

	f, err := pty.StartWithSize(cmd, sz)
	if err != nil {
		return nil, nil, err
	}

	return f, cmd, nil
}

func resizePTY(f *os.File, cols, rows uint16) error {
	sz := &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	}
	return pty.Setsize(f, sz)
}
