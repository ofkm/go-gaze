package filewatch_test

import (
	"fmt"
	"os"

	gofilewatch "go.ofkm.dev/filewatch"
)

func ExampleWatchDirectory() {
	root, err := os.MkdirTemp("", "filewatch-example-*")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.RemoveAll(root)
	}()

	events := make(chan gofilewatch.Event, 1)

	w, err := gofilewatch.WatchDirectory(
		root,
		func(cfg *gofilewatch.Config) {
			cfg.ExcludeGlobs = []string{"*.tmp"}
			cfg.OnEvent = func(evt gofilewatch.Event) {
				select {
				case events <- evt:
				default:
				}
			}
		},
	)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = w.Close()
	}()

	select {
	case evt := <-events:
		fmt.Println(evt.Op, evt.Path)
	default:
	}
}
