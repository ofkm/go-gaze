---
title: 'Examples'
description: 'Runnable examples and common watch patterns.'
weight: 5
---

## Runnable example module

The repo includes a small runnable example in `example/main.go`:

```bash
cd example
go run . /path/to/watch
```

## Watch a user-supplied path

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"go.ofkm.dev/gaze"
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

	w, err := gaze.NewWithConfig(gaze.Config{
		Exclude: func(info gaze.PathInfo) bool {
			return info.IsDir && info.Base == ".git"
		},
		Ops: gaze.OpCreate | gaze.OpWrite | gaze.OpRemove | gaze.OpRename,
		OnEvent: func(evt gaze.Event) {
			fmt.Println(evt)
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

	fmt.Printf("GAZE[WATCH] %s\n", args[0])

	<-ctx.Done()
	fmt.Println("GAZE[STOP] shutting down")
	return 0
}
```

## Watch one file

```go
cfg := gaze.Config{
	OnEvent: func(evt gaze.Event) {
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

## Multi-root watcher

```go
cfg := gaze.Config{
	OnEvent: func(evt gaze.Event) {
		fmt.Println(evt)
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
		fmt.Println("interesting:", evt)
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
		fmt.Println(evt)
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
