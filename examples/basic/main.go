package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	gofilewatch "go.ofkm.dev/filewatch"
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

	w, err := gofilewatch.WatchDirectory(
		absTarget,
		func(cfg *gofilewatch.Config) {
			cfg.ExcludeGlobs = []string{"*.tmp", "*.swp", ".DS_Store"}
			cfg.ExcludePrefixes = []string{filepath.Join(absTarget, ".git")}
			cfg.OnEvent = func(evt gofilewatch.Event) {
				if evt.OldPath != "" {
					fmt.Printf("%s %s -> %s dir=%t\n", evt.Op, evt.OldPath, evt.Path, evt.IsDir)
					return
				}
				fmt.Printf("%s %s dir=%t\n", evt.Op, evt.Path, evt.IsDir)
			}
			cfg.OnError = func(err error) {
				logger.Error("watch error", "err", err)
			}
		},
	)
	if err != nil {
		logger.Error("watch directory", "path", absTarget, "err", err)
		os.Exit(1)
	}
	defer w.Close()

	fmt.Printf("watching %s\n", absTarget)
	fmt.Println("press Ctrl+C to stop")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	<-signals
	fmt.Println("stopping")
}
