---
title: "Quickstart"
description: "Install the package and start watching directories."
weight: 1
---

## Install

```bash
go get go.ofkm.dev/filewatch
```

## Minimal callback-based watch

```go
package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	gofilewatch "go.ofkm.dev/filewatch"
)

func main() {
	logger := slog.Default()
	sig := make(chan os.Signal, 1)

	w, err := gofilewatch.WatchDirectory(
		"my-directory",
		func(cfg *gofilewatch.Config) {
			cfg.OnEvent = func(evt gofilewatch.Event) {
				fmt.Printf("%s %s\n", evt.Op, evt.Path)
			}
			cfg.OnError = func(err error) {
				logger.Error("watcher error", "err", err)
			}
		},
	)
	if err != nil {
		logger.Error("watch directory", "path", "my-directory", "err", err)
		os.Exit(1)
	}
	defer w.Close()

	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}
```

## Minimal single-file watch

```go
w, err := gofilewatch.WatchFile("config.yaml")
```

`WatchFile` watches the file's parent directory and filters events so only the target file is emitted.

## Make recursion explicit

```go
w, err := gofilewatch.WatchDirectory(
	"/srv/app",
	func(cfg *gofilewatch.Config) {
		cfg.Recursive = false
		cfg.OnEvent = func(evt gofilewatch.Event) {
			fmt.Println(evt.Op, evt.Path)
		}
	},
)
```

`WatchDirectory` is recursive by default. Set `cfg.Recursive = false` in the configure callback when you want only the root directory level.

## Install path and follow-symlink behavior

```go
w, err := gofilewatch.WatchDirectory(
	"./relative/path",
	func(cfg *gofilewatch.Config) {
		cfg.FollowSymlinks = true
	},
)
```

- by default, symlink roots are rejected unless `cfg.FollowSymlinks = true` is set
- non-symlink roots are accepted either way

## Logging fallback when no handlers are set

If you do not provide `Config.OnEvent` or `Config.OnError`, the package logs all normalized events and errors using the configured `slog.Logger` (default: `slog.Default()`).
