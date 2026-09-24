//go:build !linux && !darwin && !windows

package sysinfo

// System reads the DMI vendor and product name when available.
func System() SystemInfo {
	return SystemInfo{Manufacturer: dmi("sys_vendor"), Model: dmi("product_name")}
}

// Chassis reads the DMI chassis type when available.
func Chassis() ChassisInfo {
	t := dmiChassis()
	if t == "" {
		t = "Unknown"
	}
	return ChassisInfo{Type: t}
}
