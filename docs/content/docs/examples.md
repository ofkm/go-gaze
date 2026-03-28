---
title: "Examples"
description: "Runnable examples and common watch patterns."
weight: 5
---

## Runnable example module

The repository includes a runnable example:

```bash
cd examples/basic
go run . .
```

## 1) Basic callback watch

```go
package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	gofilewatch "go.ofkm.dev/gaze"
)

func main() {
	logger := slog.Default()
	stop := make(chan os.Signal, 1)

	cfg := gofilewatch.Config{
		ExcludeGlobs:    []string{"*.tmp", "*.swp", ".DS_Store"},
		ExcludePrefixes: []string{filepath.Join(".", ".git")},
		OnEvent: func(evt gofilewatch.Event) {
			fmt.Println(evt.Op, evt.Path)
		},
		OnError: func(err error) {
			logger.Error("watch error", "err", err)
		},
	}

	w, err := gofilewatch.WatchDirectoryWithConfig(".", cfg)
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

## 2) Watch a single file

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

## 3) Multi-root watcher with dynamic add/remove

```go
cfg := gofilewatch.Config{
	OnEvent: func(evt gofilewatch.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
	OnError: func(err error) {
		fmt.Println("watch error:", err)
	},
}

w, err := gofilewatch.NewWithConfig(cfg)
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

## 4) Op filtering

```go
cfg := gofilewatch.Config{
	Ops: gofilewatch.OpCreate | gofilewatch.OpRemove | gofilewatch.OpRename,
	OnEvent: func(evt gofilewatch.Event) {
		fmt.Println("interesting:", evt.Op, evt.Path)
	},
}

w, err := gofilewatch.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```

## 5) Logger-only fallback

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
cfg := gofilewatch.Config{
	Logger: logger,
}

w, err := gofilewatch.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```

Without handlers, Gaze writes events and internal errors to the configured logger.

## 6) Follow symlinks intentionally

```go
cfg := gofilewatch.Config{
	FollowSymlinks: true,
	OnEvent: func(evt gofilewatch.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
}

w, err := gofilewatch.WatchDirectoryWithConfig("link-or-tree-root", cfg)
if err != nil {
	panic(err)
}
```

## 7) Rename and overflow awareness

```go
if evt.Op&gofilewatch.OpRename != 0 {
	if evt.OldPath != "" {
		fmt.Printf("renamed %s -> %s\n", evt.OldPath, evt.Path)
	} else {
		fmt.Printf("rename-like change for %s\n", evt.Path)
	}
}

if evt.Op&gofilewatch.OpOverflow != 0 {
	fmt.Println("overflow detected, rebuild expected state")
}
```
