//go:build !linux && !windows

package hotkey

// Register is not available: macOS requires the main thread, which belongs
// to the webview. Use the system shortcut settings to run
// `edex-ui-go --toggle`.
func Register(accel string, fn func(Event)) (Hotkey, error) {
	if _, err := Parse(accel); err != nil {
		return nil, err
	}
	return nil, ErrUnsupported
}
