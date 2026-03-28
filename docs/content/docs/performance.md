---
title: Performance
weight: 6
---

Gaze includes a small benchmark suite. It is there to catch regressions and make changes easier to compare over time, not to claim a single universal performance number.

Run it with:

```sh
go test ./... -bench=. -benchmem
```

Current local numbers from `darwin/arm64`:

| Benchmark                             | Result                                      |
| ------------------------------------- | ------------------------------------------- |
| `BenchmarkWatchDirectoryCreateRemove` | `170497 ns/op`, `5001 B/op`, `36 allocs/op` |
| `BenchmarkOpString`                   | `77.34 ns/op`, `96 B/op`, `2 allocs/op`     |
| `BenchmarkFilterShouldExclude`        | `454.1 ns/op`, `0 B/op`, `0 allocs/op`      |
| `BenchmarkTreeMatches`                | `138.2 ns/op`, `0 B/op`, `0 allocs/op`      |

How to read that table:

- `BenchmarkWatchDirectoryCreateRemove` includes real watcher setup and teardown, so it is closer to integration work than a tight microbenchmark
- the filter and tree benchmarks are effectively allocation-free in steady state
- the absolute numbers will change with hardware, OS, filesystem, and Go version

The useful signal is trend, not the exact number. If a change noticeably moves allocations or runtime, that is worth looking at.

If you want to publish fresh numbers, rerun the command on the platform you care about and update this page with the new output.
