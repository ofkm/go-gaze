package gaze_test

import (
	"fmt"
	"os"

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

	w, err := gaze.WatchDirectoryWithConfig(root, cfg)
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
