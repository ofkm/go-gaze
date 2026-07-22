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
// To add filters, callbacks, or a logger, define a [Config] once and reuse it
// for as many watchers as you need:
//
//	cfg := gaze.Config{
//		ExcludeGlobs: []string{"*.tmp"},
//		OnEvent: func(evt gaze.Event) {
//			fmt.Println(evt)
//		},
//	}
//
//	w1, err := cfg.WatchDirectory("my-directory")
//	w2, err := cfg.WatchFile("app.log")
//
// # Choosing a constructor
//
//   - [WatchDirectory] / [Config.WatchDirectory] watch one directory,
//     recursively by default.
//   - [WatchFile] / [Config.WatchFile] watch one file (through its parent
//     directory).
//   - [New] / [Config.NewWatcher] start empty; enroll roots later with
//     [Watcher.Add], or watch several roots from a single Watcher.
//
// Every package-level constructor takes an optional [Config]; omit it for
// defaults. The types formerly in the types subpackage (Config, Event, Op,
// PathInfo, RecursionMode) now live in this package; go.ofkm.dev/gaze/types
// remains as a compatibility alias shim.
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
