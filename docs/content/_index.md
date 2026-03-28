---
title: "Gaze"
description: "Pure-Go file watching for Go."
toc: false
sidebar:
  hide: true
---

`go.ofkm.dev/gaze` is Gaze, a public Go package for filesystem events on Linux, macOS, and Windows with no CGO dependency.

The easiest way to use it is through a single entrypoint:

```go
w, err := gaze.WatchDirectory("my-directory")
```

When you need more control, build a config value and pass it to the matching `...WithConfig` constructor:

```go
cfg := gaze.Config{
	ExcludeGlobs: []string{"*.tmp", ".DS_Store"},
	OnEvent: func(evt gaze.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
}

w, err := gaze.WatchDirectoryWithConfig("my-directory", cfg)
```

Gaze is opinionated so it stays easy to use in production.

- pure Go implementation with per-platform native backends
- directory watching is recursive by default
- filtering applies both when watches are enrolled and when events are delivered
- events are normalized, including rename pairing and overflow detection
- the package owns its goroutines, so your code only handles callbacks
- overflow is reported explicitly instead of being dropped silently

## Start here

- [Quickstart](/docs/quickstart) for the minimal setup.
- [API](/docs/api) for constructors, methods, and full config surface.
- [Filtering](/docs/filtering) for glob/prefix/predicate excludes and opcode filtering.
- [Platforms](/docs/platforms) for backend behavior and tradeoffs.
- [Examples](/docs/examples) for practical patterns.

## What to expect

- callbacks always run in package-owned goroutines
- `Config.OnError` receives callback failures and runtime watcher errors
- if you do not provide handlers, Gaze logs events and errors with `slog`
- `Config.Logger` replaces the default logger used by that fallback path
