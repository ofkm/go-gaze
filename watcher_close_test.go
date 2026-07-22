package gaze

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.ofkm.dev/gaze/internal/backend"
)

// TestCloseDrainsBufferedBackendEvents verifies that events already buffered
// in the backend channels when Close is called are still delivered to OnEvent
// before Close returns.
func TestCloseDrainsBufferedBackendEvents(t *testing.T) {
	prev := newBackend
	t.Cleanup(func() { newBackend = prev })

	driver := newStubDriver()
	newBackend = func(cfg backend.Config) (backend.Watcher, error) {
		return driver, nil
	}

	const n = 10
	got := make(chan Event, n)
	w, err := New(Config{OnEvent: func(evt Event) { got <- evt }})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	root := t.TempDir()
	if err := w.Add(root); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	for i := range n {
		driver.events <- backend.Event{
			Path: filepath.Join(root, fmt.Sprintf("file-%d.txt", i)),
			Op:   backend.OpCreate,
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	for i := range n {
		select {
		case <-got:
		case <-time.After(time.Second):
			t.Fatalf("only %d of %d buffered events delivered before Close returned", i, n)
		}
	}
}

// TestConfigReuseAcrossWatchers verifies that one Config value can create
// multiple independent watchers via its constructor methods.
func TestConfigReuseAcrossWatchers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "tracked.txt")
	if err := os.WriteFile(file, []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := Config{ExcludeGlobs: []string{"*.tmp"}, OnEvent: func(Event) {}}

	w1, err := cfg.WatchDirectory(dir)
	if err != nil {
		t.Fatalf("Config.WatchDirectory() error = %v", err)
	}
	w2, err := cfg.WatchFile(file)
	if err != nil {
		t.Fatalf("Config.WatchFile() error = %v", err)
	}
	w3, err := cfg.NewWatcher()
	if err != nil {
		t.Fatalf("Config.NewWatcher() error = %v", err)
	}

	if err := w3.Add(dir); err != nil {
		t.Fatalf("Add() on Config.NewWatcher() watcher error = %v", err)
	}

	for i, w := range []*Watcher{w1, w2, w3} {
		if err := w.Close(); err != nil {
			t.Fatalf("Close() watcher %d error = %v", i, err)
		}
	}
}
