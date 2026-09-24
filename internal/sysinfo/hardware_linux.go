package sysinfo

// System reads the DMI vendor and product name.
func System() SystemInfo {
	return SystemInfo{Manufacturer: dmi("sys_vendor"), Model: dmi("product_name"), Version: dmi("product_version")}
}

// Chassis reads the DMI chassis type.
func Chassis() ChassisInfo {
	t := dmiChassis()
	if t == "" {
		t = "Unknown"
	}
	return ChassisInfo{Manufacturer: dmi("chassis_vendor"), Type: t}
}
