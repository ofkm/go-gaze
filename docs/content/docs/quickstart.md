---
title: 'Quickstart'
description: 'Install the package and start watching directories.'
weight: 1
---

## Install

```bash
go get go.ofkm.dev/gaze
```

## Smallest useful example

This is the quickest way to start watching a directory. If you do not provide handlers, Gaze logs events and watcher errors with `slog.Default()`.

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

## Using `types.Config`

Use `WatchDirectoryWithConfig` with `go.ofkm.dev/gaze/types` when you want filters, callbacks, or your own logger.

```go
package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go.ofkm.dev/gaze"
	"go.ofkm.dev/gaze/types"
)

func main() {
	logger := slog.Default()
	cfg := types.Config{
		ExcludeGlobs: []string{"*.tmp", "*.swp", ".DS_Store"},
		OnEvent: func(evt types.Event) {
			fmt.Println(evt)
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

## Watching one file

```go
cfg := types.Config{
	OnEvent: func(evt types.Event) {
		fmt.Println(evt)
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

`WatchFile` and `WatchFileWithConfig` watch the parent directory and only deliver events for the file you asked for.

## Turning recursion off

`WatchDirectory` is recursive by default. If you only want the top level, set `Recursion: types.RecursionDisabled`.

```go
cfg := types.Config{
	Recursion: types.RecursionDisabled,
	OnEvent: func(evt types.Event) {
		fmt.Println(evt)
	},
}

w, err := gaze.WatchDirectoryWithConfig("/srv/app", cfg)
if err != nil {
	panic(err)
}
```

## Following symlinks

Symlink roots are rejected unless you opt in.

```go
cfg := types.Config{
	FollowSymlinks: true,
}

w, err := gaze.WatchDirectoryWithConfig("./relative/path", cfg)
if err != nil {
	panic(err)
}
```

## Logging behavior

If you leave out `types.Config.OnEvent` and `types.Config.OnError`, Gaze logs normalized events and runtime errors with `types.Config.Logger`. If `types.Config.Logger` is nil, it falls back to `slog.Default()`.
