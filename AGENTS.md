# Project Guidelines

## Build and test

- Target Go version is `1.26.1` (see `go.mod`).
- Use `go test ./...` as the primary validation command. This repo is a library, so test coverage matters more than producing a binary.
- Run benchmarks with `go test ./... -run=^$ -bench=. -benchmem` when changing hot paths, watcher delivery, filtering, or tree/index logic.
- If benchmark output changes and you need to refresh the published performance page, regenerate it with `go run ./scripts/benchmarkdocs docs/content/docs/performance.md benchmark-artifacts/*/benchmark.txt`.

## Architecture

- The public API lives in the root `gaze` package. Start with `watcher.go`, `config.go`, `event.go`, and `doc.go` before changing behavior.
- Keep public constructors thin. Shared setup should continue to flow through `NewWithConfig` and `newWatcher` instead of being duplicated.
- OS-specific filesystem watching lives behind `internal/backend/backend.go`, with build-tagged implementations in `internal/backend/backend_*.go`.
- Path filtering is centralized in `internal/filter`, watched-root matching and rename bookkeeping live in `internal/tree`, and event buffering/backpressure handling live in `internal/queue`.
- Backend differences are real: platform-specific behavior is expected, especially around rename pairing and large recursive trees.

## Conventions

- Prefer the standard library and existing internal packages over adding dependencies.
- Keep `Config{}` valid. Preserve defaulting behavior in `resolveConfig` when adding options.
- Keep user-facing error messages prefixed with `gaze:`.
- Use `log/slog` for logging-related behavior.
- Remember the API semantics:
  - directory watches are recursive by default
  - `WatchFile*` watches the parent directory and filters to the requested file
  - symlink roots require `FollowSymlinks: true`
  - overflow events must still be delivered when fidelity is lost
- Avoid blocking callback paths unnecessarily; slow `OnEvent` handlers and small queues can cause backpressure.

## Tests

- Follow the existing split between same-package tests (`package gaze`) for internals/seams and external-package tests (`package gaze_test`) for public behavior and examples.
- Prefer `t.Parallel()` where safe, `t.TempDir()` for filesystem fixtures, and helper functions with timeouts instead of arbitrary sleeps.
- When testing internal behavior, prefer lightweight stubs and existing seams such as `newBackend` and `newMatcher`.
- If you change platform backends, verify behavior on the affected OS where possible; do not assume Linux, macOS, and Windows behave identically.

## Docs and generated content

- Edit documentation source in `docs/content/docs/`, not the generated site output in `docs/public/`.
- Prefer linking to existing docs rather than duplicating them in code comments or new instruction files:
  - `docs/content/docs/quickstart.md`
  - `docs/content/docs/api.md`
  - `docs/content/docs/filtering.md`
  - `docs/content/docs/platforms.md`
  - `docs/content/docs/examples.md`
  - `docs/content/docs/performance.md`
- `docs/content/docs/performance.md` contains generated benchmark tables; avoid hand-editing the generated results section.
- Verify example paths against the actual repository tree before documenting them. Treat docs as helpful context, not infallible truth.

## Read these first

- `watcher.go` — watcher lifecycle, event normalization, dispatch, and queue flow
- `config.go` — config defaults and option semantics
- `internal/backend/backend.go` — backend interface boundary
- `internal/filter/matcher.go` — exclude logic
- `internal/tree/index.go` — watched-root matching and rename updates
- `watcher_unit_test.go` and `filewatch_test.go` — expected testing patterns and behavior coverage
