// Package gaze provides Gaze, a pure-Go filesystem watcher with native
// backends for Linux, macOS, and Windows.
//
// The simplest way to use it is through a single entrypoint:
//
//	w, err := gaze.WatchDirectory("my-directory")
//
// If you need more control, build a Config value and pass it to a
// ...WithConfig constructor:
//
//	cfg := gaze.Config{
//		ExcludeGlobs: []string{"*.tmp"},
//		OnEvent: func(evt gaze.Event) {
//			fmt.Println(evt.Op, evt.Path)
//		},
//	}
//
//	w, err := gaze.WatchDirectoryWithConfig("my-directory", cfg)
//
// Callbacks run inside the package. You can provide handlers in Config, or let
// Gaze fall back to slog-based logging for both events and internal watcher
// errors.
//
// Linux and Windows scale best. macOS stays pure-Go and works well, but its
// backend uses more kernel watches and is less efficient on very large
// recursive trees.
package gaze
