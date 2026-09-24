//go:build !linux && !darwin

package terminal

import (
	"errors"
	"path/filepath"
	"strings"

	"edex-ui-go/internal/pty"

	"github.com/shirou/gopsutil/v4/process"
)

func processCWD(pid int) (string, error) {
	return "", errors.New("unsupported OS")
}

// foregroundProcess reports the newest descendant of the shell, since there
// is no foreground process group on Windows.
func foregroundProcess(p pty.PTY) (string, error) {
	proc, err := process.NewProcess(int32(p.Pid()))
	if err != nil {
		return "", err
	}
	for {
		children, err := proc.Children()
		if err != nil || len(children) == 0 {
			break
		}
		newest := children[0]
		var newestTime int64
		for _, c := range children {
			if t, err := c.CreateTime(); err == nil && t >= newestTime {
				newest, newestTime = c, t
			}
		}
		proc = newest
	}
	name, err := proc.Name()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(filepath.Base(name), ".exe"), nil
}
