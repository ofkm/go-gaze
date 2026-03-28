package gaze

import (
	"fmt"
	"strings"
)

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

var opStrings = func() [allOps + 1]string {
	var table [allOps + 1]string
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

	for mask := Op(1); mask <= allOps; mask++ {
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

func (o Op) String() string {
	if o <= allOps {
		return opStrings[o]
	}
	masked := o & allOps
	if masked == 0 {
		return opStrings[0]
	}
	return opStrings[masked]
}

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
