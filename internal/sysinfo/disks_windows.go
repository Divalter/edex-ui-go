package sysinfo

import (
	"golang.org/x/sys/windows"
)

// blockDevices lists drive letters with their type and volume label.
func blockDevices() ([]BlockDevice, error) {
	parts, err := partitionsAsBlockDevices()
	if err != nil {
		return nil, err
	}
	for i := range parts {
		root := parts[i].Mount + `\`
		rootPtr, err := windows.UTF16PtrFromString(root)
		if err != nil {
			continue
		}
		switch windows.GetDriveType(rootPtr) {
		case windows.DRIVE_REMOVABLE:
			parts[i].Removable = true
		case windows.DRIVE_CDROM:
			parts[i].Type = "rom"
		}
		label := make([]uint16, windows.MAX_PATH+1)
		if windows.GetVolumeInformation(rootPtr, &label[0], uint32(len(label)), nil, nil, nil, nil, 0) == nil {
			parts[i].Label = windows.UTF16ToString(label)
		}
	}
	return parts, nil
}
