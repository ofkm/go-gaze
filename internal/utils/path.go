// Package utils provides small helper utilities shared by Gaze internals.
package utils

import (
	"path/filepath"
	"strings"
)

// HasPathPrefix reports whether path equals prefix or is contained below it.
func HasPathPrefix(path, prefix string) bool {
	return path == prefix || len(path) > len(prefix) && path[:len(prefix)] == prefix && path[len(prefix)] == filepath.Separator
}

// JoinMovedPath rewrites path from oldPrefix to newPrefix.
func JoinMovedPath(path, oldPrefix, newPrefix string) string {
	if path == oldPrefix {
		return newPrefix
	}
	return newPrefix + path[len(oldPrefix):]
}

// ParentDir returns the parent directory of path using string slicing.
func ParentDir(path string) string {
	idx := strings.LastIndexByte(path, filepath.Separator)
	if idx < 0 {
		return "."
	}
	if idx == 0 {
		return string(filepath.Separator)
	}
	return path[:idx]
}
