package sysinfo

import (
	"os"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// The procfs scanner must agree with gopsutil on the fields the UI shows.
func TestProcfsMatchesGopsutil(t *testing.T) {
	s := New()
	s.Processes()
	time.Sleep(200 * time.Millisecond)
	s.procAt = time.Time{} // bypass the cache
	res, err := s.Processes()
	if err != nil {
		t.Fatal(err)
	}
	if res.All < 2 {
		t.Fatalf("only %d processes", res.All)
	}
	var self *Process
	for i := range res.List {
		if int(res.List[i].PID) == os.Getpid() {
			self = &res.List[i]
		}
	}
	if self == nil {
		t.Fatal("own process not listed")
	}
	p, _ := process.NewProcess(int32(os.Getpid()))
	name, _ := p.Name()
	ppid, _ := p.Ppid()
	user, _ := p.Username()
	if self.Name != name || self.ParentPID != ppid || self.User != user {
		t.Errorf("procfs %q ppid %d user %q; gopsutil %q ppid %d user %q", self.Name, self.ParentPID, self.User, name, ppid, user)
	}
	ct, _ := p.CreateTime()
	started, err := time.ParseInLocation("2006-01-02 15:04:05", self.Started, time.Local)
	if err != nil || started.Sub(time.UnixMilli(ct)).Abs() > 2*time.Second {
		t.Errorf("start time %q vs %v", self.Started, time.UnixMilli(ct))
	}
	if self.MemRSS == 0 || self.State == "unknown" {
		t.Errorf("unexpected %+v", *self)
	}
}

func BenchmarkProcesses(b *testing.B) {
	b.Run("procfs", func(b *testing.B) { benchScan(b, true) })
	b.Run("gopsutil", func(b *testing.B) { benchScan(b, false) })
}

func benchScan(b *testing.B, procfs bool) {
	defer func(v bool) { useProcfs = v }(useProcfs)
	useProcfs = procfs
	s := New()
	for i := 0; i < b.N; i++ {
		s.procAt = time.Time{}
		if _, err := s.Processes(); err != nil {
			b.Fatal(err)
		}
	}
}
