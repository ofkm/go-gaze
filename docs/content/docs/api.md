---
title: "API"
description: "Public package surface for Gaze."
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

- `WatchDirectory` is the shortest path for recursive directory watching with default logging.
- `WatchDirectoryWithConfig` uses the same behavior, but with explicit configuration.
- `WatchFile` watches a single file through its parent directory.
- `New` and `NewWithConfig` create an empty watcher for multi-root use via `Add`.

## Watcher lifecycle

```go
func (w *Watcher) Add(path string) error
func (w *Watcher) Remove(path string) error
func (w *Watcher) Close() error
```

- `Add` accepts both files and directories.
- `Remove` removes the exact root you added earlier.
- `Close` is idempotent and releases backend watches, queue state, and package-owned goroutines.

`ErrWatcherClosed` is returned when operations are attempted after shutdown:

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

- `Path` is the effective path for the event.
- `OldPath` is set when the backend can pair a rename reliably.
- `IsDir` reports whether the subject is a directory.
- `OpOverflow` means backend or queue fidelity was lost and the caller should rescan.

### Delivery notes

- Callbacks run serially on package-owned goroutines.
- `OpRename` may degrade to remove-plus-create when a backend cannot pair both sides.
- Callback panics are recovered and routed to `Config.OnError` or the configured logger.

## Config model

`Config` is designed to work as a plain struct literal. The package applies defaults automatically, so `Config{}` is valid.

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

- `RecursionDefault` keeps directory watches recursive and file watches non-recursive.
- `RecursionDisabled` watches only the root directory level.
- `RecursionEnabled` forces recursive directory enrollment.
- `ExcludeGlobs`, `ExcludePrefixes`, and `Exclude` are all applied both during enrollment and event dispatch.
- `OnEvent` receives normalized events that survive filtering.
- `OnError` receives runtime watcher errors and recovered handler panics.
- `Logger` replaces the fallback logger used when handlers are omitted.
- `Ops = 0` means all operations.
- `OpOverflow` is always retained and cannot be disabled.
- `QueueCapacity <= 0` falls back to the package default queue depth.
- `FollowSymlinks` opts into accepting symlink roots.

## Sane defaults

These defaults are applied even when you pass `Config{}`:

- directory watches recurse by default
- file watches stay file-only
- all event ops are enabled
- queue capacity defaults to `1024`
- fallback logging uses `slog.Default()`
- symlink roots are rejected unless `FollowSymlinks` is true

## Example config

```go
logger := slog.Default()

cfg := gofilewatch.Config{
	Recursion:       gofilewatch.RecursionEnabled,
	ExcludeGlobs:    []string{"*.tmp", "*.swp", ".DS_Store"},
	ExcludePrefixes: []string{"/srv/app/.git", "/srv/app/node_modules"},
	Exclude: func(info gofilewatch.PathInfo) bool {
		return info.IsDir && info.Base == "vendor"
	},
	Ops:           gofilewatch.OpCreate | gofilewatch.OpWrite | gofilewatch.OpRename,
	QueueCapacity: 4096,
	FollowSymlinks: false,
	OnEvent: func(evt gofilewatch.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
	OnError: func(err error) {
		logger.Error("watch error", "err", err)
	},
	Logger: logger,
}

w, err := gofilewatch.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```
