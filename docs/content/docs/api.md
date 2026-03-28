---
title: 'API'
description: 'Public package surface for Gaze.'
weight: 2
---

## Constructors

```go
func WatchDirectory(path string) (*Watcher, error)
func WatchDirectoryWithConfig(path string, cfg Config) (*Watcher, error)
func WatchFile(path string) (*Watcher, error)
func WatchFileWithConfig(path string, cfg Config) (*Watcher, error)
func New() (*Watcher, error)
func NewWithConfig(cfg Config) (*Watcher, error)
```

- `WatchDirectory` is the simplest way to watch a directory recursively.
- `WatchDirectoryWithConfig` does the same thing with explicit configuration.
- `WatchFile` watches one file by watching its parent directory.
- `New` and `NewWithConfig` create an empty watcher so you can `Add` roots later.

## Watcher lifecycle

```go
func (w *Watcher) Add(path string) error
func (w *Watcher) Remove(path string) error
func (w *Watcher) Close() error
```

- `Add` accepts either a file or a directory.
- `Remove` removes the exact root you added.
- `Close` is idempotent.

`ErrWatcherClosed` is returned when you use a watcher after shutdown:

```go
var ErrWatcherClosed = errors.New("gaze: watcher closed")
```

## Event model

```go
type Event struct {
	Path    string
	OldPath string
	Op      Op
	IsDir   bool
}

type Op uint32

const (
	OpCreate Op = 1 << iota
	OpWrite
	OpRemove
	OpRename
	OpChmod
	OpOverflow
)
```

- `Path` is the main path for the event.
- `OldPath` is set when a rename can be paired.
- `IsDir` tells you whether the subject is a directory.
- `OpOverflow` means the backend or queue lost fidelity and you should rescan if exact state matters.

### Delivery notes

- callbacks run on package-owned goroutines
- `OpRename` can degrade to remove plus create when a backend cannot pair both sides
- callback panics are recovered and forwarded to `Config.OnError` or the logger

## Config

`Config` is meant to be used as a plain struct literal. `Config{}` is valid.

```go
type Config struct {
	Recursion       RecursionMode
	ExcludeGlobs    []string
	ExcludePrefixes []string
	Exclude         func(PathInfo) bool
	OnEvent         func(Event)
	OnError         func(error)
	Logger          *slog.Logger
	Ops             Op
	QueueCapacity   int
	FollowSymlinks  bool
}
```

```go
type RecursionMode uint8

const (
	RecursionDefault RecursionMode = iota
	RecursionDisabled
	RecursionEnabled
)
```

```go
type PathInfo struct {
	Path  string
	Base  string
	IsDir bool
}
```

### Field behavior

- `RecursionDefault` keeps directory watches recursive and file watches non-recursive
- `RecursionDisabled` watches only the top-level directory
- `RecursionEnabled` forces recursive directory enrollment
- `ExcludeGlobs`, `ExcludePrefixes`, and `Exclude` are applied when watches are enrolled and when events are emitted
- `OnEvent` receives normalized events that survive filtering
- `OnError` receives runtime watcher errors and recovered handler panics
- `Logger` is used when handlers are omitted
- `Ops = 0` means all operations
- `OpOverflow` is always delivered
- `QueueCapacity <= 0` falls back to the default queue depth
- `FollowSymlinks` allows symlink roots

## Defaults

If you pass `Config{}`, Gaze uses these defaults:

- directory watches are recursive
- file watches stay file-only
- all event ops are enabled
- queue capacity is `1024`
- fallback logging uses `slog.Default()`
- symlink roots are rejected unless `FollowSymlinks` is true

## Example

```go
logger := slog.Default()

cfg := gaze.Config{
	Recursion:       gaze.RecursionEnabled,
	ExcludeGlobs:    []string{"*.tmp", "*.swp", ".DS_Store"},
	ExcludePrefixes: []string{"/srv/app/.git", "/srv/app/node_modules"},
	Exclude: func(info gaze.PathInfo) bool {
		return info.IsDir && info.Base == "vendor"
	},
	Ops:            gaze.OpCreate | gaze.OpWrite | gaze.OpRename,
	QueueCapacity:  4096,
	FollowSymlinks: false,
	OnEvent: func(evt gaze.Event) {
		fmt.Println(evt)
	},
	OnError: func(err error) {
		logger.Error("watch error", "err", err)
	},
	Logger: logger,
}

w, err := gaze.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```
