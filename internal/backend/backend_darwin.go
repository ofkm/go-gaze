//go:build darwin

package backend

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

const darwinWatchFlags = unix.NOTE_DELETE | unix.NOTE_WRITE | unix.NOTE_ATTRIB | unix.NOTE_RENAME | unix.NOTE_EXTEND | unix.NOTE_REVOKE

var (
	darwinKqueue  = unix.Kqueue
	darwinKevent  = unix.Kevent
	darwinOpen    = unix.Open
	darwinClose   = unix.Close
	darwinReadDir = os.ReadDir
	darwinLstat   = unix.Lstat
)

type darwinWatcher struct {
	cfg Config
	kq  int

	mu        sync.Mutex
	closed    bool
	roots     map[string]Target
	rootNodes map[string]map[string]struct{}
	watched   map[string]*darwinNode
	fdToPath  map[uintptr]string
	snapshots map[string]map[string]entryMeta

	events chan Event
	errors chan error
	done   chan struct{}
}

type darwinNode struct {
	fd    int
	path  string
	isDir bool
	roots map[string]struct{}
}

type entryMeta struct {
	isDir bool
	inode uint64
}

func New(cfg Config) (Watcher, error) {
	kq, err := darwinKqueue()
	if err != nil {
		return nil, fmt.Errorf("gaze: init kqueue: %w", err)
	}

	w := &darwinWatcher{
		cfg:       cfg,
		kq:        kq,
		roots:     make(map[string]Target),
		rootNodes: make(map[string]map[string]struct{}),
		watched:   make(map[string]*darwinNode),
		fdToPath:  make(map[uintptr]string),
		snapshots: make(map[string]map[string]entryMeta),
		events:    make(chan Event, cfg.BufferSize),
		errors:    make(chan error, 64),
		done:      make(chan struct{}),
	}
	go w.run()
	return w, nil
}

func (w *darwinWatcher) Add(target Target) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return os.ErrClosed
	}
	if _, ok := w.roots[target.Path]; ok {
		w.mu.Unlock()
		return nil
	}
	w.roots[target.Path] = target
	w.rootNodes[target.Path] = make(map[string]struct{})
	w.mu.Unlock()

	if target.IsDir {
		if err := w.enrollDirectoryTarget(target); err != nil {
			_ = w.Remove(target.Path)
			return err
		}
		return nil
	}

	if err := w.addPath(target.Path, target.WatchPath, true); err != nil {
		_ = w.Remove(target.Path)
		return err
	}
	if err := w.addPath(target.Path, target.Path, false); err != nil {
		_ = w.Remove(target.Path)
		return err
	}
	return nil
}

func (w *darwinWatcher) Remove(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, ok := w.roots[path]; !ok {
		return os.ErrNotExist
	}

	for watchedPath := range w.rootNodes[path] {
		node := w.watched[watchedPath]
		if node == nil {
			continue
		}
		delete(node.roots, path)
		if len(node.roots) == 0 {
			_ = w.unregisterLocked(node)
		}
	}

	delete(w.rootNodes, path)
	delete(w.roots, path)
	return nil
}

func (w *darwinWatcher) Events() <-chan Event {
	return w.events
}

func (w *darwinWatcher) Errors() <-chan error {
	return w.errors
}

func (w *darwinWatcher) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true

	for _, node := range w.watched {
		_ = darwinClose(node.fd)
	}
	w.watched = map[string]*darwinNode{}
	err := darwinClose(w.kq)
	w.mu.Unlock()

	<-w.done
	return err
}

func (w *darwinWatcher) run() {
	defer func() {
		close(w.events)
		close(w.errors)
		close(w.done)
	}()

	events := make([]unix.Kevent_t, 128)
	for {
		n, err := darwinKevent(w.kq, nil, events, nil)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			if w.isClosed() {
				return
			}
			w.emitError(fmt.Errorf("gaze: kevent: %w", err))
			continue
		}

		for i := range n {
			w.handle(events[i])
		}
	}
}

func (w *darwinWatcher) handle(ev unix.Kevent_t) {
	w.mu.Lock()
	path := w.fdToPath[uintptr(ev.Ident)]
	node := w.watched[path]
	w.mu.Unlock()
	if node == nil {
		return
	}

	if ev.Fflags&unix.NOTE_ATTRIB != 0 {
		w.emitEvent(Event{Path: path, Op: OpChmod, IsDir: node.isDir})
	}

	if node.isDir {
		if ev.Fflags&(unix.NOTE_DELETE|unix.NOTE_RENAME|unix.NOTE_REVOKE) != 0 {
			w.removePrefix(path)
			w.emitEvent(Event{Path: path, Op: OpRemove, IsDir: true})
			parent := filepath.Dir(path)
			if parent != path {
				w.rescanDir(parent)
			}
			return
		}
		if ev.Fflags&(unix.NOTE_WRITE|unix.NOTE_EXTEND) != 0 {
			w.rescanDir(path)
		}
		return
	}

	if ev.Fflags&(unix.NOTE_DELETE|unix.NOTE_RENAME|unix.NOTE_REVOKE) != 0 {
		w.removePath(path)
		w.emitEvent(Event{Path: path, Op: OpRemove, IsDir: false})
		parent := filepath.Dir(path)
		if parent != path {
			w.rescanDir(parent)
		}
		return
	}
	if ev.Fflags&(unix.NOTE_WRITE|unix.NOTE_EXTEND) != 0 {
		w.emitEvent(Event{Path: path, Op: OpWrite, IsDir: false})
	}
}

