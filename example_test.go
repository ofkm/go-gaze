package gaze_test

import (
	"fmt"
	"os"
	"path/filepath"

	"go.ofkm.dev/gaze"
	"go.ofkm.dev/gaze/types"
)

func ExampleWatchDirectory() {
	root, err := os.MkdirTemp("", "gaze-example-*")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.RemoveAll(root)
	}()

	events := make(chan types.Event, 1)
	cfg := types.Config{
		ExcludeGlobs: []string{"*.tmp"},
		OnEvent: func(evt types.Event) {
			select {
			case events <- evt:
			default:
			}
		},
	}

	w, err := gaze.WatchDirectory(root, cfg)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			panic(err)
		}
	}()

	select {
	case evt := <-events:
		fmt.Println(evt)
	default:
	}
}

// ExampleWatchFile watches a single file. Gaze watches the file's parent
// directory non-recursively and delivers only events for that file.
func ExampleWatchFile() {
	dir, err := os.MkdirTemp("", "gaze-file-*")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("debug: false\n"), 0o644); err != nil {
		panic(err)
	}

	w, err := gaze.WatchFile(path, types.Config{
		OnEvent: func(evt types.Event) {
			fmt.Println("changed:", evt.Path)
		},
	})
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			panic(err)
		}
	}()
}

// ExampleNew watches several roots from one Watcher, enrolling and dropping
// them over time with Add and Remove.
func ExampleNew() {
	dir, err := os.MkdirTemp("", "gaze-multi-*")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	w, err := gaze.New(types.Config{
		OnError: func(err error) {
			fmt.Fprintln(os.Stderr, "gaze:", err)
		},
	})
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			panic(err)
		}
	}()

	if err := w.Add(dir); err != nil {
		panic(err)
	}
	// Later, stop watching that root.
	if err := w.Remove(dir); err != nil {
		panic(err)
	}
}
