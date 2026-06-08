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
| `BenchmarkWatchDirectoryCreateRemove` | `108076 ns/op` | `3950 B/op` | `29 allocs/op` |
| `BenchmarkOpString` | `0.3195 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `367.7 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `62.75 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `99.63 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `537.9 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `348.6 ns/op` | `944 B/op` | `7 allocs/op` |

### `linux/amd64`

CPU: `AMD EPYC 9V74 80-Core Processor`

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkWatchDirectoryCreateRemove` | `72251 ns/op` | `644 B/op` | `11 allocs/op` |
| `BenchmarkOpString` | `0.3523 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `500.7 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `76.57 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `121.1 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `731.0 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `484.8 ns/op` | `944 B/op` | `7 allocs/op` |

### `windows/amd64`

CPU: `AMD EPYC 9V74 80-Core Processor`

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkWatchDirectoryCreateRemove` | `633643 ns/op` | `1503 B/op` | `11 allocs/op` |
| `BenchmarkOpString` | `0.3636 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `2007 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `85.08 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `98.44 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `5281 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `731.0 ns/op` | `944 B/op` | `7 allocs/op` |

How to read this page:

- `BenchmarkWatchDirectoryCreateRemove` is closest to real watcher work. It includes filesystem activity and backend event handling, so it is not a pure microbenchmark.
- `BenchmarkOpString`, `BenchmarkFilterShouldExclude`, and `BenchmarkTreeMatches` are tighter hot-path checks.
- Absolute numbers will move with hardware, runner class, Go version, and filesystem behavior. The useful signal is whether a change moves runtime, allocations, or both.
- If you want fresh committed numbers, run the `Benchmarks` workflow in GitHub Actions.
