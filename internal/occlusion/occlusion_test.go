package occlusion

import "testing"

func TestCovered(t *testing.T) {
	win := Rect{100, 100, 800, 600}
	tests := []struct {
		name  string
		above []Rect
		want  bool
	}{
		{"nothing above", nil, false},
		{"maximized window", []Rect{{0, 0, 1920, 1080}}, true},
		{"elsewhere", []Rect{{1000, 0, 500, 500}}, false},
		{"partly", []Rect{{0, 0, 500, 1080}}, false},
		{"two halves", []Rect{{0, 0, 500, 1080}, {500, 0, 1420, 1080}}, true},
		{"four quarters with overlap", []Rect{{0, 0, 520, 420}, {480, 0, 600, 420}, {0, 380, 520, 400}, {480, 380, 600, 400}}, true},
		{"gap of one pixel", []Rect{{0, 0, 500, 1080}, {501, 0, 1420, 1080}}, false},
	}
	for _, tt := range tests {
		if got := Covered(win, tt.above); got != tt.want {
			t.Errorf("%s: Covered = %v, want %v", tt.name, got, tt.want)
		}
	}
}
