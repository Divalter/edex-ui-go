// Package sysinfo replaces the systeminformation npm module used by eDEX-UI.
//
// Each exported method returns data with the same JSON field names as the
// corresponding systeminformation call, so the ported UI modules can consume
// it unchanged.
package sysinfo

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/sensors"
)

// SI holds the state needed to compute rates between calls.
type SI struct {
	cpuOnce   sync.Once
	cpuStatic CPU

	loadMu sync.Mutex

	procMu    sync.Mutex
	procCache *Processes
	procAt    time.Time
	procPrev  map[int32]procSample
	procWall  time.Time
	userCache map[string]string

	netMu   sync.Mutex
	netPrev map[string]netSample
}

// New creates an SI instance.
func New() *SI {
	return &SI{
		procPrev:  map[int32]procSample{},
		userCache: map[string]string{},
		netPrev:   map[string]netSample{},
	}
}

// Call dispatches a systeminformation-style call by name. args holds the
// JSON-encoded arguments of the original call.
func (s *SI) Call(method string, args []json.RawMessage) (any, error) {
	arg := func(i int) string {
		if i >= len(args) {
			return ""
		}
		var v string
		_ = json.Unmarshal(args[i], &v)
		return v
	}
	switch method {
	case "cpu":
		return s.CPU(), nil
	case "currentLoad":
		return s.CurrentLoad()
	case "cpuTemperature":
		return s.CPUTemperature(), nil
	case "mem":
		return s.Mem()
	case "processes":
		return s.Processes()
	case "battery":
		return Battery(), nil
	case "system":
		return System(), nil
	case "chassis":
		return Chassis(), nil
	case "time":
		return Time(), nil
	case "networkInterfaces":
		return NetworkInterfaces()
	case "networkStats":
		return s.NetworkStats(arg(0))
	case "networkConnections":
		return NetworkConnections()
	case "blockDevices":
		return BlockDevices()
	case "fsSize":
		return FsSize()
	}
	return nil, fmt.Errorf("sysinfo: unsupported call %q", method)
}

// CPU is the result of si.cpu().
type CPU struct {
	Manufacturer  string  `json:"manufacturer"`
	Brand         string  `json:"brand"`
	Vendor        string  `json:"vendor"`
	Speed         float64 `json:"speed"`
	SpeedMin      float64 `json:"speedMin"`
	SpeedMax      float64 `json:"speedMax"`
	Cores         int     `json:"cores"`
	PhysicalCores int     `json:"physicalCores"`
}

var (
	reFreq   = regexp.MustCompile(`@\s*[\d.]+\s*[GM]Hz`)
	reGen    = regexp.MustCompile(`^\d+(st|nd|rd|th) Gen `)
	reSpaces = regexp.MustCompile(`\s+`)
)

