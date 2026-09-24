package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/wailsapp/wails/v2/pkg/options"
)

// forwardToRunningInstance hands the command line to an already running
// instance and reports whether it did. It speaks the D-Bus protocol of the
// Wails single instance lock, but runs before the backend starts: Wails only
// checks the lock after launching OnStartup, and the backend (shell, config
// files) must not start in a process that is about to exit, such as
// `edex-ui-go --toggle`.
func forwardToRunningInstance(uniqueID string) bool {
	id := "wails_app_" + strings.ReplaceAll(strings.ReplaceAll(uniqueID, "-", "_"), ".", "_")
	name := "org." + id + ".SingleInstance"
	path := dbus.ObjectPath("/org/" + id + "/SingleInstance")

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return false
	}
	defer conn.Close()
	var hasOwner bool
	if err := conn.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, name).Store(&hasOwner); err != nil || !hasOwner {
		return false
	}
	wd, _ := os.Getwd()
	data, err := json.Marshal(options.SecondInstanceData{Args: os.Args[1:], WorkingDirectory: wd})
	if err != nil {
		return false
	}
	return conn.Object(name, path).Call(name+".SendMessage", 0, string(data)).Err == nil
}
