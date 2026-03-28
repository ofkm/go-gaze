//go:build linux

package backend

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const linuxWatchMask = unix.IN_CREATE | unix.IN_MOVED_TO | unix.IN_MOVED_FROM | unix.IN_CLOSE_WRITE |
	unix.IN_MODIFY | unix.IN_DELETE | unix.IN_DELETE_SELF | unix.IN_ATTRIB | unix.IN_MOVE_SELF | unix.IN_ONLYDIR

type linuxWatcher struct {
	cfg Config
	fd  int

	mu       sync.Mutex
	closed   bool
	roots    map[string]Target
	rootDirs map[string]map[string]struct{}
	watched  map[string]*linuxNode
	wdToPath map[int]string
	pending  map[uint32]pendingRename

	events chan Event
	errors chan error
	done   chan struct{}
}

type linuxNode struct {
	wd    int
	path  string
	roots map[string]struct{}
}

type pendingRename struct {
	path  string
	isDir bool
	at    time.Time
}

func New(cfg Config) (Watcher, error) {
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC)
	if err != nil {
		return nil, fmt.Errorf("filewatch: init inotify: %w", err)
	}

	w := &linuxWatcher{
		cfg:      cfg,
		fd:       fd,
		roots:    make(map[string]Target),
		rootDirs: make(map[string]map[string]struct{}),
		watched:  make(map[string]*linuxNode),
		wdToPath: make(map[int]string),
		pending:  make(map[uint32]pendingRename),
		events:   make(chan Event, cfg.BufferSize),
		errors:   make(chan error, 64),
		done:     make(chan struct{}),
	}
	go w.run()
	return w, nil
}

func (w *linuxWatcher) Add(target Target) error {
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
	w.rootDirs[target.Path] = make(map[string]struct{})
	w.mu.Unlock()

	var addErr error
	if target.IsDir {
		addErr = walkPath(target.Path, target.Recursive, w.cfg.ShouldExclude, func(path string, d os.DirEntry) error {
			if !d.IsDir() {
				return nil
			}
			return w.addDir(target.Path, path)
		})
	} else {
		addErr = w.addDir(target.Path, target.WatchPath)
	}
	if addErr != nil {
		_ = w.Remove(target.Path)
		return addErr
	}

	return nil
}

func (w *linuxWatcher) Remove(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, ok := w.roots[path]; !ok {
		return os.ErrNotExist
	}

	for dir := range w.rootDirs[path] {
		node := w.watched[dir]
		if node == nil {
			continue
		}
		delete(node.roots, path)
		if len(node.roots) == 0 {
			_, _ = unix.InotifyRmWatch(w.fd, uint32(node.wd))
			delete(w.watched, dir)
			delete(w.wdToPath, node.wd)
		}
	}

	delete(w.rootDirs, path)
	delete(w.roots, path)
	return nil
}

func (w *linuxWatcher) Events() <-chan Event {
	return w.events
}

func (w *linuxWatcher) Errors() <-chan error {
	return w.errors
}

func (w *linuxWatcher) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	fd := w.fd
	w.mu.Unlock()

	err := unix.Close(fd)
	<-w.done
	return err
}

func (w *linuxWatcher) addDir(root, dir string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return os.ErrClosed
	}
	if node := w.watched[dir]; node != nil {
		node.roots[root] = struct{}{}
		w.rootDirs[root][dir] = struct{}{}
		return nil
	}

	wd, err := unix.InotifyAddWatch(w.fd, dir, linuxWatchMask)
	if err != nil {
		return fmt.Errorf("filewatch: watch %q: %w", dir, err)
	}
	w.watched[dir] = &linuxNode{
		wd:    wd,
		path:  dir,
		roots: map[string]struct{}{root: {}},
	}
	w.wdToPath[wd] = dir
	w.rootDirs[root][dir] = struct{}{}
	return nil
}

func (w *linuxWatcher) run() {
	defer func() {
		close(w.events)
		close(w.errors)
		close(w.done)
	}()

	buf := make([]byte, 1<<20)
	for {
		w.flushPending(time.Now().Add(-250 * time.Millisecond))

		poll := []unix.PollFd{{Fd: int32(w.fd), Events: unix.POLLIN}}
		n, err := unix.Poll(poll, 250)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			if w.isClosed() {
				return
			}
			w.emitError(fmt.Errorf("filewatch: poll inotify: %w", err))
			continue
		}
		if n == 0 {
			continue
		}

		read, err := unix.Read(w.fd, buf)
		if err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			if w.isClosed() {
				return
			}
			w.emitError(fmt.Errorf("filewatch: read inotify: %w", err))
			continue
		}

		offset := 0
		for offset+unix.SizeofInotifyEvent <= read {
			raw := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			offset += unix.SizeofInotifyEvent

			nameBytes := buf[offset : offset+int(raw.Len)]
			offset += int(raw.Len)
			name := trimNull(nameBytes)
			w.handle(raw, name)
		}
	}
}