// cleanBrand reproduces how systeminformation splits the CPU model name
// into manufacturer and brand.
func cleanBrand(model string) (manufacturer, brand string) {
	b := strings.NewReplacer("(R)", "®", "(r)", "®", "(TM)", "™", "(tm)", "™", "(C)", "©", "CPU", "", "Processor", "").Replace(model)
	b = reFreq.ReplaceAllString(b, "")
	b = strings.TrimSpace(reSpaces.ReplaceAllString(b, " "))
	b = reGen.ReplaceAllString(b, "")
	parts := strings.SplitN(b, " ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return b, ""
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// CPU returns static CPU information plus the current average frequency.
func (s *SI) CPU() CPU {
	s.cpuOnce.Do(func() {
		c := CPU{}
		if infos, err := cpu.Info(); err == nil && len(infos) > 0 {
			c.Manufacturer, c.Brand = cleanBrand(infos[0].ModelName)
			c.Vendor = infos[0].VendorID
			c.SpeedMax = round2(infos[0].Mhz / 1000)
		}
		if n, err := cpu.Counts(true); err == nil {
			c.Cores = n
		}
		if n, err := cpu.Counts(false); err == nil {
			c.PhysicalCores = n
		}
		if c.Cores == 0 {
			c.Cores = runtime.NumCPU()
		}
		if mhz := maxFreqMHz(); mhz > 0 {
			c.SpeedMax = round2(mhz / 1000)
		}
		s.cpuStatic = c
	})
	c := s.cpuStatic
	c.Speed = c.SpeedMax
	if mhz := currentFreqMHz(); mhz > 0 {
		c.Speed = round2(mhz / 1000)
	}
	return c
}

// currentFreqMHz returns the average current frequency on Linux.
func currentFreqMHz() float64 {
	if runtime.GOOS != "linux" {
		return 0
	}
	return averageSysFreq("scaling_cur_freq")
}

func maxFreqMHz() float64 {
	if runtime.GOOS != "linux" {
		return 0
	}
	return averageSysFreq("cpuinfo_max_freq")
}

func averageSysFreq(file string) float64 {
	paths, _ := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*/cpufreq/" + file)
	var sum float64
	var n int
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		khz, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		if err != nil {
			continue
		}
		sum += khz / 1000
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// CoreLoad is one entry of currentLoad().cpus.
type CoreLoad struct {
	Load float64 `json:"load"`
}

// CurrentLoad is the result of si.currentLoad().
type CurrentLoad struct {
	CurrentLoad float64    `json:"currentLoad"`
	CPUs        []CoreLoad `json:"cpus"`
}

// CurrentLoad returns the per-core load since the previous call.
func (s *SI) CurrentLoad() (CurrentLoad, error) {
	s.loadMu.Lock()
	defer s.loadMu.Unlock()
	per, err := cpu.Percent(0, true)
	if err != nil {
		return CurrentLoad{}, err
	}
	res := CurrentLoad{CPUs: make([]CoreLoad, len(per))}
	var total float64
	for i, p := range per {
		res.CPUs[i].Load = p
		total += p
	}
	if len(per) > 0 {
		res.CurrentLoad = total / float64(len(per))
	}
	return res, nil
}

// Temperature is the result of si.cpuTemperature().
type Temperature struct {
	Main  *float64  `json:"main"`
	Max   *float64  `json:"max"`
	Cores []float64 `json:"cores"`
}

var cpuSensorKeys = []string{"coretemp", "k10temp", "zenpower", "cpu", "package", "tctl", "tdie", "soc"}

// CPUTemperature returns the CPU temperature, or nulls when unavailable.
func (s *SI) CPUTemperature() Temperature {
	res := Temperature{Cores: []float64{}}
	temps, _ := sensors.SensorsTemperatures()
	var cpuTemps, all []float64
	for _, t := range temps {
		if t.Temperature <= 0 || t.Temperature > 150 {
			continue
		}
		all = append(all, t.Temperature)
		key := strings.ToLower(t.SensorKey)
		for _, k := range cpuSensorKeys {
			if strings.Contains(key, k) {
				cpuTemps = append(cpuTemps, t.Temperature)
				break
			}
		}
	}
	if len(cpuTemps) == 0 {
		cpuTemps = all
	}
	if len(cpuTemps) == 0 {
		return res
	}
	sort.Float64s(cpuTemps)
	main := math.Round(cpuTemps[0])
	max := math.Round(cpuTemps[len(cpuTemps)-1])
	res.Main, res.Max = &main, &max
	return res
}

// Mem is the result of si.mem(). Like systeminformation, used is
// total - free and active is total - available.
type Mem struct {
	Total     uint64 `json:"total"`
	Free      uint64 `json:"free"`
	Used      uint64 `json:"used"`
	Active    uint64 `json:"active"`
	Available uint64 `json:"available"`
	Buffcache uint64 `json:"buffcache"`
	SwapTotal uint64 `json:"swaptotal"`
	SwapUsed  uint64 `json:"swapused"`
	SwapFree  uint64 `json:"swapfree"`
}

// Mem returns memory usage.
func (s *SI) Mem() (Mem, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return Mem{}, err
	}
	m := Mem{
		Total:     vm.Total,
		Free:      vm.Free,
		Used:      vm.Total - vm.Free,
		Available: vm.Available,
		Active:    vm.Total - vm.Available,
		Buffcache: vm.Buffers + vm.Cached,
	}
	if sw, err := mem.SwapMemory(); err == nil {
		m.SwapTotal, m.SwapUsed, m.SwapFree = sw.Total, sw.Used, sw.Free
	}
	return m, nil
}

// TimeInfo is the result of si.time().
type TimeInfo struct {
	Current  int64  `json:"current"`
	Uptime   uint64 `json:"uptime"`
	Timezone string `json:"timezone"`
}

// Time returns the current time and system uptime in seconds.
func Time() TimeInfo {
	up, _ := host.Uptime()
	zone, _ := time.Now().Zone()
	return TimeInfo{Current: time.Now().UnixMilli(), Uptime: up, Timezone: zone}
}
