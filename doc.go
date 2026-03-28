// Package gaze provides Gaze, a pure-Go filesystem watcher for Linux, macOS,
// and Windows.
//
// For the common case, start with a directory watch:
//
//	w, err := gaze.WatchDirectory("my-directory")
//
// If you need filters, callbacks, or logger control, use a Config value with a
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
// Gaze owns the watcher goroutines internally. You can handle events and
// errors with callbacks, or let the package log them through slog.
//
// Linux and Windows generally handle large recursive trees best. macOS is
// still pure Go and works well for normal project sizes, but it uses more
// kernel watches and tends to be less efficient on very large trees.
package gaze
