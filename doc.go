// Package filewatch provides a pure-Go filesystem watcher with native backends.
//
// It is designed around a simple entrypoint:
//
//	w, err := filewatch.WatchDirectory("my-directory")
//
// Event dispatch stays inside the package. Callers provide handlers by mutating
// Config inside the constructor callback, or rely on the default slog-based
// logging path for both events and internal watcher errors.
//
// Linux and Windows are the strongest scalability targets. macOS remains pure-Go
// and functional, but its backend enrolls more kernel watches and should be
// expected to scale less efficiently on very large recursive trees.
package filewatch
