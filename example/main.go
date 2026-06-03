package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"go.ofkm.dev/gaze"
	"go.ofkm.dev/gaze/types"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: %s PATH\n", filepath.Base(os.Args[0]))
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	w, err := gaze.New(types.Config{
		Exclude: func(info types.PathInfo) bool {
			return info.IsDir && info.Base == ".git"
		},
		Ops: types.OpCreate | types.OpWrite | types.OpRemove | types.OpRename,
		OnEvent: func(evt types.Event) {
			fmt.Fprintln(os.Stdout, evt)
		},
		OnError: func(err error) {
			fmt.Fprintln(os.Stderr, "GAZE[ERROR]", err)
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "GAZE[ERROR] create watcher:", err)
		return 1
	}
	defer func() {
		if err := w.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "GAZE[ERROR] close watcher:", err)
		}
	}()

	if err := w.Add(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "GAZE[ERROR] watch %q: %v\n", args[0], err)
		return 1
	}

	fmt.Fprintf(os.Stdout, "GAZE[WATCH] %s\n", args[0])

	<-ctx.Done()
	fmt.Fprintln(os.Stdout, "GAZE[STOP] shutting down")
	return 0
}
