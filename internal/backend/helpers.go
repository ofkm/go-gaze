package backend

import (
	"io/fs"
	"os"
	"path/filepath"
)

func walkPath(root string, recursive bool, shouldExclude func(path string, isDir bool) bool, fn func(path string, d fs.DirEntry) error) error {
	if !recursive {
		info, err := os.Lstat(root)
		if err != nil {
			return err
		}
		dirEntry := fs.FileInfoToDirEntry(info)
		return fn(root, dirEntry)
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if shouldExclude != nil && shouldExclude(path, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		return fn(path, d)
	})
}

func hasPathPrefix(path, prefix string) bool {
	return path == prefix || len(path) > len(prefix) && path[:len(prefix)] == prefix && path[len(prefix)] == filepath.Separator
}
