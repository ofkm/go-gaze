// Package types contains Gaze's public configuration and event types.
package types

import (
	"fmt"
	"log/slog"
	"strings"
)

// RecursionMode controls whether directory watches include descendants.
type RecursionMode uint8

const (
	// RecursionDefault lets the constructor choose: WatchDirectory watches
	// directories recursively, WatchFile watches non-recursively.
	RecursionDefault RecursionMode = iota
	// RecursionDisabled watches only the target directory or file.
	RecursionDisabled
	// RecursionEnabled watches directory descendants recursively.
	RecursionEnabled
)

// Op identifies one or more filesystem operations.
type Op uint32

const (
	// OpCreate reports a created file or directory.
	OpCreate Op = 1 << iota
	// OpWrite reports changed file contents.
	OpWrite
	// OpRemove reports a removed file or directory.
	OpRemove
	// OpRename reports a paired rename when the backend can provide one.
	OpRename
	// OpChmod reports metadata or permission changes.
	OpChmod
	// OpOverflow reports lost backend fidelity after an event queue overflow.
	OpOverflow
)

// AllOps contains every operation Gaze can deliver.
const AllOps = OpCreate | OpWrite | OpRemove | OpRename | OpChmod | OpOverflow

// Config controls watcher behavior. The zero value, Config{}, is valid and
// selects every default described on the fields below.
type Config struct {
	// Recursion controls whether directory watches include descendants. The
	// zero value, RecursionDefault, lets the constructor decide: WatchDirectory
	// watches recursively, WatchFile watches non-recursively. RecursionDisabled
	// and RecursionEnabled force the choice.
	Recursion RecursionMode

	// ExcludeGlobs skips paths whose base name or slash-separated path matches
	// any pattern (see path/filepath.Match). Applied both when enrolling
	// watches and when delivering events. Nil disables glob exclusion.
	ExcludeGlobs []string

	// ExcludePrefixes skips any path at or below one of these path prefixes.
	// Applied both when enrolling watches and when delivering events. Nil
	// disables prefix exclusion.
	ExcludePrefixes []string

	// Exclude, when non-nil, skips any path for which it returns true. Applied
	// both when enrolling watches and when delivering events; it may be called
	// from more than one goroutine, so it must be safe for concurrent use.
	Exclude func(PathInfo) bool

	// OnEvent receives each delivered event. It is called serially from a
	// single watcher-owned goroutine. If nil, events are logged via Logger.
	OnEvent func(Event)

	// OnError receives runtime errors and recovered handler panics. It may be
	// called concurrently from more than one goroutine, so it must be safe for
	// concurrent use. If nil, errors are logged via Logger.
	OnError func(error)

	// Logger is used when OnEvent or OnError is nil. A nil Logger means
	// slog.Default().
	Logger *slog.Logger

	// Ops is a bitmask of the operations to deliver. The zero value means all
	// operations. OpOverflow is always delivered regardless of this mask.
	Ops Op

	// QueueCapacity is the depth of the internal event buffer. Values <= 0 mean
	// the default (1024). A small queue with a slow OnEvent handler causes
	// backpressure and can lose fidelity (see OpOverflow).
	QueueCapacity int

	// FollowSymlinks, when false, rejects symlink roots. Set it true to resolve
	// and watch through symlinked roots.
	FollowSymlinks bool
}

// Event describes one filesystem change delivered by a watcher.
type Event struct {
	Path    string
	OldPath string
	Op      Op
	IsDir   bool
}

// PathInfo describes a path passed to a Config.Exclude callback.
type PathInfo struct {
	Path  string
	Base  string
	IsDir bool
}

// Has reports whether o contains other.
func (o Op) Has(other Op) bool {
	return o&other != 0
}

var opStrings = func() [AllOps + 1]string {
	var table [AllOps + 1]string
	table[0] = "none"

	names := [...]struct {
		op   Op
		name string
	}{
		{op: OpCreate, name: "create"},
		{op: OpWrite, name: "write"},
		{op: OpRemove, name: "remove"},
		{op: OpRename, name: "rename"},
		{op: OpChmod, name: "chmod"},
		{op: OpOverflow, name: "overflow"},
	}

	for mask := Op(1); mask <= AllOps; mask++ {
		var buf [48]byte
		n := 0
		for _, item := range names {
			if !mask.Has(item.op) {
				continue
			}
			if n > 0 {
				buf[n] = '|'
				n++
			}
			n += copy(buf[n:], item.name)
		}
		table[mask] = string(buf[:n])
	}

	return table
}()

// String returns a stable, pipe-separated operation label.
func (o Op) String() string {
	if o <= AllOps {
		return opStrings[o]
	}
	masked := o & AllOps
	if masked == 0 {
		return opStrings[0]
	}
	return opStrings[masked]
}

// String returns a compact event label for logs and examples.
func (e Event) String() string {
	label := "GAZE[" + strings.ToUpper(e.Op.String()) + "]"
	if e.Op.Has(OpRename) && e.OldPath != "" {
		return fmt.Sprintf("%s %s -> %s", label, e.OldPath, e.Path)
	}
	if e.Path == "" {
		return label
	}
	return label + " " + e.Path
}
