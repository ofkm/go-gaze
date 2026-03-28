---
title: "Examples"
description: "Runnable examples and common watch patterns."
weight: 5
---

## Runnable example module

The repo includes a small runnable example:

```bash
cd examples/basic
go run .
```

## Basic directory watch

```go
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
	stop := make(chan os.Signal, 1)

	cfg := gaze.Config{
		ExcludeGlobs:    []string{"*.tmp", "*.swp", ".DS_Store"},
		ExcludePrefixes: []string{filepath.Join(".", ".git")},
		OnEvent: func(evt gaze.Event) {
			fmt.Println(evt.Op, evt.Path)
		},
		OnError: func(err error) {
			logger.Error("watch error", "err", err)
		},
	}

	w, err := gaze.WatchDirectoryWithConfig(".", cfg)
	if err != nil {
		logger.Error("watch directory", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := w.Close(); err != nil {
			logger.Error("close watcher", "err", err)
		}
	}()

	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
```

## Watch one file

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

## Multi-root watcher

```go
cfg := gaze.Config{
	OnEvent: func(evt gaze.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
	OnError: func(err error) {
		fmt.Println("watch error:", err)
	},
}

w, err := gaze.NewWithConfig(cfg)
if err != nil {
	panic(err)
}
defer func() {
	if err := w.Close(); err != nil {
		panic(err)
	}
}()

if err := w.Add("/srv/app/config"); err != nil {
	panic(err)
}
if err := w.Add("/srv/app/templates"); err != nil {
	panic(err)
}
if err := w.Remove("/srv/app/config"); err != nil {
	panic(err)
}
```

## Filter by operation

```go
cfg := gaze.Config{
	Ops: gaze.OpCreate | gaze.OpRemove | gaze.OpRename,
	OnEvent: func(evt gaze.Event) {
		fmt.Println("interesting:", evt.Op, evt.Path)
	},
}

w, err := gaze.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```

## Logger-only setup

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
cfg := gaze.Config{
	Logger: logger,
}

w, err := gaze.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```

If you leave out handlers, Gaze writes events and internal errors to the configured logger.

## Follow symlinks

```go
cfg := gaze.Config{
	FollowSymlinks: true,
	OnEvent: func(evt gaze.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
}

w, err := gaze.WatchDirectoryWithConfig("link-or-tree-root", cfg)
if err != nil {
	panic(err)
}
```

## Rename and overflow handling

```go
if evt.Op&gaze.OpRename != 0 {
	if evt.OldPath != "" {
		fmt.Printf("renamed %s -> %s\n", evt.OldPath, evt.Path)
	} else {
		fmt.Printf("rename-like change for %s\n", evt.Path)
	}
}

if evt.Op&gaze.OpOverflow != 0 {
	fmt.Println("overflow detected, rebuild expected state")
}
```
