package sysinfo

import (
	"os"
	"sort"

	"github.com/shirou/gopsutil/v4/disk"
)

// FsSizeEntry is one entry of si.fsSize().
type FsSizeEntry struct {
	FS        string  `json:"fs"`
	Type      string  `json:"type"`
	Size      uint64  `json:"size"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	Use       float64 `json:"use"`
	Mount     string  `json:"mount"`
}

// FsSize returns usage for every mounted filesystem, shortest mount point
// first: the file browser keeps the last mount that prefixes the cwd, so
// the most specific one wins.
func FsSize() ([]FsSizeEntry, error) {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	list := []FsSizeEntry{}
	for _, p := range parts {
		if seen[p.Mountpoint] {
			continue
		}
		u, err := disk.Usage(p.Mountpoint)
		if err != nil || u.Total == 0 {
			continue
		}
		seen[p.Mountpoint] = true
		list = append(list, FsSizeEntry{
			FS:        p.Device,
			Type:      p.Fstype,
			Size:      u.Total,
			Used:      u.Used,
			Available: u.Free,
			Use:       round2(u.UsedPercent),
			Mount:     p.Mountpoint,
		})
	}
	sort.SliceStable(list, func(i, j int) bool { return len(list[i].Mount) < len(list[j].Mount) })
	return list, nil
}

// BlockDevice is one entry of si.blockDevices().
type BlockDevice struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	FsType    string `json:"fsType"`
	Mount     string `json:"mount"`
	Label     string `json:"label"`
	Removable bool   `json:"removable"`
}

// BlockDevices lists the mounted block devices whose mount point exists
// (the original filtered them with fs.existsSync in the renderer).
func BlockDevices() ([]BlockDevice, error) {
	devs, err := blockDevices()
	if err != nil {
		return nil, err
	}
	list := []BlockDevice{}
	for _, d := range devs {
		if d.Mount == "" {
			continue
		}
		if _, err := os.Stat(d.Mount); err != nil {
			continue
		}
		list = append(list, d)
	}
	return list, nil
}

// partitionsAsBlockDevices is the portable fallback.
func partitionsAsBlockDevices() ([]BlockDevice, error) {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}
	list := make([]BlockDevice, 0, len(parts))
	for _, p := range parts {
		list = append(list, BlockDevice{Name: p.Device, Type: "part", FsType: p.Fstype, Mount: p.Mountpoint})
	}
	return list, nil
}
