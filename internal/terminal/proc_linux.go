package terminal

import (
	"fmt"
	"os"
	"strings"

	"edex-ui-go/internal/pty"
)

func processCWD(pid int) (string, error) {
	return os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
}

func foregroundProcess(p pty.PTY) (string, error) {
	pgid, err := p.ForegroundPid()
	if err != nil {
		return "", err
	}
	comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pgid))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(comm)), nil
}
