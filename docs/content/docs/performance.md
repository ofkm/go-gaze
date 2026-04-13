---
title: Performance
weight: 6
---

Gaze includes a small benchmark suite. The point is to watch trend lines across platforms, not to pretend one machine tells the whole story.

Run the local suite with:

```sh
go test ./... -run=^$ -bench=. -benchmem
```

The committed numbers below are generated from the `Benchmarks` GitHub Actions workflow. That workflow runs the same benchmark suite on Linux, macOS, and Windows, then rewrites this page with the latest published results.

## Latest published results

### `darwin/arm64`

CPU: `Apple M1 (Virtual)`

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkWatchDirectoryCreateRemove` | `196024 ns/op` | `3906 B/op` | `29 allocs/op` |
| `BenchmarkOpString` | `0.3358 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `393.7 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `70.32 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `113.2 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `569.1 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `536.6 ns/op` | `944 B/op` | `7 allocs/op` |

### `linux/amd64`

CPU: `AMD EPYC 7763 64-Core Processor`

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkWatchDirectoryCreateRemove` | `78794 ns/op` | `642 B/op` | `11 allocs/op` |
| `BenchmarkOpString` | `0.3125 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `494.4 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `78.24 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `133.8 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `716.0 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `615.0 ns/op` | `944 B/op` | `7 allocs/op` |

### `windows/amd64`

CPU: `AMD EPYC 9V74 80-Core Processor`

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkWatchDirectoryCreateRemove` | `571382 ns/op` | `1455 B/op` | `10 allocs/op` |
| `BenchmarkOpString` | `0.3560 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `2008 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `85.31 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `100.1 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `5264 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `858.7 ns/op` | `944 B/op` | `7 allocs/op` |

How to read this page:

- `BenchmarkWatchDirectoryCreateRemove` is closest to real watcher work. It includes filesystem activity and backend event handling, so it is not a pure microbenchmark.
- `BenchmarkOpString`, `BenchmarkFilterShouldExclude`, and `BenchmarkTreeMatches` are tighter hot-path checks.
- Absolute numbers will move with hardware, runner class, Go version, and filesystem behavior. The useful signal is whether a change moves runtime, allocations, or both.
- If you want fresh committed numbers, run the `Benchmarks` workflow in GitHub Actions.
