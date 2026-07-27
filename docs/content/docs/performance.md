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
| `BenchmarkWatchDirectoryCreateRemove` | `181564 ns/op` | `3373 B/op` | `25 allocs/op` |
| `BenchmarkOpString` | `0.3784 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `316.4 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `79.71 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `124.9 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `547.1 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `552.6 ns/op` | `944 B/op` | `7 allocs/op` |

### `linux/amd64`

CPU: `AMD EPYC 9V74 80-Core Processor`

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkWatchDirectoryCreateRemove` | `66421 ns/op` | `643 B/op` | `11 allocs/op` |
| `BenchmarkOpString` | `0.2732 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `312.9 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `59.26 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `94.25 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `465.0 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `379.4 ns/op` | `944 B/op` | `7 allocs/op` |

### `windows/amd64`

CPU: `AMD EPYC 9V74 80-Core Processor`

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkWatchDirectoryCreateRemove` | `586209 ns/op` | `1465 B/op` | `10 allocs/op` |
| `BenchmarkOpString` | `0.3577 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `2014 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `85.00 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `100.8 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `5351 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `711.4 ns/op` | `944 B/op` | `7 allocs/op` |

How to read this page:

- `BenchmarkWatchDirectoryCreateRemove` is closest to real watcher work. It includes filesystem activity and backend event handling, so it is not a pure microbenchmark.
- `BenchmarkOpString`, `BenchmarkFilterShouldExclude`, and `BenchmarkTreeMatches` are tighter hot-path checks.
- Absolute numbers will move with hardware, runner class, Go version, and filesystem behavior. The useful signal is whether a change moves runtime, allocations, or both.
- If you want fresh committed numbers, run the `Benchmarks` workflow in GitHub Actions.
