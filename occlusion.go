package main

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"edex-ui-go/internal/occlusion"
)

// startOcclusion tells the UI when the window is entirely covered, so that
// it pauses its animations (see frontend/src/host/frameclock.js). Only X11
// needs it; the other platforms stop drawing covered webviews themselves.
func (h *Host) startOcclusion() {
	h.backend.Bridge().Handle("window.covered", func([]json.RawMessage) (any, error) {
		return h.covered.Load(), nil
	})
	stop, err := occlusion.Watch(500*time.Millisecond, func(covered bool) {
		h.covered.Store(covered)
		h.emit("window", map[bool]string{true: "covered", false: "visible"}[covered])
	})
	if err != nil {
		if !errors.Is(err, occlusion.ErrUnsupported) {
			log.Printf("Occlusion tracking unavailable: %v", err)
		}
		return
	}
	h.stopOcclusion = stop
}
