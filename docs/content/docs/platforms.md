---
title: "Platforms"
description: "Native backend behavior across Linux, macOS, and Windows."
weight: 4
---

The package ships a pure-Go platform layer for:

- Linux: `inotify`
- Windows: `ReadDirectoryChangesW`
- macOS: `kqueue`

## Linux (`inotify`)

- recursive trees are represented with per-directory watch descriptors
- queue overflow is surfaced as `OpOverflow`
- rename pairing uses rename cookies when available

Operational notes:

- strong scaling characteristics for very large recursive trees
- rename and overflow fidelity are preserved when the backend signal stream is healthy

## Windows (`ReadDirectoryChangesW`)

- single root handle with subtree watching where requested
- `FILE_ACTION_RENAMED_OLD_NAME`/`NEW_NAME` pairing is normalized to `OldPath` + `Path`
- rename and move heavy write loads can produce rename/update bursts depending on file system behavior

Operational notes:

- stable under frequent change load
- recursion delegated to OS handles (when enabled in the watcher)

## macOS (`kqueue`)

- pure-Go implementation to keep the package cgo-free
- recursive behavior uses native mechanisms available on that platform
- scale is adequate for typical project sizes and is slower to scale on very large trees than Linux/Windows

Operational notes:

- useful baseline on macOS when you need strict pure-Go delivery
- large tree workloads may increase kernel watch count and CPU usage faster

## Cross-platform behavior contract

All backends emit the same public normalized `Event` model.

- `OpRename` includes paired paths when pairing succeeds (`OldPath` and `Path`)
- when pairing fails, you can observe `OpRemove` + `OpCreate` instead
- callback behavior and handler errors are platform-independent
