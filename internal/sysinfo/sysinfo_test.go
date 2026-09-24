package sysinfo

import (
	"encoding/json"
	"testing"
)

func TestCleanBrand(t *testing.T) {
	cases := map[string][2]string{
		"Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz": {"Intel®", "Core™ i7-8550U"},
		"AMD Ryzen 7 5800X 8-Core Processor":       {"AMD", "Ryzen 7 5800X 8-Core"},
		"12th Gen Intel(R) Core(TM) i5-1235U":      {"Intel®", "Core™ i5-1235U"},
	}
	for in, want := range cases {
		m, b := cleanBrand(in)
		if m != want[0] || b != want[1] {
			t.Errorf("cleanBrand(%q) = %q, %q; want %q, %q", in, m, b, want[0], want[1])
		}
	}
}

func TestParseHexAddr(t *testing.T) {
	ip, port := parseHexAddr("0100007F:0035", false)
	if ip != "127.0.0.1" || port != "53" {
		t.Errorf("got %s:%s", ip, port)
	}
	ip, _ = parseHexAddr("0000000000000000FFFF00000100007F:0050", true)
	if ip != "127.0.0.1" {
		t.Errorf("v4-mapped v6 address decoded as %s", ip)
	}
}

func TestMemInvariant(t *testing.T) {
	m, err := New().Mem()
	if err != nil {
		t.Skip(err)
	}
	// The RAM watcher module throws when this does not hold.
	if m.Free+m.Used != m.Total {
		t.Errorf("free+used != total: %+v", m)
	}
}

func TestCallShapes(t *testing.T) {
	s := New()
	for _, method := range []string{"cpu", "currentLoad", "cpuTemperature", "mem", "processes", "battery", "system", "chassis", "time", "networkInterfaces", "networkConnections", "blockDevices", "fsSize"} {
		res, err := s.Call(method, nil)
		if err != nil {
			t.Errorf("%s: %v", method, err)
			continue
		}
		if _, err := json.Marshal(res); err != nil {
			t.Errorf("%s: marshal: %v", method, err)
		}
	}
	if _, err := s.Call("nope", nil); err == nil {
		t.Error("expected error for unknown call")
	}
}
