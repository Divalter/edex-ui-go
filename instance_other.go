//go:build !linux

package main

// forwardToRunningInstance is handled by the Wails single instance lock on
// the other platforms.
func forwardToRunningInstance(string) bool { return false }
