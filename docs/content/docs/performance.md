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

CPU: `Apple M2 Pro (Virtual)`

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkWatchDirectoryCreateRemove` | `149005 ns/op` | `3736 B/op` | `28 allocs/op` |
| `BenchmarkOpString` | `0.3540 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `338.2 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `76.96 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `122.5 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `516.1 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `510.4 ns/op` | `944 B/op` | `7 allocs/op` |

### `linux/amd64`

CPU: `AMD EPYC 7763 64-Core Processor`

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkWatchDirectoryCreateRemove` | `102964 ns/op` | `640 B/op` | `11 allocs/op` |
| `BenchmarkOpString` | `0.3363 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `398.5 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `83.43 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `128.6 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `618.1 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `525.3 ns/op` | `944 B/op` | `7 allocs/op` |

### `windows/amd64`

CPU: `AMD EPYC 9V74 80-Core Processor`

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkWatchDirectoryCreateRemove` | `579133 ns/op` | `1762 B/op` | `12 allocs/op` |
| `BenchmarkOpString` | `0.3651 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `2156 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `96.75 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `105.7 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `5294 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `680.5 ns/op` | `944 B/op` | `7 allocs/op` |

How to read this page:

- `BenchmarkWatchDirectoryCreateRemove` is closest to real watcher work. It includes filesystem activity and backend event handling, so it is not a pure microbenchmark.
- `BenchmarkOpString`, `BenchmarkFilterShouldExclude`, and `BenchmarkTreeMatches` are tighter hot-path checks.
- Absolute numbers will move with hardware, runner class, Go version, and filesystem behavior. The useful signal is whether a change moves runtime, allocations, or both.
- If you want fresh committed numbers, run the `Benchmarks` workflow in GitHub Actions.
