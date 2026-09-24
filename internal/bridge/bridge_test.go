package bridge

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRPCRequiresToken(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatal(err)
	}
	s.Handle("echo", func(args []json.RawMessage) (any, error) {
		var v string
		_ = Arg(args, 0, &v)
		return v, nil
	})
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	resp, err := http.Post(s.URL()+"/rpc/echo", "application/json", strings.NewReader(`["hi"]`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("request without token: status %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPost, s.URL()+"/rpc/echo", strings.NewReader(`["hi"]`))
	req.Header.Set("X-Edex-Token", s.Token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body rpcResponse
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body.Result != "hi" {
		t.Errorf("unexpected response %+v", body)
	}
}

func TestDispatchUnknown(t *testing.T) {
	s, _ := New()
	if _, err := s.Dispatch("nope", nil); err == nil {
		t.Error("expected error")
	}
}

func TestServeFileRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	_ = os.WriteFile(path, []byte("0123456789"), 0o644)
	s, _ := New()
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	req, _ := http.NewRequest(http.MethodGet, s.URL()+"/file?token="+s.Token+"&path="+path, nil)
	req.Header.Set("Range", "bytes=2-4")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 10)
	n, _ := resp.Body.Read(buf)
	if resp.StatusCode != http.StatusPartialContent || string(buf[:n]) != "234" {
		t.Errorf("range: status %d body %q", resp.StatusCode, buf[:n])
	}
}

func TestRejectsForeignOrigin(t *testing.T) {
	s, _ := New()
	s.Handle("ping", func([]json.RawMessage) (any, error) { return "pong", nil })
	if err := s.Start("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for origin, want := range map[string]int{"https://evil.example": http.StatusForbidden, s.URL(): http.StatusOK} {
		req, _ := http.NewRequest(http.MethodPost, s.URL()+"/rpc/ping", strings.NewReader(`[]`))
		req.Header.Set("X-Edex-Token", s.Token)
		req.Header.Set("Origin", origin)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Errorf("origin %s: status %d, want %d", origin, resp.StatusCode, want)
		}
	}
}
