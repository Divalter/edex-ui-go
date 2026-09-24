package files

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadDirStats(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hello"), 0o644)
	_ = os.Mkdir(filepath.Join(dir, "sub"), 0o755)
	_ = os.Symlink("f.txt", filepath.Join(dir, "link"))
	entries, err := ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Entry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	if e := byName["f.txt"]; !e.IsFile || e.Size != 5 {
		t.Errorf("file entry %+v", e)
	}
	if !byName["sub"].IsDir {
		t.Errorf("dir entry %+v", byName["sub"])
	}
	if !byName["link"].IsSymlink {
		t.Errorf("symlink entry %+v", byName["link"])
	}
}

func TestWriteKeepsPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.sh")
	_ = os.WriteFile(path, []byte("a"), 0o755)
	if err := WriteFile(path, "b"); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o755 {
		t.Errorf("mode changed to %v", info.Mode())
	}
}

func TestWatch(t *testing.T) {
	dir := t.TempDir()
	got := make(chan []any, 4)
	w := NewWatcher(func(channel string, args ...any) { got <- append([]any{channel}, args...) })
	defer w.Close()
	id, err := w.Watch(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "new"), nil, 0o644)
	select {
	case ev := <-got:
		if ev[0] != "fswatch-1" || id != 1 || ev[1] != "rename" || ev[2] != "new" {
			t.Errorf("unexpected event %v", ev)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no watch event")
	}
}
