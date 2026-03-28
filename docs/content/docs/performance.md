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

| Benchmark                             |          ns/op |        B/op |      allocs/op |
| ------------------------------------- | -------------: | ----------: | -------------: |
| `BenchmarkWatchDirectoryCreateRemove` | `343137 ns/op` | `3319 B/op` | `23 allocs/op` |
| `BenchmarkOpString`                   | `0.3992 ns/op` |    `0 B/op` |  `0 allocs/op` |
| `BenchmarkFilterShouldExclude`        |  `660.1 ns/op` |    `0 B/op` |  `0 allocs/op` |
| `BenchmarkTreeMatches`                |  `216.5 ns/op` |    `0 B/op` |  `0 allocs/op` |

### `linux/amd64`

CPU: `AMD EPYC 7763 64-Core Processor`

| Benchmark                             |          ns/op |       B/op |      allocs/op |
| ------------------------------------- | -------------: | ---------: | -------------: |
| `BenchmarkWatchDirectoryCreateRemove` |  `78575 ns/op` | `642 B/op` | `11 allocs/op` |
| `BenchmarkOpString`                   | `0.3123 ns/op` |   `0 B/op` |  `0 allocs/op` |
| `BenchmarkFilterShouldExclude`        |  `495.3 ns/op` |   `0 B/op` |  `0 allocs/op` |
| `BenchmarkTreeMatches`                |  `156.1 ns/op` |   `0 B/op` |  `0 allocs/op` |

### `windows/amd64`

CPU: `AMD EPYC 7763 64-Core Processor`

| Benchmark                             |          ns/op |        B/op |      allocs/op |
| ------------------------------------- | -------------: | ----------: | -------------: |
| `BenchmarkWatchDirectoryCreateRemove` | `610521 ns/op` | `2301 B/op` | `18 allocs/op` |
| `BenchmarkOpString`                   | `0.3332 ns/op` |    `0 B/op` |  `0 allocs/op` |
| `BenchmarkFilterShouldExclude`        |   `1511 ns/op` |  `120 B/op` |  `5 allocs/op` |
| `BenchmarkTreeMatches`                |  `625.2 ns/op` |   `64 B/op` |  `4 allocs/op` |

How to read this page:

- `BenchmarkWatchDirectoryCreateRemove` is closest to real watcher work. It includes filesystem activity and backend event handling, so it is not a pure microbenchmark.
- `BenchmarkOpString`, `BenchmarkFilterShouldExclude`, and `BenchmarkTreeMatches` are tighter hot-path checks.
- Absolute numbers will move with hardware, runner class, Go version, and filesystem behavior. The useful signal is whether a change moves runtime, allocations, or both.
