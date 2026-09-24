package sysinfo

import (
	"os/exec"
	"regexp"
	"strings"
)

var lsblkPair = regexp.MustCompile(`([A-Z]+)="([^"]*)"`)

// blockDevices uses lsblk, like systeminformation.
func blockDevices() ([]BlockDevice, error) {
	out, err := exec.Command("lsblk", "-Pno", "NAME,TYPE,FSTYPE,MOUNTPOINT,LABEL,RM").Output()
	if err != nil {
		return partitionsAsBlockDevices()
	}
	var list []BlockDevice
	for _, line := range strings.Split(string(out), "\n") {
		kv := map[string]string{}
		for _, m := range lsblkPair.FindAllStringSubmatch(line, -1) {
			kv[m[1]] = unescapeLsblk(m[2])
		}
		if kv["NAME"] == "" || kv["TYPE"] == "loop" {
			continue
		}
		list = append(list, BlockDevice{
			Name:      kv["NAME"],
			Type:      kv["TYPE"],
			FsType:    kv["FSTYPE"],
			Mount:     kv["MOUNTPOINT"],
			Label:     kv["LABEL"],
			Removable: kv["RM"] == "1",
		})
	}
	return list, nil
}

// unescapeLsblk decodes the \xNN escapes lsblk uses in pair mode.
func unescapeLsblk(s string) string {
	if !strings.Contains(s, `\x`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) && s[i+1] == 'x' {
			var v byte
			ok := true
			for _, c := range s[i+2 : i+4] {
				v <<= 4
				switch {
				case c >= '0' && c <= '9':
					v |= byte(c - '0')
				case c >= 'a' && c <= 'f':
					v |= byte(c - 'a' + 10)
				case c >= 'A' && c <= 'F':
					v |= byte(c - 'A' + 10)
				default:
					ok = false
				}
			}
			if ok {
				b.WriteByte(v)
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
