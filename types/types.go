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
	// RecursionDefault preserves the constructor's default recursion behavior.
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

// Config controls watcher behavior.
type Config struct {
	Recursion       RecursionMode
	ExcludeGlobs    []string
	ExcludePrefixes []string
	Exclude         func(PathInfo) bool
	OnEvent         func(Event)
	OnError         func(error)
	Logger          *slog.Logger
	Ops             Op
	QueueCapacity   int
	FollowSymlinks  bool
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
