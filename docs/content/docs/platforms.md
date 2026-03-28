---
title: "Platforms"
description: "Native backend behavior across Linux, macOS, and Windows."
weight: 4
---

Gaze uses the native watcher API on each supported platform:

- Linux: `inotify`
- Windows: `ReadDirectoryChangesW`
- macOS: `kqueue`

The public API stays the same across all three, but the underlying tradeoffs are different.

## Linux (`inotify`)

- recursive trees are represented with one watch per directory
- queue overflow is surfaced as `OpOverflow`
- rename pairing uses rename cookies when the kernel provides them

In practice, Linux tends to scale well for large recursive trees.

## Windows (`ReadDirectoryChangesW`)

- subtree watching is delegated to the OS when recursion is enabled
- rename pairs are normalized into `OldPath` and `Path`
- heavy rename and move workloads can still produce short bursts of activity

In practice, Windows handles sustained change load well and usually needs less bookkeeping from the watcher than Linux or macOS.

## macOS (`kqueue`)

- the implementation stays pure Go and does not rely on cgo
- recursive watching is built from the native pieces available on macOS
- large trees usually cost more kernel watches than they do on Windows, and often more overhead than Linux

In practice, macOS works well for normal project sizes, but it is the least efficient backend for very large recursive trees.

## Cross-platform behavior

All backends feed the same public `Event` type.

- `OpRename` includes both paths when a rename can be paired
- if pairing fails, you may see remove plus create instead
- callback behavior and error handling are consistent across platforms
