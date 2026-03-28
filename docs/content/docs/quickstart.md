---
title: "Quickstart"
description: "Install the package and start watching directories."
weight: 1
---

## Install

```bash
go get go.ofkm.dev/gaze
```

## Simplest watch

This is the lowest-friction path. The package owns the watcher goroutines and logs normalized events and internal errors with `slog.Default()`.

```go
package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go.ofkm.dev/gaze"
)

func main() {
	w, err := gaze.WatchDirectory("my-directory")
	if err != nil {
		slog.Default().Error("watch directory", "path", "my-directory", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := w.Close(); err != nil {
			slog.Default().Error("close watcher", "err", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}
```

## Structured config

When you need filters, callbacks, or logger control, use a plain struct literal with `WatchDirectoryWithConfig`.

```go
package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go.ofkm.dev/gaze"
)

func main() {
	logger := slog.Default()
	cfg := gaze.Config{
		ExcludeGlobs: []string{"*.tmp", "*.swp", ".DS_Store"},
		OnEvent: func(evt gaze.Event) {
			fmt.Printf("%s %s\n", evt.Op, evt.Path)
		},
		OnError: func(err error) {
			logger.Error("watcher error", "err", err)
		},
	}

	w, err := gaze.WatchDirectoryWithConfig("my-directory", cfg)
	if err != nil {
		logger.Error("watch directory", "path", "my-directory", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := w.Close(); err != nil {
			logger.Error("close watcher", "err", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}
```

## Watch a single file

```go
cfg := gaze.Config{
	OnEvent: func(evt gaze.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
}

w, err := gaze.WatchFileWithConfig("config.yaml", cfg)
if err != nil {
	panic(err)
}
defer func() {
	if err := w.Close(); err != nil {
		panic(err)
	}
}()
```

`WatchFile` and `WatchFileWithConfig` watch the file's parent directory and emit only events for the target file.

## Disable recursion explicitly

`WatchDirectory` is recursive by default. To watch only the top-level directory, set `Recursion: gaze.RecursionDisabled`.

```go
cfg := gaze.Config{
	Recursion: gaze.RecursionDisabled,
	OnEvent: func(evt gaze.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
}

w, err := gaze.WatchDirectoryWithConfig("/srv/app", cfg)
if err != nil {
	panic(err)
}
```

## Follow symlinks intentionally

Symlink roots are rejected unless you opt in.

```go
cfg := gaze.Config{
	FollowSymlinks: true,
}

w, err := gaze.WatchDirectoryWithConfig("./relative/path", cfg)
if err != nil {
	panic(err)
}
```

## Logging fallback

If you omit `Config.OnEvent` and `Config.OnError`, Gaze logs normalized events and runtime errors with `Config.Logger`. If `Config.Logger` is nil, the package uses `slog.Default()`.
