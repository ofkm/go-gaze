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

	gofilewatch "go.ofkm.dev/filewatch"
)

func main() {
	logger := slog.Default()
	stop := make(chan os.Signal, 1)

	w, err := gofilewatch.WatchDirectory(
		".",
		func(cfg *gofilewatch.Config) {
			cfg.ExcludeGlobs = []string{"*.tmp", "*.swp", ".DS_Store"}
			cfg.ExcludePrefixes = []string{filepath.Join(".", ".git")}
			cfg.OnEvent = func(evt gofilewatch.Event) {
				fmt.Println(evt.Op, evt.Path)
			}
			cfg.OnError = func(err error) {
				logger.Error("watch error", "err", err)
			}
		},
	)
	if err != nil {
		logger.Error("watch directory", "err", err)
		os.Exit(1)
	}
	defer w.Close()

	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
```

## 2) Watch a single file

```go
w, err := gofilewatch.WatchFile("config.yaml")
if err != nil {
	panic(err)
}
defer w.Close()

// WatchFile inherits your handlers from the configure callback when provided.
```

## 3) Multi-root watcher with dynamic add/remove

```go
w, err := gofilewatch.New(
	func(cfg *gofilewatch.Config) {
		cfg.OnEvent = func(evt gofilewatch.Event) {
			fmt.Println(evt.Op, evt.Path)
		}
		cfg.OnError = func(err error) {
			fmt.Println("watch error:", err)
		}
	},
)
if err != nil {
	panic(err)
}
defer w.Close()

_ = w.Add("/srv/app/config")
_ = w.Add("/srv/app/templates")
_ = w.Remove("/srv/app/config")
```

## 4) Op filtering

```go
w, _ := gofilewatch.WatchDirectory(
	"my-directory",
	func(cfg *gofilewatch.Config) {
		cfg.Ops = gofilewatch.OpCreate | gofilewatch.OpRemove | gofilewatch.OpRename
		cfg.OnEvent = func(evt gofilewatch.Event) {
			fmt.Println("interesting:", evt.Op, evt.Path)
		}
	},
)
```

## 5) Custom logger only for fallback logs

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

w, _ := gofilewatch.WatchDirectory(
	"my-directory",
	func(cfg *gofilewatch.Config) {
		cfg.Logger = logger
	},
	// No handlers passed:
	// events and errors are logged with this logger
)
```

## 6) Follow symlinks intentionally

```go
w, err := gofilewatch.WatchDirectory(
	"link-or-tree-root",
	func(cfg *gofilewatch.Config) {
		cfg.FollowSymlinks = true
		cfg.OnEvent = func(evt gofilewatch.Event) {
			fmt.Println(evt.Op, evt.Path)
		}
	},
)
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
	// events may have been lost: resync subtree if possible
	fmt.Println("overflow detected, rebuild expected state")
}
```
