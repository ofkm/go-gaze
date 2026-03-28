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

	gofilewatch "go.ofkm.dev/gaze"
)

func main() {
	w, err := gofilewatch.WatchDirectory("my-directory")
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

	gofilewatch "go.ofkm.dev/gaze"
)

func main() {
	logger := slog.Default()
	cfg := gofilewatch.Config{
		ExcludeGlobs: []string{"*.tmp", "*.swp", ".DS_Store"},
		OnEvent: func(evt gofilewatch.Event) {
			fmt.Printf("%s %s\n", evt.Op, evt.Path)
		},
		OnError: func(err error) {
			logger.Error("watcher error", "err", err)
		},
	}

	w, err := gofilewatch.WatchDirectoryWithConfig("my-directory", cfg)
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
cfg := gofilewatch.Config{
	OnEvent: func(evt gofilewatch.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
}

w, err := gofilewatch.WatchFileWithConfig("config.yaml", cfg)
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

`WatchDirectory` is recursive by default. To watch only the top-level directory, set `Recursion: gofilewatch.RecursionDisabled`.

```go
cfg := gofilewatch.Config{
	Recursion: gofilewatch.RecursionDisabled,
	OnEvent: func(evt gofilewatch.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
}

w, err := gofilewatch.WatchDirectoryWithConfig("/srv/app", cfg)
if err != nil {
	panic(err)
}
```

## Follow symlinks intentionally

Symlink roots are rejected unless you opt in.

```go
cfg := gofilewatch.Config{
	FollowSymlinks: true,
}

w, err := gofilewatch.WatchDirectoryWithConfig("./relative/path", cfg)
if err != nil {
	panic(err)
}
```

## Logging fallback

If you omit `Config.OnEvent` and `Config.OnError`, Gaze logs normalized events and runtime errors with `Config.Logger`. If `Config.Logger` is nil, the package uses `slog.Default()`.
