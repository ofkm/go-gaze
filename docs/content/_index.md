---
title: 'Gaze'
description: 'Gaze is a pure-Go filesystem watcher for Linux, macOS, and Windows.'
toc: false
sidebar:
  hide: true
---

Gaze watches files and directories without cgo. It uses the native backend for each platform, keeps the API small, and normalizes the events you get back.

If you just want to start watching a directory, this is enough:

```go
w, err := gaze.WatchDirectory("my-directory")
```

If you need filters or callbacks, pass a config:

```go
cfg := gaze.Config{
	ExcludeGlobs: []string{"*.tmp", ".DS_Store"},
	OnEvent: func(evt gaze.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
}

w, err := gaze.WatchDirectoryWithConfig("my-directory", cfg)
```

A few things to know up front:

- it is pure Go, with native backends for Linux, macOS, and Windows
- directory watches are recursive by default
- filtering happens both when watches are enrolled and when events are delivered
- rename and overflow events are normalized into one event model
- the package manages its own goroutines
- if you do not provide handlers, events and errors are logged with `slog`

## Start here

- [Quickstart](/docs/quickstart) for the smallest working setup
- [API](/docs/api) for constructors, config fields, and lifecycle methods
- [Filtering](/docs/filtering) for glob, prefix, predicate, and op filters
- [Platforms](/docs/platforms) for backend behavior and tradeoffs
- [Examples](/docs/examples) for common usage patterns
- [Performance](/docs/performance) for benchmark notes and current numbers
