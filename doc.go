// Package gaze is a pure-Go filesystem watcher for Linux, macOS, and Windows.
//
// For the common case, start with a directory watch:
//
//	w, err := gaze.WatchDirectory("my-directory")
//	if err != nil {
//		// handle error
//	}
//	defer w.Close()
//
// To add filters, callbacks, or a logger, pass a [types.Config]:
//
//	cfg := types.Config{
//		ExcludeGlobs: []string{"*.tmp"},
//		OnEvent: func(evt types.Event) {
//			fmt.Println(evt)
//		},
//	}
//
//	w, err := gaze.WatchDirectory("my-directory", cfg)
//
// # Choosing a constructor
//
//   - [WatchDirectory] watches one directory, recursively by default.
//   - [WatchFile] watches one file (through its parent directory).
//   - [New] starts empty; enroll roots later with [Watcher.Add], or watch
//     several roots from a single Watcher.
//
// Every constructor takes an optional [types.Config]; omit it for defaults.
//
// # Choosing how to exclude paths
//
// The three exclude mechanisms apply both when enrolling watches and when
// delivering events, and may be combined:
//
//   - Config.ExcludeGlobs matches base names or slash paths, e.g. "*.tmp".
//   - Config.ExcludePrefixes skips whole subtrees by path prefix.
//   - Config.Exclude is a predicate for logic globs and prefixes cannot express.
//
// Gaze owns the watcher goroutines internally. Handle events and errors with
// the Config callbacks, or let the package log them through slog; callbacks run
// on package-owned goroutines. See https://gaze.ofkm.dev for the full guide.
//
// Linux and Windows generally handle large recursive trees best. macOS is still
// pure Go and works well for normal project sizes, but it uses more kernel
// watches and tends to be less efficient on very large trees.
package gaze
