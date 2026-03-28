---
title: "API"
description: "Public package surface for go-filewatch."
weight: 2
---

## Constructors

```go
type Configure func(*Config)

func WatchDirectory(path string, configure ...Configure) (*Watcher, error)
func WatchFile(path string, configure ...Configure) (*Watcher, error)
func New(configure ...Configure) (*Watcher, error)
func DefaultConfig() Config
```

- `WatchDirectory` validates and registers a single path immediately.
- `WatchFile` sets recursion to false and registers only that file.
- `New` creates an empty watcher for adding many roots later.
- `DefaultConfig` returns the package defaults for callers that want to inspect or reuse them.

## Watcher lifecycle

```go
func (w *Watcher) Add(path string) error
func (w *Watcher) Remove(path string) error
func (w *Watcher) Close() error
```

- `Add` can be used with both files and directories.
  - a directory root is added according to the active `Config.Recursive` setting
  - a file root is normalized to watch only that file's parent with file-only filtering
- `Remove` removes the matching root from the watcher.
- `Close` is idempotent and releases kernel watchers, queue goroutines, and OS resources.

`ErrWatcherClosed` is available when a watcher has been closed:

```go
var ErrWatcherClosed = errors.New("filewatch: watcher closed")
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
- `OldPath` is set for paired renames and is empty for non-rename events.
- `IsDir` reports whether the subject path is a directory.
- `OpOverflow` indicates lost fidelity and that the caller should rescan the watched tree.

### `Event` dispatch notes

- callbacks are always called serially by package-owned workers in the order delivered by the backend queue
- `OpRename` emits paired `OldPath -> Path` when available; when pairing fails you may see remove/create instead
- callback panics are captured and surfaced to the error path

## Config model

```go
type Config struct {
	Recursive       bool
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

- `Recursive` defaults to `true` for directory roots.
- `ExcludeGlobs` and `ExcludePrefixes` can be combined with `Exclude`.
- `Exclude` receives `PathInfo`:

```go
type PathInfo struct {
	Path  string
	Base  string
	IsDir bool
}
```

- `OnEvent` is called for each event that passes filtering.
- `OnError` receives internal errors and event-handler panics.
- `Logger` replaces the logger used for default fallback logging.
- `Ops` configures event op filtering.
  - `Ops = 0` is treated as "all ops".
  - `OpOverflow` is always retained and cannot be disabled.
- `QueueCapacity` controls the watcher queue depth and therefore burst buffering.
- `FollowSymlinks` controls whether symlink roots are accepted and enrolled.

## Callback contract

The package keeps callback dispatch internal; user code does not start watcher goroutines directly.

- event handler: `cfg.OnEvent = func(evt Event) { ... }`
- error handler: `cfg.OnError = func(err error) { ... }`

If you omit handlers, filewatch emits both events and errors through `slog` instead.

## Example with full configuration

```go
w, err := gofilewatch.WatchDirectory(
	"my-directory",
	func(cfg *gofilewatch.Config) {
		cfg.Recursive = true
		cfg.ExcludeGlobs = []string{"*.tmp", "*.swp", ".DS_Store"}
		cfg.ExcludePrefixes = []string{filepath.Join(".", "node_modules")}
		cfg.Exclude = func(info gofilewatch.PathInfo) bool {
			return info.IsDir && info.Base == ".git"
		}
		cfg.Ops = gofilewatch.OpCreate | gofilewatch.OpWrite | gofilewatch.OpRename
		cfg.QueueCapacity = 4096
		cfg.FollowSymlinks = false
		cfg.OnEvent = func(evt gofilewatch.Event) {
			// handle event
		}
		cfg.OnError = func(err error) {
			// handle watcher error
		}
		cfg.Logger = slog.Default()
	},
)
```