func (w *darwinWatcher) enrollDirectoryTarget(target Target) error {
	if target.Recursive {
		return walkPath(target.Path, true, w.cfg.ShouldExclude, func(path string, d os.DirEntry) error {
			return w.addPath(target.Path, path, d.IsDir())
		})
	}

	if err := w.addPath(target.Path, target.Path, true); err != nil {
		return err
	}
	entries, err := darwinReadDir(target.Path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		child := filepath.Join(target.Path, entry.Name())
		if w.cfg.ShouldExclude != nil && w.cfg.ShouldExclude(child, false) {
			continue
		}
		if err := w.addPath(target.Path, child, false); err != nil {
			return err
		}
	}
	return nil
}

func (w *darwinWatcher) addPath(root, path string, isDir bool) error {
	if w.cfg.ShouldExclude != nil && w.cfg.ShouldExclude(path, isDir) && path != w.roots[root].Path {
		return nil
	}

	w.mu.Lock()
	if node := w.watched[path]; node != nil {
		node.roots[root] = struct{}{}
		w.rootNodes[root][path] = struct{}{}
		w.mu.Unlock()
		return nil
	}
	w.mu.Unlock()

	flag := unix.O_RDONLY
	if isDir {
		flag = unix.O_EVTONLY
	}
	fd, err := darwinOpen(path, flag, 0)
	if err != nil {
		return fmt.Errorf("gaze: open %q: %w", path, err)
	}

	change := []unix.Kevent_t{{}}
	unix.SetKevent(&change[0], fd, unix.EVFILT_VNODE, unix.EV_ADD|unix.EV_ENABLE|unix.EV_CLEAR)
	change[0].Fflags = darwinWatchFlags
	if _, err := darwinKevent(w.kq, change, nil, nil); err != nil {
		_ = darwinClose(fd)
		return fmt.Errorf("gaze: register %q: %w", path, err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		_ = darwinClose(fd)
		return os.ErrClosed
	}
	if node := w.watched[path]; node != nil {
		_ = darwinClose(fd)
		node.roots[root] = struct{}{}
		w.rootNodes[root][path] = struct{}{}
		return nil
	}

	w.watched[path] = &darwinNode{
		fd:    fd,
		path:  path,
		isDir: isDir,
		roots: map[string]struct{}{root: {}},
	}
	w.fdToPath[uintptr(fd)] = path
	w.rootNodes[root][path] = struct{}{}
	if isDir {
		w.snapshots[path] = w.readDirSnapshot(path)
	}
	return nil
}

func (w *darwinWatcher) rescanDir(path string) {
	w.mu.Lock()
	node := w.watched[path]
	if node == nil || !node.isDir {
		w.mu.Unlock()
		return
	}
	oldSnapshot := copySnapshot(w.snapshots[path])
	roots := make([]string, 0, len(node.roots))
	for root := range node.roots {
		roots = append(roots, root)
	}
	w.mu.Unlock()

	newSnapshot := w.readDirSnapshot(path)

	added := make(map[string]entryMeta)
	removed := make(map[string]entryMeta)
	for name, meta := range oldSnapshot {
		if _, ok := newSnapshot[name]; !ok {
			removed[name] = meta
		}
	}
	for name, meta := range newSnapshot {
		if _, ok := oldSnapshot[name]; !ok {
			added[name] = meta
		}
	}

	for oldName, oldMeta := range removed {
		for newName, newMeta := range added {
			if oldMeta.inode == 0 || oldMeta.inode != newMeta.inode || oldMeta.isDir != newMeta.isDir {
				continue
			}
			oldPath := filepath.Join(path, oldName)
			newPath := filepath.Join(path, newName)
			if oldMeta.isDir {
				w.renamePrefix(oldPath, newPath)
			} else {
				w.renamePath(oldPath, newPath)
			}
			w.emitEvent(Event{Path: newPath, OldPath: oldPath, Op: OpRename, IsDir: oldMeta.isDir})
			delete(removed, oldName)
			delete(added, newName)
			break
		}
	}

	for name, meta := range removed {
		child := filepath.Join(path, name)
		if meta.isDir {
			w.removePrefix(child)
		} else {
			w.removePath(child)
		}
		w.emitEvent(Event{Path: child, Op: OpRemove, IsDir: meta.isDir})
	}

	for name, meta := range added {
		child := filepath.Join(path, name)
		if w.cfg.ShouldExclude != nil && w.cfg.ShouldExclude(child, meta.isDir) {
			continue
		}

		for _, root := range roots {
			target := w.rootTarget(root)
			switch {
			case target.Path == child && !meta.isDir:
				_ = w.addPath(root, child, false)
			case target.IsDir && target.Recursive && hasPathPrefix(child, target.Path):
				if meta.isDir {
					_ = walkPath(child, true, w.cfg.ShouldExclude, func(candidate string, d os.DirEntry) error {
						return w.addPath(root, candidate, d.IsDir())
					})
				} else {
					_ = w.addPath(root, child, false)
				}
			case target.IsDir && !target.Recursive && filepath.Dir(child) == target.Path && !meta.isDir:
				_ = w.addPath(root, child, false)
			}
		}
		w.emitEvent(Event{Path: child, Op: OpCreate, IsDir: meta.isDir})
	}

	w.mu.Lock()
	w.snapshots[path] = newSnapshot
	w.mu.Unlock()
}

func (w *darwinWatcher) removePrefix(prefix string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for path, node := range w.watched {
		if hasPathPrefix(path, prefix) {
			_ = w.unregisterLocked(node)
		}
	}
}

func (w *darwinWatcher) removePath(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if node := w.watched[path]; node != nil {
		_ = w.unregisterLocked(node)
	}
}

func (w *darwinWatcher) renamePrefix(oldPrefix, newPrefix string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	updated := make(map[string]*darwinNode, len(w.watched))
	updatedSnapshots := make(map[string]map[string]entryMeta, len(w.snapshots))
	for path, node := range w.watched {
		if hasPathPrefix(path, oldPrefix) {
			newPath := filepath.Clean(newPrefix + path[len(oldPrefix):])
			node.path = newPath
			updated[newPath] = node
			w.fdToPath[uintptr(node.fd)] = newPath
			for root := range node.roots {
				delete(w.rootNodes[root], path)
				w.rootNodes[root][newPath] = struct{}{}
			}
			if snapshot, ok := w.snapshots[path]; ok {
				updatedSnapshots[newPath] = snapshot
				delete(w.snapshots, path)
			}
			continue
		}
		updated[path] = node
	}
	for path, snapshot := range w.snapshots {
		if _, ok := updatedSnapshots[path]; !ok {
			updatedSnapshots[path] = snapshot
		}
	}
	w.watched = updated
	w.snapshots = updatedSnapshots
}

func (w *darwinWatcher) renamePath(oldPath, newPath string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	node := w.watched[oldPath]
	if node == nil {
		return
	}
	delete(w.watched, oldPath)
	w.watched[newPath] = node
	w.fdToPath[uintptr(node.fd)] = newPath
	node.path = newPath
	for root := range node.roots {
		delete(w.rootNodes[root], oldPath)
		w.rootNodes[root][newPath] = struct{}{}
	}
}

func (w *darwinWatcher) unregisterLocked(node *darwinNode) error {
	delete(w.fdToPath, uintptr(node.fd))
	delete(w.snapshots, node.path)
	delete(w.watched, node.path)
	for root := range node.roots {
		delete(w.rootNodes[root], node.path)
	}
	return darwinClose(node.fd)
}

func (w *darwinWatcher) rootTarget(root string) Target {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.roots[root]
}

func (w *darwinWatcher) readDirSnapshot(path string) map[string]entryMeta {
	entries, err := darwinReadDir(path)
	if err != nil {
		return map[string]entryMeta{}
	}
	snapshot := make(map[string]entryMeta, len(entries))
	for _, entry := range entries {
		child := filepath.Join(path, entry.Name())
		if w.cfg.ShouldExclude != nil && w.cfg.ShouldExclude(child, entry.IsDir()) {
			continue
		}
		snapshot[entry.Name()] = entryMeta{
			isDir: entry.IsDir(),
			inode: inodeForPath(child),
		}
	}
	return snapshot
}

func (w *darwinWatcher) emitEvent(evt Event) {
	w.events <- evt
}

func (w *darwinWatcher) emitError(err error) {
	select {
	case w.errors <- err:
	default:
	}
}

func (w *darwinWatcher) isClosed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}

func inodeForPath(path string) uint64 {
	var stat unix.Stat_t
	if err := darwinLstat(path, &stat); err != nil {
		return 0
	}
	return stat.Ino
}

func copySnapshot(src map[string]entryMeta) map[string]entryMeta {
	dst := make(map[string]entryMeta, len(src))
	maps.Copy(dst, src)
	return dst
}

var _ = unsafe.Pointer(nil)
