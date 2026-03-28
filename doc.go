// Package gaze powers Gaze, a pure-Go filesystem watcher with native backends.
//
// It is designed around a simple entrypoint:
//
//	w, err := gaze.WatchDirectory("my-directory")
//
// For explicit configuration, build a Config value directly and pass it to a
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
// Event dispatch stays inside the package. Callers provide handlers through a
// Config value, or rely on the default slog-based logging path for both events
// and internal watcher errors.
//
// Linux and Windows are the strongest scalability targets. macOS remains pure-Go
// and functional, but its backend enrolls more kernel watches and should be
// expected to scale less efficiently on very large recursive trees.
package gaze
