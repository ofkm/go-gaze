# Gaze

### Simple Filesystem Watcher Written in Pure Go

> [!IMPORTANT]
> Gaze is still pre-release software. Expect Minor versions to have breaking changes until v1 is hit.

> [!NOTE]
> Recent breaking change: the `…WithConfig` constructors were removed. `WatchDirectory`, `WatchFile`, and `New` now take an optional `types.Config` directly — e.g. `gaze.WatchDirectory("dir", cfg)`. The bare forms (`gaze.WatchDirectory("dir")`) are unchanged.
