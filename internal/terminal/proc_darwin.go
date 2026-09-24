package terminal

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"edex-ui-go/internal/pty"
)

// processCWD uses lsof, like the original implementation for macOS.
func processCWD(pid int) (string, error) {
	out, err := exec.Command("lsof", "-a", "-d", "cwd", "-p", strconv.Itoa(pid), "-Fn").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n") {
			return line[1:], nil
		}
	}
	return "", errors.New("cwd not found in lsof output")
}

func foregroundProcess(p pty.PTY) (string, error) {
	pgid, err := p.ForegroundPid()
	if err != nil {
		return "", err
	}
	out, err := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(pgid)).Output()
	if err != nil {
		return "", err
	}
	return filepath.Base(strings.TrimSpace(string(out))), nil
}
