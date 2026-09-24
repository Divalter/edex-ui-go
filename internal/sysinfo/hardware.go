package sysinfo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/distatus/battery"
)

// SystemInfo is the result of si.system().
type SystemInfo struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Version      string `json:"version"`
}

// ChassisInfo is the result of si.chassis().
type ChassisInfo struct {
	Manufacturer string `json:"manufacturer"`
	Type         string `json:"type"`
}

// SMBIOS chassis types, as listed by systeminformation.
var chassisTypes = []string{
	"Other", "Unknown", "Desktop", "Low Profile Desktop", "Pizza Box", "Mini Tower", "Tower",
	"Portable", "Laptop", "Notebook", "Hand Held", "Docking Station", "All in One", "Sub Notebook",
	"Space-Saving", "Lunch Box", "Main System Chassis", "Expansion Chassis", "SubChassis",
	"Bus Expansion Chassis", "Peripheral Chassis", "Storage Chassis", "Rack Mount Chassis",
	"Sealed-Case PC", "Multi-System Chassis", "Compact PCI", "Advanced TCA", "Blade",
	"Blade Enclosure", "Tablet", "Convertible", "Detachable", "IoT Gateway", "Embedded PC",
	"Mini PC", "Stick PC",
}

func chassisName(code int) string {
	if code >= 1 && code <= len(chassisTypes) {
		return chassisTypes[code-1]
	}
	return "Unknown"
}

func readTrim(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func dmi(field string) string { return readTrim("/sys/class/dmi/id/" + field) }

func dmiChassis() string {
	code, err := strconv.Atoi(dmi("chassis_type"))
	if err != nil {
		return ""
	}
	return chassisName(code)
}

// BatteryInfo is the result of si.battery().
type BatteryInfo struct {
	HasBattery  bool    `json:"hasBattery"`
	IsCharging  bool    `json:"isCharging"`
	ACConnected bool    `json:"acConnected"`
	Percent     float64 `json:"percent"`
}

// Battery returns the state of the first battery.
func Battery() BatteryInfo {
	bats, _ := battery.GetAll()
	if len(bats) == 0 || bats[0] == nil || bats[0].Full <= 0 {
		return BatteryInfo{ACConnected: true}
	}
	b := bats[0]
	info := BatteryInfo{
		HasBattery: true,
		IsCharging: b.State.Raw == battery.Charging,
		Percent:    float64(int(b.Current/b.Full*100 + 0.5)),
	}
	info.ACConnected = b.State.Raw != battery.Discharging
	if ac, ok := linuxACOnline(); ok {
		info.ACConnected = ac
	}
	return info
}

// linuxACOnline reads the mains adapter state from sysfs.
func linuxACOnline() (online bool, found bool) {
	supplies, _ := filepath.Glob("/sys/class/power_supply/*")
	for _, s := range supplies {
		if readTrim(filepath.Join(s, "type")) != "Mains" {
			continue
		}
		found = true
		if readTrim(filepath.Join(s, "online")) == "1" {
			return true, true
		}
	}
	return false, found
}

// hasBattery is used to guess the chassis where it cannot be read.
func hasBattery() bool {
	bats, _ := battery.GetAll()
	return len(bats) > 0
}
