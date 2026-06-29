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
| `BenchmarkWatchDirectoryCreateRemove` | `443031 ns/op` | `4047 B/op` | `29 allocs/op` |
| `BenchmarkOpString` | `0.4689 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `680.2 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `108.8 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `171.5 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `834.1 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `888.8 ns/op` | `944 B/op` | `7 allocs/op` |

### `linux/amd64`

CPU: `AMD EPYC 7763 64-Core Processor`

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkWatchDirectoryCreateRemove` | `83277 ns/op` | `598 B/op` | `11 allocs/op` |
| `BenchmarkOpString` | `0.3120 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `493.5 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `77.13 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `122.6 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `716.7 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `500.2 ns/op` | `944 B/op` | `7 allocs/op` |

### `windows/amd64`

CPU: `AMD EPYC 7763 64-Core Processor`

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkWatchDirectoryCreateRemove` | `685079 ns/op` | `1723 B/op` | `12 allocs/op` |
| `BenchmarkOpString` | `0.3196 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExclude` | `1863 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatches` | `83.80 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMatchesDeepPath` | `96.67 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkFilterShouldExcludeDeepPath` | `5070 ns/op` | `0 B/op` | `0 allocs/op` |
| `BenchmarkTreeMovePrefix` | `741.4 ns/op` | `944 B/op` | `7 allocs/op` |

How to read this page:

- `BenchmarkWatchDirectoryCreateRemove` is closest to real watcher work. It includes filesystem activity and backend event handling, so it is not a pure microbenchmark.
- `BenchmarkOpString`, `BenchmarkFilterShouldExclude`, and `BenchmarkTreeMatches` are tighter hot-path checks.
- Absolute numbers will move with hardware, runner class, Go version, and filesystem behavior. The useful signal is whether a change moves runtime, allocations, or both.
- If you want fresh committed numbers, run the `Benchmarks` workflow in GitHub Actions.
