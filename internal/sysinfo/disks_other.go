//go:build !linux && !windows

package sysinfo

func blockDevices() ([]BlockDevice, error) {
	return partitionsAsBlockDevices()
}
