package filewatch_test

import (
	"fmt"
	"os"

	gofilewatch "go.ofkm.dev/gaze"
)

func ExampleWatchDirectory() {
	root, err := os.MkdirTemp("", "gaze-example-*")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.RemoveAll(root)
	}()

	events := make(chan gofilewatch.Event, 1)
	cfg := gofilewatch.Config{
		ExcludeGlobs: []string{"*.tmp"},
		OnEvent: func(evt gofilewatch.Event) {
			select {
			case events <- evt:
			default:
			}
		},
	}

	w, err := gofilewatch.WatchDirectoryWithConfig(root, cfg)
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
		fmt.Println(evt.Op, evt.Path)
	default:
	}
}
