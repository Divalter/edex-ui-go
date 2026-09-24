// Package buildinfo exposes version information.
package buildinfo

import (
	"runtime"
	"runtime/debug"
)

// Version is the eDEX-UI-GO version, set at build time with
// -ldflags "-X edex-ui-go/internal/buildinfo.Version=x.y.z".
var Version = "0.0.1"

// Runtime describes the Go and Wails versions, e.g. "Wails v2.16.0 / Go 1.25.0".
func Runtime(withWails bool) string {
	goVersion := "Go " + runtime.Version()[2:]
	if !withWails {
		return goVersion
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range bi.Deps {
			if dep.Path == "github.com/wailsapp/wails/v2" {
				return "Wails " + dep.Version + " / " + goVersion
			}
		}
	}
	return "Wails v2 / " + goVersion
}
