// Package files provides the file system operations the UI used to perform
// through Node's fs module (file browser, text editor, settings editor).
package files

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
)

// Entry describes one directory entry, with its lstat information.
type Entry struct {
	Name      string `json:"name"`
	IsDir     bool   `json:"isDir"`
	IsFile    bool   `json:"isFile"`
	IsSymlink bool   `json:"isSymlink"`
	Size      int64  `json:"size"`
	Mtime     int64  `json:"mtime"`
	// Error holds a Node-style error code (EPERM, EBUSY...) when lstat failed.
	Error string `json:"error,omitempty"`
}

// ReadDir lists dir and lstats every entry in one call.
func ReadDir(dir string) ([]Entry, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(des))
	for _, de := range des {
		e := Lstat(filepath.Join(dir, de.Name()))
		e.Name = de.Name()
		entries = append(entries, e)
	}
	return entries, nil
}

// Lstat returns the lstat information of path.
func Lstat(path string) Entry {
	e := Entry{Name: filepath.Base(path)}
	info, err := os.Lstat(path)
	if err != nil {
		e.Error = errorCode(err)
		return e
	}
	mode := info.Mode()
	e.IsDir = mode.IsDir()
	e.IsFile = mode.IsRegular()
	e.IsSymlink = mode&fs.ModeSymlink != 0
	e.Size = info.Size()
	e.Mtime = info.ModTime().UnixMilli()
	return e
}

// errorCode maps an error to the Node.js error code the UI checks for.
func errorCode(err error) string {
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "EPERM"
	case errors.Is(err, fs.ErrNotExist):
		return "ENOENT"
	case errors.Is(err, syscall.EBUSY):
		return "EBUSY"
	}
	return "EIO"
}

// ReadFile returns the content of a text file.
func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}

// WriteFile replaces the content of a file, keeping its permissions.
func WriteFile(path, content string) error {
	perm := fs.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}
	return os.WriteFile(path, []byte(content), perm)
}

// Exists reports whether path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Watcher multiplexes directory watches identified by an id, like the
// objects returned by Node's fs.watch.
type Watcher struct {
	emit func(channel string, args ...any)

	mu      sync.Mutex
	next    int
	watches map[int]*fsnotify.Watcher
}

// NewWatcher creates a Watcher. emit receives ("fswatch-<id>", eventType,
// filename) where eventType is "rename" or "change" as in Node.
func NewWatcher(emit func(channel string, args ...any)) *Watcher {
	return &Watcher{emit: emit, watches: map[int]*fsnotify.Watcher{}}
}

// Watch starts watching dir and returns the watch id.
func (w *Watcher) Watch(dir string) (int, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return 0, err
	}
	if err := fw.Add(dir); err != nil {
		fw.Close()
		return 0, err
	}
	w.mu.Lock()
	w.next++
	id := w.next
	w.watches[id] = fw
	w.mu.Unlock()

	channel := "fswatch-" + strconv.Itoa(id)
	go func() {
		for {
			select {
			case ev, ok := <-fw.Events:
				if !ok {
					return
				}
				kind := "rename"
				if ev.Op&(fsnotify.Write|fsnotify.Chmod) != 0 && ev.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
					kind = "change"
				}
				w.emit(channel, kind, filepath.Base(ev.Name))
			case _, ok := <-fw.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return id, nil
}

// Unwatch stops the watch with the given id.
func (w *Watcher) Unwatch(id int) {
	w.mu.Lock()
	fw := w.watches[id]
	delete(w.watches, id)
	w.mu.Unlock()
	if fw != nil {
		fw.Close()
	}
}

// Close stops every watch.
func (w *Watcher) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for id, fw := range w.watches {
		fw.Close()
		delete(w.watches, id)
	}
}
