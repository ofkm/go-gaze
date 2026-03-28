---
title: "go-filewatch"
description: "Pure-Go cross-platform file watching for Go."
toc: false
sidebar:
  hide: true
---

`go.ofkm.dev/filewatch` is a public Go package for filesystem events across Linux, macOS, and Windows with no CGO dependency.

It is callback-first by design:

```go
w, err := filewatch.WatchDirectory("my-directory")
```

The package is intentionally opinionated for easy production use.

- pure Go implementation with per-platform native backends
- directory watch by default, recursive automatically
- filtering in watch enrollment and event dispatch
- normalized event model with rename pairing and overflow detection
- internal goroutine ownership so application code handles only callbacks
- explicit overflow signaling instead of silent drops

## Start here

- [Quickstart](/docs/quickstart) for the minimal setup.
- [API](/docs/api) for constructors, methods, and full config surface.
- [Filtering](/docs/filtering) for glob/prefix/predicate excludes and opcode filtering.
- [Platforms](/docs/platforms) for backend behavior and tradeoffs.
- [Examples](/docs/examples) for practical patterns.

## How this package behaves

- callbacks always run in package-owned goroutines
- `Config.OnError` receives callback failures and runtime watcher errors
- if no handler is provided, events and errors are logged with `slog`
- `Config.Logger` replaces the default logger used by that fallback path
