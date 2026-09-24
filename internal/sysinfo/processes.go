package sysinfo

import (
	"os/user"
	"runtime"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

// procCacheTTL lets the several modules polling the process list (toplist,
// cpuinfo, process list modal) share one scan.
const procCacheTTL = 900 * time.Millisecond

// Process is one entry of si.processes().list.
type Process struct {
	PID       int32   `json:"pid"`
	ParentPID int32   `json:"parentPid"`
	Name      string  `json:"name"`
	CPU       float64 `json:"cpu"`
	Mem       float64 `json:"mem"`
	Priority  int32   `json:"priority"`
	MemRSS    uint64  `json:"memRss"`
	MemVSZ    uint64  `json:"memVsz"`
	Nice      int32   `json:"nice"`
	Started   string  `json:"started"`
	State     string  `json:"state"`
	User      string  `json:"user"`
	Command   string  `json:"command"`
}

// Processes is the result of si.processes().
type Processes struct {
	All      int       `json:"all"`
	Running  int       `json:"running"`
	Blocked  int       `json:"blocked"`
	Sleeping int       `json:"sleeping"`
	Unknown  int       `json:"unknown"`
	List     []Process `json:"list"`
}

// useProcfs selects the fast Linux scanner (see procfs_linux.go).
var useProcfs = runtime.GOOS == "linux"

type procSample struct {
	cpuSeconds float64
}

// Processes returns every process with its CPU usage (share of the whole
// machine since the previous scan) and memory usage (share of total RAM).
func (s *SI) Processes() (*Processes, error) {
	s.procMu.Lock()
	defer s.procMu.Unlock()
	if s.procCache != nil && time.Since(s.procAt) < procCacheTTL {
		return s.procCache, nil
	}

	var totalMem uint64
	if vm, err := mem.VirtualMemory(); err == nil {
		totalMem = vm.Total
	}
	if useProcfs {
		return s.scanProcfs(totalMem)
	}
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	wall := now.Sub(s.procWall).Seconds()
	firstScan := s.procWall.IsZero()
	ncpu := float64(runtime.NumCPU())
	samples := make(map[int32]procSample, len(procs))

	res := &Processes{List: make([]Process, 0, len(procs))}
	for _, p := range procs {
		name, err := p.Name()
		if err != nil {
			continue
		}
		e := Process{PID: p.Pid, Name: name}

		if t, err := p.Times(); err == nil {
			secs := t.User + t.System
			samples[p.Pid] = procSample{cpuSeconds: secs}
			if prev, ok := s.procPrev[p.Pid]; ok && !firstScan && wall > 0 {
				e.CPU = (secs - prev.cpuSeconds) / (wall * ncpu) * 100
				if e.CPU < 0 {
					e.CPU = 0
				}
			}
		}
		if m, err := p.MemoryInfo(); err == nil {
			e.MemRSS, e.MemVSZ = m.RSS/1024, m.VMS/1024
			if totalMem > 0 {
				e.Mem = float64(m.RSS) / float64(totalMem) * 100
			}
		}
		if ppid, err := p.Ppid(); err == nil {
			e.ParentPID = ppid
		}
		if nice, err := p.Nice(); err == nil {
			e.Nice = nice
		}
		if ct, err := p.CreateTime(); err == nil {
			e.Started = time.UnixMilli(ct).Format("2006-01-02 15:04:05")
		}
		if st, err := p.Status(); err == nil && len(st) > 0 {
			e.State = mapState(st[0])
		} else {
			e.State = "unknown"
		}
		e.User = s.username(p)
		if cmd, err := p.Cmdline(); err == nil {
			e.Command = cmd
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

func mapState(st string) string {
	switch st {
	case process.Running:
		return "running"
	case process.Sleep:
		return "sleeping"
	case process.Idle:
		return "idle"
	case process.Stop:
		return "stopped"
	case process.Zombie:
		return "zombie"
	case process.Wait, process.Lock:
		return "blocked"
	}
	return "unknown"
}

// username resolves the owner of a process, caching uid lookups.
func (s *SI) username(p *process.Process) string {
	if runtime.GOOS == "windows" {
		name, _ := p.Username()
		return name
	}
	uids, err := p.Uids()
	if err != nil || len(uids) == 0 {
		return ""
	}
	return s.uidName(uids[0])
}

// uidName resolves a user id, caching the lookups.
func (s *SI) uidName(id uint32) string {
	uid := strconv.FormatUint(uint64(id), 10)
	if name, ok := s.userCache[uid]; ok {
		return name
	}
	name := uid
	if u, err := user.LookupId(uid); err == nil {
		name = u.Username
	}
	s.userCache[uid] = name
	return name
}
