//go:build !windows

package filter

// toSlashFast on Unix is a no-op since filepath.Separator is already '/'.
func toSlashFast(path string) string {
	return path
}
