//go:build !linux

package occlusion

import "time"

// Watch is not needed: WebView2 and WKWebView stop drawing covered windows.
func Watch(interval time.Duration, fn func(covered bool)) (stop func(), err error) {
	return nil, ErrUnsupported
}
