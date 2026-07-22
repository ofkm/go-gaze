// Package types is a compatibility shim. The canonical definitions now live
// in the root gaze package; prefer gaze.Config, gaze.Event, and friends in new
// code. Every identifier here is an alias, so existing imports keep compiling
// unchanged.
package types

import "go.ofkm.dev/gaze"

// Aliases for the canonical types in the root gaze package.
type (
	Config        = gaze.Config
	Event         = gaze.Event
	Op            = gaze.Op
	PathInfo      = gaze.PathInfo
	RecursionMode = gaze.RecursionMode
)

// Operation constants re-exported from the root gaze package.
const (
	OpCreate   = gaze.OpCreate
	OpWrite    = gaze.OpWrite
	OpRemove   = gaze.OpRemove
	OpRename   = gaze.OpRename
	OpChmod    = gaze.OpChmod
	OpOverflow = gaze.OpOverflow

	// AllOps contains every operation Gaze can deliver.
	AllOps = gaze.AllOps
)

// Recursion constants re-exported from the root gaze package.
const (
	RecursionDefault  = gaze.RecursionDefault
	RecursionDisabled = gaze.RecursionDisabled
	RecursionEnabled  = gaze.RecursionEnabled
)
