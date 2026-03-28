package filewatch

import "strings"

type Op uint32

const (
	OpCreate Op = 1 << iota
	OpWrite
	OpRemove
	OpRename
	OpChmod
	OpOverflow
)

const allOps = OpCreate | OpWrite | OpRemove | OpRename | OpChmod | OpOverflow

type Event struct {
	Path    string
	OldPath string
	Op      Op
	IsDir   bool
}

type PathInfo struct {
	Path  string
	Base  string
	IsDir bool
}

func (o Op) Has(other Op) bool {
	return o&other != 0
}

func (o Op) String() string {
	if o == 0 {
		return "none"
	}

	var parts []string
	if o.Has(OpCreate) {
		parts = append(parts, "create")
	}
	if o.Has(OpWrite) {
		parts = append(parts, "write")
	}
	if o.Has(OpRemove) {
		parts = append(parts, "remove")
	}
	if o.Has(OpRename) {
		parts = append(parts, "rename")
	}
	if o.Has(OpChmod) {
		parts = append(parts, "chmod")
	}
	if o.Has(OpOverflow) {
		parts = append(parts, "overflow")
	}
	return strings.Join(parts, "|")
}
