//go:build !linux

package sysinfo

import "errors"

// scanProcfs is only used on Linux.
func (s *SI) scanProcfs(uint64) (*Processes, error) {
	return nil, errors.New("procfs is not available")
}
