//go:build windows

package filter

import "path/filepath"

// toSlashFast on Windows delegates to filepath.ToSlash.
func toSlashFast(path string) string {
	return filepath.ToSlash(path)
}
