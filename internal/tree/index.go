package tree

import (
	"sync"

	"go.ofkm.dev/gaze/pkg/utils"
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

	var updated map[string]Root
	for key, root := range i.roots {
		rootChanged := false
		if root.Path == oldPath {
			root.Path = newPath
			rootChanged = true
		} else if root.IsDir && utils.HasPathPrefix(root.Path, oldPath) {
			root.Path = utils.JoinMovedPath(root.Path, oldPath, newPath)
			rootChanged = true
		}

		if root.WatchPath == oldPath {
			root.WatchPath = newPath
			rootChanged = true
		} else if utils.HasPathPrefix(root.WatchPath, oldPath) {
			root.WatchPath = utils.JoinMovedPath(root.WatchPath, oldPath, newPath)
			rootChanged = true
		}

		if rootChanged {
			if updated == nil {
				updated = make(map[string]Root, len(i.roots))
				for existingKey, existingRoot := range i.roots {
					updated[existingKey] = existingRoot
				}
			}
			delete(updated, key)
			updated[root.Path] = root
			continue
		}
	}
	if updated == nil {
		return
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

	parent := utils.ParentDir(path)
	if _, ok := i.flatDirs[parent]; ok {
		return true
	}

	current := path
	for {
		if _, ok := i.recursiveDirs[current]; ok {
			return true
		}
		next := utils.ParentDir(current)
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
