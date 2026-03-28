package tree

import (
	"path/filepath"
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

	updated := make(map[string]Root, len(i.roots))
	for _, root := range i.roots {
		if root.Path == oldPath {
			root.Path = newPath
		} else if root.IsDir && hasPathPrefix(root.Path, oldPath) {
			root.Path = joinMovedPath(root.Path, oldPath, newPath)
		}

		if root.WatchPath == oldPath {
			root.WatchPath = newPath
		} else if hasPathPrefix(root.WatchPath, oldPath) {
			root.WatchPath = joinMovedPath(root.WatchPath, oldPath, newPath)
		}

		updated[root.Path] = root
	}
	i.roots = updated
	i.rebuild()
}

func (i *Index) Matches(path string) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if _, ok := i.fileRoots[path]; ok {
		return true
	}

	parent := filepath.Dir(path)
	if _, ok := i.flatDirs[parent]; ok {
		return true
	}

	current := path
	for {
		if _, ok := i.recursiveDirs[current]; ok {
			return true
		}
		next := filepath.Dir(current)
		if next == current {
			return false
		}
		current = next
	}
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
