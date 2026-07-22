# Gaze

### Simple Filesystem Watcher Written in Pure Go

> [!IMPORTANT]
> Gaze is still pre-release software. Expect Minor versions to have breaking changes until v1 is hit.

> [!NOTE]
> Recent breaking change: the `…WithConfig` constructors were removed. `WatchDirectory`, `WatchFile`, and `New` now take an optional `types.Config` directly — e.g. `gaze.WatchDirectory("dir", cfg)`. The bare forms (`gaze.WatchDirectory("dir")`) are unchanged.

The public types (`Config`, `Event`, `Op`, …) now live in the root `gaze` package; `go.ofkm.dev/gaze/types` remains as a compatibility alias shim, so existing imports keep working. A `Config` can be defined once and reused to create any number of watchers:

```go
cfg := gaze.Config{
	ExcludeGlobs: []string{"*.tmp"},
	OnEvent:      func(evt gaze.Event) { fmt.Println(evt) },
}

w1, err := cfg.WatchDirectory("/path/to/dir")
w2, err := cfg.WatchFile("/path/to/app.log")
w3, err := cfg.NewWatcher() // empty; enroll roots with w3.Add(...)
```
