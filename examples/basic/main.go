package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"go.ofkm.dev/gaze"
)

func main() {
	logger := slog.Default()

	target := "."
	if len(os.Args) > 1 {
		target = os.Args[1]
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		logger.Error("resolve target", "err", err)
		os.Exit(1)
	}

	cfg := gaze.Config{
		ExcludeGlobs:    []string{"*.tmp", "*.swp", ".DS_Store"},
		ExcludePrefixes: []string{filepath.Join(absTarget, ".git")},
		OnEvent: func(evt gaze.Event) {
			if evt.OldPath != "" {
				fmt.Printf("%s %s -> %s dir=%t\n", evt.Op, evt.OldPath, evt.Path, evt.IsDir)
				return
			}
			fmt.Printf("%s %s dir=%t\n", evt.Op, evt.Path, evt.IsDir)
		},
		OnError: func(err error) {
			logger.Error("watch error", "err", err)
		},
	}

	w, err := gaze.WatchDirectoryWithConfig(absTarget, cfg)
	if err != nil {
		logger.Error("watch directory", "path", absTarget, "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := w.Close(); err != nil {
			logger.Error("close watcher", "err", err)
		}
	}()

	fmt.Printf("watching %s\n", absTarget)
	fmt.Println("press Ctrl+C to stop")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	<-signals
	fmt.Println("stopping")
}
