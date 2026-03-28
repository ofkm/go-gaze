package tree

import (
	"path/filepath"
	"strings"
	"sync"
)

type Root struct {
	Path      string
	WatchPath string
	IsDir     bool
	Recursive bool
}

type Index struct {
	mu            sync.RWMutex
	roots         map[string]Root
	fileRoots     map[string]struct{}
	flatDirs      map[string]struct{}
	recursiveDirs map[string]struct{}
}

func New() *Index {
	return &Index{
		roots:         make(map[string]Root),
		fileRoots:     make(map[string]struct{}),
		flatDirs:      make(map[string]struct{}),
		recursiveDirs: make(map[string]struct{}),
	}
}

func (i *Index) Add(root Root) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.roots[root.Path] = root
	i.rebuild()
	return nil
}

func (i *Index) Remove(path string) (Root, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	root, ok := i.roots[path]
	if !ok {
		return Root{}, false
	}
	delete(i.roots, path)
	i.rebuild()
	return root, true
}

func (i *Index) MovePrefix(oldPath, newPath string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	var toDelete []string
	var toAdd []Root
	for key, root := range i.roots {
		changed := false
		if root.Path == oldPath {
			root.Path = newPath
			changed = true
		} else if root.IsDir && hasPathPrefix(root.Path, oldPath) {
			root.Path = joinMovedPath(root.Path, oldPath, newPath)
			changed = true
		}

		if root.WatchPath == oldPath {
			root.WatchPath = newPath
		} else if hasPathPrefix(root.WatchPath, oldPath) {
			root.WatchPath = joinMovedPath(root.WatchPath, oldPath, newPath)
		}

		if changed {
			toDelete = append(toDelete, key)
			toAdd = append(toAdd, root)
		} else {
			i.roots[key] = root
		}
	}
	for _, key := range toDelete {
		delete(i.roots, key)
	}
	for _, root := range toAdd {
		i.roots[root.Path] = root
	}
	i.rebuild()
}

func (i *Index) Matches(path string) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if _, ok := i.fileRoots[path]; ok {
		return true
	}

	parent := parentDir(path)
	if _, ok := i.flatDirs[parent]; ok {
		return true
	}

	current := path
	for {
		if _, ok := i.recursiveDirs[current]; ok {
			return true
		}
		next := parentDir(current)
		if next == current {
			return false
		}
		current = next
	}
}

// parentDir returns the parent directory of path using zero-allocation
// string slicing instead of filepath.Dir.
func parentDir(path string) string {
	idx := strings.LastIndexByte(path, filepath.Separator)
	if idx < 0 {
		return "."
	}
	if idx == 0 {
		return string(filepath.Separator)
	}
	return path[:idx]
}

func (i *Index) rebuild() {
	clear(i.fileRoots)
	clear(i.flatDirs)
	clear(i.recursiveDirs)

	for _, root := range i.roots {
		switch {
		case root.IsDir && root.Recursive:
			i.recursiveDirs[root.Path] = struct{}{}
		case root.IsDir:
			i.flatDirs[root.Path] = struct{}{}
		default:
			i.fileRoots[root.Path] = struct{}{}
		}
	}
}

func hasPathPrefix(path, prefix string) bool {
	return path == prefix || len(path) > len(prefix) && path[:len(prefix)] == prefix && path[len(prefix)] == filepath.Separator
}

func joinMovedPath(path, oldPath, newPath string) string {
	if path == oldPath {
		return newPath
	}
	rel := path[len(oldPath):]
	return filepath.Clean(newPath + rel)
}
