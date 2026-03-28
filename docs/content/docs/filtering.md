---
title: "Filtering"
description: "Exclude paths before they are watched and before events are emitted."
weight: 3
---

Filtering is applied in two stages:

1. during watch enrollment (so excluded directories are not walked recursively), and
2. during event dispatch (so excluded files and paths never reach your handler).

This keeps scans and callback load low on large trees.

## Glob excludes

```go
w, err := gofilewatch.WatchDirectory(
	"my-directory",
	func(cfg *gofilewatch.Config) {
		cfg.ExcludeGlobs = []string{"*.tmp", "*.swp", ".DS_Store"}
		cfg.OnEvent = func(gofilewatch.Event) {}
	},
)
```

- patterns apply to base names and path segments.
- keep patterns narrow for predictable behavior.

## Prefix excludes

```go
w, err := gofilewatch.WatchDirectory(
	"my-directory",
	func(cfg *gofilewatch.Config) {
		cfg.ExcludePrefixes = []string{
			"/absolute/path/to/my-directory/.git",
			"/absolute/path/to/my-directory/node_modules",
		}
		cfg.OnEvent = func(gofilewatch.Event) {}
	},
)
```

Prefix excludes are good for large immutable trees you never want to watch or receive events for.

## Predicate excludes

```go
w, err := gofilewatch.WatchDirectory(
	"my-directory",
	func(cfg *gofilewatch.Config) {
		cfg.Exclude = func(info gofilewatch.PathInfo) bool {
			return info.IsDir && info.Base == "vendor"
		}
		cfg.OnEvent = func(gofilewatch.Event) {}
	},
)
```

Use this when naming or directory rules are stateful and cannot be expressed with globs.

## Op filtering

```go
w, err := gofilewatch.WatchDirectory(
	"my-directory",
	func(cfg *gofilewatch.Config) {
		cfg.Ops = gofilewatch.OpCreate | gofilewatch.OpWrite | gofilewatch.OpRename
		cfg.OnEvent = func(evt gofilewatch.Event) {
			fmt.Println(evt.Op, evt.Path)
		}
	},
)
```

- `cfg.Ops = 0` means all operations.
- `OpOverflow` is always kept and cannot be masked out because it indicates data loss risk.
- filtering happens after rename pairing and root/ancestor matching.

## Error path interaction

If exclusion logic drops too many events (for example on a broad include set), your handler should still assume there can be missed intermediate transitions for paths you do not observe. Use `OpOverflow` and reconciliation scans if that matters.
