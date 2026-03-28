package filewatch_test

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	gofilewatch "go.ofkm.dev/gaze"
)

func TestWatchDirectoryLifecycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	events := make(chan gofilewatch.Event, 32)
	errs := make(chan error, 32)
	cfg := gofilewatch.Config{
		OnEvent: func(evt gofilewatch.Event) {
			events <- evt
		},
		OnError: func(err error) {
			errs <- err
		},
	}

	w, err := gofilewatch.WatchDirectoryWithConfig(root, cfg)
	if err != nil {
		t.Fatalf("WatchDirectory() error = %v", err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	target := filepath.Join(root, "note.txt")
	if err := os.WriteFile(target, []byte("one"), 0o644); err != nil {
		t.Fatalf("WriteFile(create) error = %v", err)
	}
	waitForEvent(t, events, errs, func(evt gofilewatch.Event) bool {
		return evt.Path == target && evt.Op.Has(gofilewatch.OpCreate)
	})

	if err := os.WriteFile(target, []byte("two"), 0o644); err != nil {
		t.Fatalf("WriteFile(update) error = %v", err)
	}
	waitForEvent(t, events, errs, func(evt gofilewatch.Event) bool {
		return evt.Path == target && evt.Op.Has(gofilewatch.OpWrite)
	})

	if err := os.Remove(target); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	waitForEvent(t, events, errs, func(evt gofilewatch.Event) bool {
		return evt.Path == target && evt.Op.Has(gofilewatch.OpRemove)
	})
}

func TestWatchDirectoryRecursiveAndExclude(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	events := make(chan gofilewatch.Event, 32)
	errs := make(chan error, 32)
	cfg := gofilewatch.Config{
		ExcludeGlobs: []string{"*.tmp"},
		OnEvent: func(evt gofilewatch.Event) {
			events <- evt
		},
		OnError: func(err error) {
			errs <- err
		},
	}

	w, err := gofilewatch.WatchDirectoryWithConfig(root, cfg)
	if err != nil {
		t.Fatalf("WatchDirectory() error = %v", err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	subdir := filepath.Join(root, "nested")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	waitForEvent(t, events, errs, func(evt gofilewatch.Event) bool {
		return evt.Path == subdir && evt.IsDir && evt.Op.Has(gofilewatch.OpCreate)
	})

	nested := filepath.Join(subdir, "keep.txt")
	if err := os.WriteFile(nested, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile(nested) error = %v", err)
	}
	waitForEvent(t, events, errs, func(evt gofilewatch.Event) bool {
		return evt.Path == nested && evt.Op.Has(gofilewatch.OpCreate)
	})

	excluded := filepath.Join(root, "skip.tmp")
	if err := os.WriteFile(excluded, []byte("nope"), 0o644); err != nil {
		t.Fatalf("WriteFile(excluded) error = %v", err)
	}
	assertNoEvent(t, events, errs, func(evt gofilewatch.Event) bool {
		return evt.Path == excluded
	})
}

func TestWatchDirectoryNonRecursive(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	events := make(chan gofilewatch.Event, 32)
	errs := make(chan error, 32)
	cfg := gofilewatch.Config{
		Recursion: gofilewatch.RecursionDisabled,
		OnEvent: func(evt gofilewatch.Event) {
			events <- evt
		},
		OnError: func(err error) {
			errs <- err
		},
	}

	w, err := gofilewatch.WatchDirectoryWithConfig(root, cfg)
	if err != nil {
		t.Fatalf("WatchDirectoryWithConfig() error = %v", err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	subdir := filepath.Join(root, "nested")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	waitForEvent(t, events, errs, func(evt gofilewatch.Event) bool {
		return evt.Path == subdir && evt.IsDir && evt.Op.Has(gofilewatch.OpCreate)
	})

	nested := filepath.Join(subdir, "child.txt")
	if err := os.WriteFile(nested, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile(nested) error = %v", err)
	}
	assertNoEvent(t, events, errs, func(evt gofilewatch.Event) bool {
		return evt.Path == nested
	})
}

func TestWatchDirectoryOnEvent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	received := make(chan gofilewatch.Event, 8)
	errs := make(chan error, 8)
	cfg := gofilewatch.Config{
		OnEvent: func(evt gofilewatch.Event) {
			received <- evt
		},
		OnError: func(err error) {
			errs <- err
		},
	}

	w, err := gofilewatch.WatchDirectoryWithConfig(root, cfg)
	if err != nil {
		t.Fatalf("WatchDirectory() error = %v", err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	target := filepath.Join(root, "handler.txt")
	if err := os.WriteFile(target, []byte("one"), 0o644); err != nil {
		t.Fatalf("WriteFile(create) error = %v", err)
	}

	deadline := time.After(5 * time.Second)
	for {
		select {
		case evt := <-received:
			if evt.Path == target && evt.Op.Has(gofilewatch.OpCreate) {
				select {
				case err := <-errs:
					t.Fatalf("unexpected watcher error: %v", err)
				default:
				}
				return
			}
		case err := <-errs:
			t.Fatalf("watcher error: %v", err)
		case <-deadline:
			t.Fatal("timed out waiting for handler event")
		}
	}
}

func TestWatchDirectoryOnEventPanicBecomesError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	var once sync.Once
	errs := make(chan error, 8)
	cfg := gofilewatch.Config{
		OnEvent: func(gofilewatch.Event) {
			once.Do(func() {
				panic("boom")
			})
		},
		OnError: func(err error) {
			errs <- err
		},
	}

	w, err := gofilewatch.WatchDirectoryWithConfig(root, cfg)
	if err != nil {
		t.Fatalf("WatchDirectory() error = %v", err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	target := filepath.Join(root, "panic.txt")
	if err := os.WriteFile(target, []byte("one"), 0o644); err != nil {
		t.Fatalf("WriteFile(create) error = %v", err)
	}

	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("expected handler panic error, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for handler panic error")
	}
}

func waitForEvent(t *testing.T, events <-chan gofilewatch.Event, errs <-chan error, match func(gofilewatch.Event) bool) {
	t.Helper()

	deadline := time.After(5 * time.Second)
	for {
		select {
		case evt := <-events:
			if match(evt) {
				return
			}
		case err := <-errs:
			if err != nil && !errors.Is(err, os.ErrClosed) {
				t.Fatalf("watcher error: %v", err)
			}
		case <-deadline:
			t.Fatal("timed out waiting for matching event")
		}
	}
}

func assertNoEvent(t *testing.T, events <-chan gofilewatch.Event, errs <-chan error, match func(gofilewatch.Event) bool) {
	t.Helper()

	timeout := time.After(400 * time.Millisecond)
	for {
		select {
		case evt := <-events:
			if match(evt) {
				t.Fatalf("unexpected event: %+v", evt)
			}
		case err := <-errs:
			if err != nil && !errors.Is(err, os.ErrClosed) {
				t.Fatalf("watcher error: %v", err)
			}
		case <-timeout:
			return
		}
	}
}
