---
title: 'Filtering'
description: 'Exclude paths before they are watched and before events are emitted.'
weight: 3
---

Filtering happens in two places:

1. when Gaze is deciding what to watch
2. when Gaze is deciding what events to deliver

That matters for large trees. Good filters reduce both watch count and callback noise.

## Glob excludes

```go
cfg := types.Config{
	ExcludeGlobs: []string{"*.tmp", "*.swp", ".DS_Store"},
	OnEvent: func(types.Event) {},
}

w, err := gaze.WatchDirectory("my-directory", cfg)
if err != nil {
	panic(err)
}
```

- globs match base names and, where it makes sense, path segments
- smaller, specific patterns are usually easier to reason about than broad wildcards

## Prefix excludes

```go
cfg := types.Config{
	ExcludePrefixes: []string{
		"/absolute/path/to/my-directory/.git",
		"/absolute/path/to/my-directory/node_modules",
	},
	OnEvent: func(types.Event) {},
}

w, err := gaze.WatchDirectory("my-directory", cfg)
if err != nil {
	panic(err)
}
```

Use prefix excludes for directories you never want to watch at all.

## Predicate excludes

```go
cfg := types.Config{
	Exclude: func(info types.PathInfo) bool {
		return info.IsDir && info.Base == "vendor"
	},
	OnEvent: func(types.Event) {},
}

w, err := gaze.WatchDirectory("my-directory", cfg)
if err != nil {
	panic(err)
}
```

Use `Exclude` when globs and prefixes are not quite enough.

## Op filtering

```go
cfg := types.Config{
	Ops: types.OpCreate | types.OpWrite | types.OpRename,
	OnEvent: func(evt types.Event) {
		fmt.Println(evt)
	},
}

w, err := gaze.WatchDirectory("my-directory", cfg)
if err != nil {
	panic(err)
}
```

- `cfg.Ops = 0` means all operations
- `OpOverflow` is always delivered
- op filtering happens after backend normalization and rename pairing

## A practical note

Filtering makes watchers cheaper and quieter, but it also means you are intentionally ignoring part of the tree. If exact external state matters, combine filtering with reconciliation when you receive `OpOverflow`.
