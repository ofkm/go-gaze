---
title: "Filtering"
description: "Exclude paths before they are watched and before events are emitted."
weight: 3
---

Filtering is applied in two places:

1. during watch enrollment, so excluded directories are not walked or enrolled recursively
2. during event dispatch, so excluded paths do not reach your handlers

That keeps both kernel watch count and callback load down on large trees.

## Glob excludes

```go
cfg := gofilewatch.Config{
	ExcludeGlobs: []string{"*.tmp", "*.swp", ".DS_Store"},
	OnEvent: func(gofilewatch.Event) {},
}

w, err := gofilewatch.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```

- glob patterns match base names and relevant path segments
- narrow patterns are easier to reason about than broad wildcards

## Prefix excludes

```go
cfg := gofilewatch.Config{
	ExcludePrefixes: []string{
		"/absolute/path/to/my-directory/.git",
		"/absolute/path/to/my-directory/node_modules",
	},
	OnEvent: func(gofilewatch.Event) {},
}

w, err := gofilewatch.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```

Prefix excludes are ideal for large trees you never want to enroll at all.

## Predicate excludes

```go
cfg := gofilewatch.Config{
	Exclude: func(info gofilewatch.PathInfo) bool {
		return info.IsDir && info.Base == "vendor"
	},
	OnEvent: func(gofilewatch.Event) {},
}

w, err := gofilewatch.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```

Use `Exclude` when the decision depends on path state that globs and fixed prefixes cannot express cleanly.

## Op filtering

```go
cfg := gofilewatch.Config{
	Ops: gofilewatch.OpCreate | gofilewatch.OpWrite | gofilewatch.OpRename,
	OnEvent: func(evt gofilewatch.Event) {
		fmt.Println(evt.Op, evt.Path)
	},
}

w, err := gofilewatch.WatchDirectoryWithConfig("my-directory", cfg)
if err != nil {
	panic(err)
}
```

- `cfg.Ops = 0` means all operations
- `OpOverflow` is always retained because it signals lost fidelity
- op filtering happens after backend normalization and rename pairing

## Excludes and correctness

Excludes reduce noise and watch pressure, but they also mean you are intentionally not observing some transitions. If exact external state matters, combine excludes with reconciliation scans when you receive `OpOverflow`.