func (w *linuxWatcher) handle(raw *unix.InotifyEvent, name string) {
	if raw.Mask&unix.IN_Q_OVERFLOW != 0 {
		w.emitEvent(Event{Op: OpOverflow})
		return
	}

	w.mu.Lock()
	base := w.wdToPath[int(raw.Wd)]
	w.mu.Unlock()
	if base == "" {
		return
	}

	fullPath := base
	if name != "" {
		fullPath = filepath.Join(base, name)
	}
	isDir := raw.Mask&unix.IN_ISDIR != 0

	if raw.Mask&unix.IN_MOVED_FROM != 0 {
		w.mu.Lock()
		w.pending[raw.Cookie] = pendingRename{
			path:  fullPath,
			isDir: isDir,
			at:    time.Now(),
		}
		w.mu.Unlock()
		return
	}

	if raw.Mask&unix.IN_MOVED_TO != 0 {
		oldPath, paired := w.takeRename(raw.Cookie)
		if isDir {
			w.addRecursiveDirIfNeeded(fullPath)
		}
		if paired {
			if isDir {
				w.renameWatchedPrefix(oldPath.path, fullPath)
			}
			w.emitEvent(Event{
				Path:    fullPath,
				OldPath: oldPath.path,
				Op:      OpRename,
				IsDir:   isDir,
			})
			return
		}
		w.emitEvent(Event{Path: fullPath, Op: OpCreate, IsDir: isDir})
		return
	}

	if raw.Mask&unix.IN_CREATE != 0 {
		if isDir {
			w.addRecursiveDirIfNeeded(fullPath)
		}
		w.emitEvent(Event{Path: fullPath, Op: OpCreate, IsDir: isDir})
	}
	if raw.Mask&unix.IN_CLOSE_WRITE != 0 || raw.Mask&unix.IN_MODIFY != 0 {
		w.emitEvent(Event{Path: fullPath, Op: OpWrite, IsDir: isDir})
	}
	if raw.Mask&unix.IN_ATTRIB != 0 {
		w.emitEvent(Event{Path: fullPath, Op: OpChmod, IsDir: isDir})
	}
	if raw.Mask&unix.IN_DELETE != 0 {
		if isDir {
			w.removeWatchedPrefix(fullPath)
		}
		w.emitEvent(Event{Path: fullPath, Op: OpRemove, IsDir: isDir})
	}
	if raw.Mask&unix.IN_DELETE_SELF != 0 || raw.Mask&unix.IN_MOVE_SELF != 0 {
		w.removeWatchedPrefix(fullPath)
		w.emitEvent(Event{Path: fullPath, Op: OpRemove, IsDir: true})
	}
	if raw.Mask&unix.IN_IGNORED != 0 {
		w.dropWatch(raw.Wd, fullPath)
	}
}

func (w *linuxWatcher) addRecursiveDirIfNeeded(path string) {
	w.mu.Lock()
	roots := make([]string, 0, len(w.roots))
	for rootPath, target := range w.roots {
		if !target.IsDir || !target.Recursive {
			continue
		}
		if hasPathPrefix(path, target.Path) {
			roots = append(roots, rootPath)
		}
	}
	w.mu.Unlock()

	for _, root := range roots {
		_ = walkPath(path, true, w.cfg.ShouldExclude, func(candidate string, d os.DirEntry) error {
			if !d.IsDir() {
				return nil
			}
			return w.addDir(root, candidate)
		})
	}
}

func (w *linuxWatcher) removeWatchedPrefix(prefix string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for path, node := range w.watched {
		if !hasPathPrefix(path, prefix) {
			continue
		}
		_, _ = unix.InotifyRmWatch(w.fd, uint32(node.wd))
		delete(w.wdToPath, node.wd)
		delete(w.watched, path)
		for root := range node.roots {
			delete(w.rootDirs[root], path)
		}
	}
}

func (w *linuxWatcher) renameWatchedPrefix(oldPrefix, newPrefix string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	updated := make(map[string]*linuxNode, len(w.watched))
	for path, node := range w.watched {
		if hasPathPrefix(path, oldPrefix) {
			newPath := filepath.Clean(newPrefix + path[len(oldPrefix):])
			node.path = newPath
			updated[newPath] = node
			w.wdToPath[node.wd] = newPath
			for root := range node.roots {
				delete(w.rootDirs[root], path)
				w.rootDirs[root][newPath] = struct{}{}
			}
			continue
		}
		updated[path] = node
	}
	w.watched = updated
}

func (w *linuxWatcher) dropWatch(wd int32, path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	node := w.watched[path]
	if node == nil {
		return
	}
	delete(w.wdToPath, int(wd))
	delete(w.watched, path)
	for root := range node.roots {
		delete(w.rootDirs[root], path)
	}
}

func (w *linuxWatcher) takeRename(cookie uint32) (pendingRename, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	rename, ok := w.pending[cookie]
	if ok {
		delete(w.pending, cookie)
	}
	return rename, ok
}

func (w *linuxWatcher) flushPending(cutoff time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for cookie, pending := range w.pending {
		if pending.at.After(cutoff) {
			continue
		}
		delete(w.pending, cookie)
		if pending.isDir {
			w.removeWatchedPrefixLocked(pending.path)
		}
		w.emitEventLocked(Event{Path: pending.path, Op: OpRemove, IsDir: pending.isDir})
	}
}

func (w *linuxWatcher) removeWatchedPrefixLocked(prefix string) {
	for path, node := range w.watched {
		if !hasPathPrefix(path, prefix) {
			continue
		}
		_, _ = unix.InotifyRmWatch(w.fd, uint32(node.wd))
		delete(w.wdToPath, node.wd)
		delete(w.watched, path)
		for root := range node.roots {
			delete(w.rootDirs[root], path)
		}
	}
}

func (w *linuxWatcher) emitEvent(evt Event) {
	w.events <- evt
}

func (w *linuxWatcher) emitEventLocked(evt Event) {
	w.events <- evt
}

func (w *linuxWatcher) emitError(err error) {
	select {
	case w.errors <- err:
	default:
	}
}

func (w *linuxWatcher) isClosed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}

func trimNull(buf []byte) string {
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}
