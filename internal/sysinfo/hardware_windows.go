package sysinfo

import (
	"golang.org/x/sys/windows/registry"
)

func biosValue(name string) string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\BIOS`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, err := k.GetStringValue(name)
	if err != nil {
		return ""
	}
	return v
}

// System reads the manufacturer and model from the BIOS registry key.
func System() SystemInfo {
	return SystemInfo{Manufacturer: biosValue("SystemManufacturer"), Model: biosValue("SystemProductName")}
}

// Chassis guesses the chassis type: the SMBIOS value is only reachable
// through WMI, which is too slow to poll.
func Chassis() ChassisInfo {
	t := "Desktop"
	if hasBattery() {
		t = "Laptop"
	}
	return ChassisInfo{Manufacturer: biosValue("SystemManufacturer"), Type: t}
}
