package hotkey

import "testing"

func TestParse(t *testing.T) {
	cases := map[string]Accelerator{
		"F12":         {Key: "f12"},
		"ctrl+alt+T":  {Ctrl: true, Alt: true, Key: "t"},
		"Super+`":     {Super: true, Key: "grave"},
		"Shift+Space": {Shift: true, Key: "space"},
		"F1":          {Key: "f1"},
	}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "F25", "F0", "Hyper+F12", "Ctrl+", "F012", "Ctrl+Up"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) should fail", bad)
		}
	}
}
