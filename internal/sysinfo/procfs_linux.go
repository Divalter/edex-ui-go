package sysinfo

import (
	"bytes"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// clockTicks is USER_HZ, 100 on every Linux architecture Go supports.
const clockTicks = 100

var pageSize = uint64(os.Getpagesize())

// scanProcfs lists processes by reading a single file per process
// (/proc/<pid>/stat) plus the owner of /proc/<pid>. gopsutil reads about
// nine files per process, which made the process list the largest CPU cost
// of the backend. Must be called with procMu held.
func (s *SI) scanProcfs(totalMem uint64) (*Processes, error) {
	dir, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}
	names, err := dir.Readdirnames(-1)
	dir.Close()
	if err != nil {
		return nil, err
	}
	boot := bootTime()
	now := time.Now()
	wall := now.Sub(s.procWall).Seconds()
	firstScan := s.procWall.IsZero()
	ncpu := float64(runtime.NumCPU())
	samples := make(map[int32]procSample, len(names))

	res := &Processes{List: make([]Process, 0, len(names))}
	for _, name := range names {
		pid64, err := strconv.ParseInt(name, 10, 32)
		if err != nil {
			continue
		}
		pid := int32(pid64)
		data, err := os.ReadFile("/proc/" + name + "/stat")
		if err != nil {
			continue // process exited
		}
		open, close := bytes.IndexByte(data, '('), bytes.LastIndexByte(data, ')')
		if open < 0 || close < open || close+2 > len(data) {
			continue
		}
		f := strings.Fields(string(data[close+2:]))
		if len(f) < 22 {
			continue
		}
		num := func(i int) int64 { v, _ := strconv.ParseInt(f[i], 10, 64); return v }

		e := Process{PID: pid, Name: string(data[open+1 : close])}
		e.State = linuxState(f[0])
		e.ParentPID = int32(num(1))
		e.Priority = int32(num(15))
		e.Nice = int32(num(16))
		secs := float64(num(11)+num(12)) / clockTicks
		samples[pid] = procSample{cpuSeconds: secs}
		if prev, ok := s.procPrev[pid]; ok && !firstScan && wall > 0 {
			e.CPU = (secs - prev.cpuSeconds) / (wall * ncpu) * 100
			if e.CPU < 0 {
				e.CPU = 0
			}
		}
		e.MemVSZ = uint64(num(20)) / 1024
		rss := uint64(num(21)) * pageSize
		e.MemRSS = rss / 1024
		if totalMem > 0 {
			e.Mem = float64(rss) / float64(totalMem) * 100
		}
		if !boot.IsZero() {
			started := boot.Add(time.Duration(num(19)) * time.Second / clockTicks)
			e.Started = started.Format("2006-01-02 15:04:05")
		}
		if info, err := os.Stat("/proc/" + name); err == nil {
			if st, ok := info.Sys().(*syscall.Stat_t); ok {
				e.User = s.uidName(st.Uid)
			}
		}

		switch e.State {
		case "running":
			res.Running++
		case "blocked":
			res.Blocked++
		case "sleeping", "idle":
			res.Sleeping++
		default:
			res.Unknown++
		}
		res.List = append(res.List, e)
	}
	res.All = len(res.List)

	s.procPrev = samples
	s.procWall = now
	s.procCache = res
	s.procAt = now
	return res, nil
}

func linuxState(code string) string {
	switch code {
	case "R":
		return "running"
	case "S":
		return "sleeping"
	case "D":
		return "blocked"
	case "I":
		return "idle"
	case "Z":
		return "zombie"
	case "T", "t":
		return "stopped"
	}
	return "unknown"
}

var (
	bootOnce sync.Once
	bootAt   time.Time
)

// bootTime reads the btime line of /proc/stat.
func bootTime() time.Time {
	bootOnce.Do(func() {
		data, err := os.ReadFile("/proc/stat")
		if err != nil {
			return
		}
		for _, line := range strings.Split(string(data), "\n") {
			if v, ok := strings.CutPrefix(line, "btime "); ok {
				if secs, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
					bootAt = time.Unix(secs, 0)
				}
				return
			}
		}
	})
	return bootAt
}
