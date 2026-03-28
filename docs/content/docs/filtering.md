---
title: "Filtering"
description: "Exclude paths before they are watched and before events are emitted."
weight: 3
---

Filtering happens in two places:

1. when Gaze is deciding what to watch
2. when Gaze is deciding what events to deliver

That matters for large trees. Good filters reduce both watch count and callback noise.

## Glob excludes

```go
cfg := gaze.Config{
	ExcludeGlobs: []string{"*.tmp", "*.swp", ".DS_Store"},
	OnEvent: func(gaze.Event) {},
}

w, err := gaze.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```

- globs match base names and, where it makes sense, path segments
- smaller, specific patterns are usually easier to reason about than broad wildcards

## Prefix excludes

```go
cfg := gaze.Config{
	ExcludePrefixes: []string{
		"/absolute/path/to/my-directory/.git",
		"/absolute/path/to/my-directory/node_modules",
	},
	OnEvent: func(gaze.Event) {},
}

w, err := gaze.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```

Use prefix excludes for directories you never want to watch at all.

## Predicate excludes

```go
cfg := gaze.Config{
	Exclude: func(info gaze.PathInfo) bool {
		return info.IsDir && info.Base == "vendor"
	},
	OnEvent: func(gaze.Event) {},
}

w, err := gaze.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```

Use `Exclude` when globs and prefixes are not quite enough.

## Op filtering

```go
cfg := gaze.Config{
	Ops: gaze.OpCreate | gaze.OpWrite | gaze.OpRename,
	OnEvent: func(evt gaze.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
}

w, err := gaze.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```

- `cfg.Ops = 0` means all operations
- `OpOverflow` is always delivered
- op filtering happens after backend normalization and rename pairing

## A practical note

Filtering makes watchers cheaper and quieter, but it also means you are intentionally ignoring part of the tree. If exact external state matters, combine filtering with reconciliation when you receive `OpOverflow`.
