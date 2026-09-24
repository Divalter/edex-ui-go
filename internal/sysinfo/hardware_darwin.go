package sysinfo

import (
	"os/exec"
	"strings"
)

func sysctl(name string) string {
	out, err := exec.Command("sysctl", "-n", name).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// System returns the Mac model identifier.
func System() SystemInfo {
	return SystemInfo{Manufacturer: "Apple Inc.", Model: sysctl("hw.model")}
}

// Chassis guesses the chassis from the model identifier.
func Chassis() ChassisInfo {
	t := "Desktop"
	if strings.Contains(strings.ToLower(sysctl("hw.model")), "book") {
		t = "Laptop"
	}
	return ChassisInfo{Manufacturer: "Apple Inc.", Type: t}
}
