// Package occlusion tells when the window of this process is entirely
// covered by other windows.
//
// Windows (WebView2), macOS (WKWebView) and Wayland compositors already stop
// drawing a webview nobody can see. On X11 with a compositing window manager
// every window is drawn off screen, so WebKitGTK keeps animating behind other
// windows: Watch lets the UI pause itself there.
package occlusion

import (
	"errors"
	"slices"
)

// ErrUnsupported is returned where the platform handles it by itself.
var ErrUnsupported = errors.New("occlusion tracking not needed on this platform")

// Rect is a window area in root coordinates.
type Rect struct{ X, Y, W, H int }

func (r Rect) empty() bool { return r.W <= 0 || r.H <= 0 }

func (r Rect) intersect(o Rect) Rect {
	x0, y0 := max(r.X, o.X), max(r.Y, o.Y)
	x1, y1 := min(r.X+r.W, o.X+o.W), min(r.Y+r.H, o.Y+o.H)
	return Rect{x0, y0, x1 - x0, y1 - y0}
}

// Covered reports whether the union of above covers all of target.
func Covered(target Rect, above []Rect) bool {
	if target.empty() {
		return true
	}
	var clipped []Rect
	xs, ys := []int{target.X, target.X + target.W}, []int{target.Y, target.Y + target.H}
	for _, r := range above {
		c := r.intersect(target)
		if c.empty() {
			continue
		}
		if c == target {
			return true
		}
		clipped = append(clipped, c)
		xs = append(xs, c.X, c.X+c.W)
		ys = append(ys, c.Y, c.Y+c.H)
	}
	if len(clipped) == 0 {
		return false
	}
	slices.Sort(xs)
	slices.Sort(ys)
	xs, ys = slices.Compact(xs), slices.Compact(ys)
	// Every cell of the grid made by the rectangle edges must be inside one.
	for i := 0; i+1 < len(xs); i++ {
		for j := 0; j+1 < len(ys); j++ {
			cx, cy := xs[i], ys[j]
			if !slices.ContainsFunc(clipped, func(r Rect) bool {
				return cx >= r.X && cx < r.X+r.W && cy >= r.Y && cy < r.Y+r.H
			}) {
				return false
			}
		}
	}
	return true
}
