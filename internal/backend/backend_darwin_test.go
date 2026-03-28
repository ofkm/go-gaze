//go:build darwin

package backend

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func openPathForTest(t *testing.T, path string, isDir bool) int {
	t.Helper()

	flag := unix.O_RDONLY
	if isDir {
		flag = unix.O_EVTONLY
	}
	fd, err := unix.Open(path, flag, 0)
	if err != nil {
		t.Fatalf("unix.Open(%q) error = %v", path, err)
	}
	return fd
}

func TestDarwinWatcherLifecycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	childFile := filepath.Join(root, "child.txt")
	if err := os.WriteFile(childFile, []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	wi, err := New(Config{BufferSize: 8})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	w := wi.(*darwinWatcher)

	if err := w.Add(Target{Path: root, WatchPath: root, IsDir: true, Recursive: false}); err != nil {
		t.Fatalf("Add(directory) error = %v", err)
	}
	if err := w.Add(Target{Path: root, WatchPath: root, IsDir: true, Recursive: false}); err != nil {
		t.Fatalf("Add(directory duplicate) error = %v", err)
	}
	if len(w.rootNodes[root]) == 0 {
		t.Fatal("directory target did not enroll any nodes")
	}
	if err := w.Remove(root); err != nil {
		t.Fatalf("Remove(directory) error = %v", err)
	}
	if err := w.Remove(root); !os.IsNotExist(err) {
		t.Fatalf("second Remove(directory) error = %v, want not exist", err)
	}

	fileWatcherAny, err := New(Config{BufferSize: 8})
	if err != nil {
		t.Fatalf("New(file watcher) error = %v", err)
	}
	fileWatcher := fileWatcherAny.(*darwinWatcher)
	if err := fileWatcher.Add(Target{
		Path:      childFile,
		WatchPath: root,
		IsDir:     false,
		Recursive: false,
	}); err != nil {
		t.Fatalf("Add(file) error = %v", err)
	}
	if _, ok := fileWatcher.watched[childFile]; !ok {
		t.Fatal("file target did not register file path")
	}
	if err := fileWatcher.Close(); err != nil {
		t.Fatalf("file watcher Close() error = %v", err)
	}
	if err := fileWatcher.Close(); err != nil {
		t.Fatalf("second file watcher Close() error = %v", err)
	}
	if err := fileWatcher.Add(Target{Path: childFile, WatchPath: root}); !strings.Contains(err.Error(), os.ErrClosed.Error()) {
		t.Fatalf("Add(closed watcher) error = %v, want closed error", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestDarwinWatcherHelpers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	oldDir := filepath.Join(root, "old")
	newDir := filepath.Join(root, "new")
	if err := os.Mkdir(oldDir, 0o755); err != nil {
		t.Fatalf("Mkdir(old) error = %v", err)
	}
	if err := os.Mkdir(newDir, 0o755); err != nil {
		t.Fatalf("Mkdir(new) error = %v", err)
	}

	oldFile := filepath.Join(oldDir, "file.txt")
	newFile := filepath.Join(newDir, "file.txt")
	if err := os.WriteFile(oldFile, []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile(oldFile) error = %v", err)
	}
	if err := os.WriteFile(newFile, []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile(newFile) error = %v", err)
	}

	dirFD := openPathForTest(t, oldDir, true)
	fileFD := openPathForTest(t, oldFile, false)

	w := &darwinWatcher{
		roots: map[string]Target{
			oldDir: {Path: oldDir, WatchPath: oldDir, IsDir: true, Recursive: true},
		},
		rootNodes: map[string]map[string]struct{}{
			oldDir: {
				oldDir:  {},
				oldFile: {},
			},
		},
		watched: map[string]*darwinNode{
			oldDir: {
				fd:    dirFD,
				path:  oldDir,
				isDir: true,
				roots: map[string]struct{}{oldDir: {}},
			},
			oldFile: {
				fd:    fileFD,
				path:  oldFile,
				isDir: false,
				roots: map[string]struct{}{oldDir: {}},
			},
		},
		fdToPath: map[uintptr]string{
			uintptr(dirFD):  oldDir,
			uintptr(fileFD): oldFile,
		},
		snapshots: map[string]map[string]entryMeta{
			oldDir: {
				"file.txt": {inode: 1},
			},
		},
		errors: make(chan error, 1),
	}

	w.renamePrefix(oldDir, newDir)
	if _, ok := w.watched[newDir]; !ok {
		t.Fatal("renamePrefix did not move watched directory")
	}
	if _, ok := w.rootNodes[oldDir][newDir]; !ok {
		t.Fatal("renamePrefix did not move root node membership")
	}
	if _, ok := w.snapshots[newDir]; !ok {
		t.Fatal("renamePrefix did not move snapshot")
	}

	w.renamePath(filepath.Join(newDir, "file.txt"), newFile)
	if _, ok := w.watched[newFile]; !ok {
		t.Fatal("renamePath did not move watched file")
	}

	w.emitError(os.ErrClosed)
	select {
	case err := <-w.errors:
		if err == nil {
			t.Fatal("emitError() delivered nil")
		}
	default:
		t.Fatal("emitError() did not deliver error")
	}
	w.errors <- os.ErrExist
	w.emitError(os.ErrInvalid)

	w.closed = true
	if !w.isClosed() {
		t.Fatal("isClosed() = false, want true")
	}
	w.closed = false

	if got := w.rootTarget(oldDir); got.Path != oldDir {
		t.Fatalf("rootTarget() = %+v, want root %q", got, oldDir)
	}

	w.removePath(newFile)
	if _, ok := w.watched[newFile]; ok {
		t.Fatal("removePath did not unregister file")
	}

	w.removePrefix(newDir)
	if _, ok := w.watched[newDir]; ok {
		t.Fatal("removePrefix did not unregister directory")
	}

	if err := w.Remove(oldDir); err != nil {
		t.Fatalf("Remove(root) error = %v", err)
	}
	if err := w.Remove(oldDir); !os.IsNotExist(err) {
		t.Fatalf("second Remove(root) error = %v, want not exist", err)
	}
}

func TestDarwinSnapshotHelpers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	keep := filepath.Join(root, "keep.txt")
	skip := filepath.Join(root, "skip.txt")
	if err := os.WriteFile(keep, []byte("keep"), 0o644); err != nil {
		t.Fatalf("WriteFile(keep) error = %v", err)
	}
	if err := os.WriteFile(skip, []byte("skip"), 0o644); err != nil {
		t.Fatalf("WriteFile(skip) error = %v", err)
	}

	w := &darwinWatcher{
		cfg: Config{
			ShouldExclude: func(path string, isDir bool) bool {
				return filepath.Base(path) == "skip.txt"
			},
		},
	}

	snapshot := w.readDirSnapshot(root)
	if _, ok := snapshot["keep.txt"]; !ok {
		t.Fatal("readDirSnapshot did not include keep.txt")
	}
	if _, ok := snapshot["skip.txt"]; ok {
		t.Fatal("readDirSnapshot unexpectedly included excluded file")
	}

	if inode := inodeForPath(keep); inode == 0 {
		t.Fatal("inodeForPath(existing) = 0, want non-zero inode")
	}
	if inode := inodeForPath(filepath.Join(root, "missing.txt")); inode != 0 {
		t.Fatalf("inodeForPath(missing) = %d, want 0", inode)
	}

	copied := copySnapshot(snapshot)
	copied["keep.txt"] = entryMeta{inode: 99}
	if snapshot["keep.txt"].inode == copied["keep.txt"].inode {
		t.Fatal("copySnapshot() did not create an independent copy")
	}
}

func TestDarwinWatcherRescanDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	oldName := filepath.Join(root, "before.txt")
	removeName := filepath.Join(root, "remove.txt")
	if err := os.WriteFile(oldName, []byte("before"), 0o644); err != nil {
		t.Fatalf("WriteFile(before) error = %v", err)
	}
	if err := os.WriteFile(removeName, []byte("remove"), 0o644); err != nil {
		t.Fatalf("WriteFile(remove) error = %v", err)
	}

	kq, err := unix.Kqueue()
	if err != nil {
		t.Fatalf("unix.Kqueue() error = %v", err)
	}
	defer func() {
		_ = unix.Close(kq)
	}()

	rootFD := openPathForTest(t, root, true)
	oldFD := openPathForTest(t, oldName, false)
	removeFD := openPathForTest(t, removeName, false)

	w := &darwinWatcher{
		cfg: Config{},
		kq:  kq,
		roots: map[string]Target{
			root: {Path: root, WatchPath: root, IsDir: true, Recursive: true},
		},
		rootNodes: map[string]map[string]struct{}{
			root: {
				root:       {},
				oldName:    {},
				removeName: {},
			},
		},
		watched: map[string]*darwinNode{
			root: {
				fd:    rootFD,
				path:  root,
				isDir: true,
				roots: map[string]struct{}{root: {}},
			},
			oldName: {
				fd:    oldFD,
				path:  oldName,
				isDir: false,
				roots: map[string]struct{}{root: {}},
			},
			removeName: {
				fd:    removeFD,
				path:  removeName,
				isDir: false,
				roots: map[string]struct{}{root: {}},
			},
		},
		fdToPath: map[uintptr]string{
			uintptr(rootFD):   root,
			uintptr(oldFD):    oldName,
			uintptr(removeFD): removeName,
		},
		snapshots: map[string]map[string]entryMeta{},
		events:    make(chan Event, 16),
		errors:    make(chan error, 1),
	}
	w.snapshots[root] = w.readDirSnapshot(root)

	renamedName := filepath.Join(root, "after.txt")
	if err := os.Rename(oldName, renamedName); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if err := os.Remove(removeName); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	addedName := filepath.Join(root, "added.txt")
	if err := os.WriteFile(addedName, []byte("added"), 0o644); err != nil {
		t.Fatalf("WriteFile(added) error = %v", err)
	}
	addedDir := filepath.Join(root, "dir")
	if err := os.Mkdir(addedDir, 0o755); err != nil {
		t.Fatalf("Mkdir(dir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(addedDir, "nested.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatalf("WriteFile(nested) error = %v", err)
	}

	w.rescanDir(root)

	var sawRename, sawRemove, sawAddedFile, sawAddedDir bool
	timeout := time.After(time.Second)
	for !sawRename || !sawRemove || !sawAddedFile || !sawAddedDir {
		select {
		case evt := <-w.events:
			switch {
			case evt.Path == renamedName && evt.OldPath == oldName && evt.Op == OpRename:
				sawRename = true
			case evt.Path == removeName && evt.Op == OpRemove:
				sawRemove = true
			case evt.Path == addedName && evt.Op == OpCreate:
				sawAddedFile = true
			case evt.Path == addedDir && evt.Op == OpCreate && evt.IsDir:
				sawAddedDir = true
			}
		case <-timeout:
			t.Fatalf("timed out waiting for rescan events: rename=%v remove=%v addFile=%v addDir=%v", sawRename, sawRemove, sawAddedFile, sawAddedDir)
		}
	}

	if _, ok := w.watched[renamedName]; !ok {
		t.Fatal("rescanDir did not update watched file path after rename")
	}
	if _, ok := w.watched[filepath.Join(addedDir, "nested.txt")]; !ok {
		t.Fatal("rescanDir did not enroll nested file in added directory")
	}

	w.removePrefix(root)
}

func TestDarwinWatcherHandle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	filePath := filepath.Join(root, "file.txt")
	if err := os.WriteFile(filePath, []byte("file"), 0o644); err != nil {
		t.Fatalf("WriteFile(file) error = %v", err)
	}
	dirPath := filepath.Join(root, "dir")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatalf("Mkdir(dir) error = %v", err)
	}

	rootFD := openPathForTest(t, root, true)
	fileFD := openPathForTest(t, filePath, false)
	dirFD := openPathForTest(t, dirPath, true)

	w := &darwinWatcher{
		roots: map[string]Target{
			root: {Path: root, WatchPath: root, IsDir: true, Recursive: true},
		},
		rootNodes: map[string]map[string]struct{}{
			root: {
				root:     {},
				filePath: {},
				dirPath:  {},
			},
		},
		watched: map[string]*darwinNode{
			root: {
				fd:    rootFD,
				path:  root,
				isDir: true,
				roots: map[string]struct{}{root: {}},
			},
			filePath: {
				fd:    fileFD,
				path:  filePath,
				isDir: false,
				roots: map[string]struct{}{root: {}},
			},
			dirPath: {
				fd:    dirFD,
				path:  dirPath,
				isDir: true,
				roots: map[string]struct{}{root: {}},
			},
		},
		fdToPath: map[uintptr]string{
			uintptr(rootFD): root,
			uintptr(fileFD): filePath,
			uintptr(dirFD):  dirPath,
		},
		snapshots: map[string]map[string]entryMeta{
			root:    {},
			dirPath: {},
		},
		events: make(chan Event, 16),
		errors: make(chan error, 1),
	}

	w.handle(unix.Kevent_t{Ident: uint64(fileFD), Fflags: unix.NOTE_WRITE})
	w.handle(unix.Kevent_t{Ident: uint64(dirFD), Fflags: unix.NOTE_ATTRIB})
	w.handle(unix.Kevent_t{Ident: 999999, Fflags: unix.NOTE_WRITE})

	child := filepath.Join(dirPath, "child.txt")
	if err := os.WriteFile(child, []byte("child"), 0o644); err != nil {
		t.Fatalf("WriteFile(child) error = %v", err)
	}
	w.handle(unix.Kevent_t{Ident: uint64(dirFD), Fflags: unix.NOTE_WRITE})

	if err := os.Remove(filePath); err != nil {
		t.Fatalf("Remove(file) error = %v", err)
	}
	w.handle(unix.Kevent_t{Ident: uint64(fileFD), Fflags: unix.NOTE_DELETE})

	if err := os.RemoveAll(dirPath); err != nil {
		t.Fatalf("RemoveAll(dir) error = %v", err)
	}
	w.handle(unix.Kevent_t{Ident: uint64(dirFD), Fflags: unix.NOTE_DELETE})

	var sawWrite, sawChmod, sawCreate, sawFileRemove, sawDirRemove bool
	timeout := time.After(time.Second)
	for !sawWrite || !sawChmod || !sawCreate || !sawFileRemove || !sawDirRemove {
		select {
		case evt := <-w.events:
			switch {
			case evt.Path == filePath && evt.Op == OpWrite:
				sawWrite = true
			case evt.Path == dirPath && evt.Op == OpChmod:
				sawChmod = true
			case evt.Path == child && evt.Op == OpCreate:
				sawCreate = true
			case evt.Path == filePath && evt.Op == OpRemove:
				sawFileRemove = true
			case evt.Path == dirPath && evt.Op == OpRemove && evt.IsDir:
				sawDirRemove = true
			}
		case <-timeout:
			t.Fatalf("timed out waiting for handle events: write=%v chmod=%v create=%v fileRemove=%v dirRemove=%v", sawWrite, sawChmod, sawCreate, sawFileRemove, sawDirRemove)
		}
	}

	w.removePrefix(root)
}

func mustMkdirDarwinTest(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir(%q) error = %v", path, err)
	}
}

func mustWriteFileDarwinTest(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func newDarwinWatcherForTests(cfg Config) *darwinWatcher {
	return &darwinWatcher{
		cfg:       cfg,
		kq:        1,
		roots:     make(map[string]Target),
		rootNodes: make(map[string]map[string]struct{}),
		watched:   make(map[string]*darwinNode),
		fdToPath:  make(map[uintptr]string),
		snapshots: make(map[string]map[string]entryMeta),
		events:    make(chan Event, 32),
		errors:    make(chan error, 32),
		done:      make(chan struct{}),
	}
}

func stubDarwinSyscalls(t *testing.T) {
	t.Helper()

	prevKqueue := darwinKqueue
	prevKevent := darwinKevent
	prevOpen := darwinOpen
	prevClose := darwinClose
	prevReadDir := darwinReadDir
	prevLstat := darwinLstat
	var nextFD int

	darwinKqueue = func() (int, error) { return 1, nil }
	darwinKevent = func(int, []unix.Kevent_t, []unix.Kevent_t, *unix.Timespec) (int, error) { return 0, nil }
	darwinOpen = func(string, int, uint32) (int, error) {
		nextFD++
		return nextFD, nil
	}
	darwinClose = func(int) error { return nil }
	darwinReadDir = os.ReadDir
	darwinLstat = unix.Lstat

	t.Cleanup(func() {
		darwinKqueue = prevKqueue
		darwinKevent = prevKevent
		darwinOpen = prevOpen
		darwinClose = prevClose
		darwinReadDir = prevReadDir
		darwinLstat = prevLstat
	})
}

func TestNewReturnsKqueueError(t *testing.T) {
	prev := darwinKqueue
	t.Cleanup(func() { darwinKqueue = prev })

	want := errors.New("kqueue boom")
	darwinKqueue = func() (int, error) { return 0, want }

	_, err := New(Config{})
	if err == nil || !errors.Is(err, want) {
		t.Fatalf("New() error = %v, want %v", err, want)
	}
}

func TestDarwinWatcherRunHandlesEINTRAndError(t *testing.T) {
	stubDarwinSyscalls(t)
	w := newDarwinWatcherForTests(Config{})
	calls := 0
	darwinKevent = func(int, []unix.Kevent_t, []unix.Kevent_t, *unix.Timespec) (int, error) {
		calls++
		switch calls {
		case 1:
			return 0, unix.EINTR
		case 2:
			return 0, unix.EBADF
		default:
			w.mu.Lock()
			w.closed = true
			w.mu.Unlock()
			return 0, unix.EBADF
		}
	}

	w.run()

	select {
	case err := <-w.errors:
		if err == nil || err.Error() == "" {
			t.Fatalf("errors channel = %v, want emitted kevent error", err)
		}
	default:
		t.Fatal("expected kevent error to be emitted")
	}
}

func TestDarwinWatcherAddClosed(t *testing.T) {
	stubDarwinSyscalls(t)
	w := newDarwinWatcherForTests(Config{})
	w.closed = true

	err := w.Add(Target{Path: "/tmp/root", WatchPath: "/tmp/root", IsDir: true})
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("Add() error = %v, want %v", err, os.ErrClosed)
	}
}

func TestDarwinWatcherAddDuplicateRoot(t *testing.T) {
	stubDarwinSyscalls(t)
	w := newDarwinWatcherForTests(Config{})
	target := Target{Path: "/tmp/root", WatchPath: "/tmp/root", IsDir: true}
	w.roots[target.Path] = target
	w.rootNodes[target.Path] = map[string]struct{}{}

	if err := w.Add(target); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
}

func TestDarwinWatcherAddFileCleanupOnSecondAddFailure(t *testing.T) {
	stubDarwinSyscalls(t)
	w := newDarwinWatcherForTests(Config{})
	target := Target{Path: "/tmp/root/file.txt", WatchPath: "/tmp/root", IsDir: false}
	failPath := target.Path
	darwinOpen = func(path string, flag int, perm uint32) (int, error) {
		if path == failPath {
			return 0, errors.New("open boom")
		}
		return 1, nil
	}

	err := w.Add(target)
	if err == nil || err.Error() == "" {
		t.Fatal("expected Add() to fail on second addPath")
	}
	if _, ok := w.roots[target.Path]; ok {
		t.Fatal("expected failed add to remove root registration")
	}
	if _, ok := w.rootNodes[target.Path]; ok {
		t.Fatal("expected failed add to remove root node registration")
	}
}

func TestDarwinWatcherAddFileCleanupOnFirstAddFailure(t *testing.T) {
	stubDarwinSyscalls(t)
	w := newDarwinWatcherForTests(Config{})
	target := Target{Path: "/tmp/root/file.txt", WatchPath: "/tmp/root", IsDir: false}
	failPath := target.WatchPath
	darwinOpen = func(path string, flag int, perm uint32) (int, error) {
		if path == failPath {
			return 0, errors.New("open boom")
		}
		return 1, nil
	}

	err := w.Add(target)
	if err == nil || err.Error() == "" {
		t.Fatal("expected Add() to fail on first addPath")
	}
	if _, ok := w.roots[target.Path]; ok {
		t.Fatal("expected failed add to remove root registration")
	}
}

func TestDarwinWatcherAddDirectoryCleanupOnEnrollFailure(t *testing.T) {
	stubDarwinSyscalls(t)
	root := t.TempDir()
	target := Target{Path: filepath.Join(root, "missing"), WatchPath: filepath.Join(root, "missing"), IsDir: true, Recursive: true}
	w := newDarwinWatcherForTests(Config{})

	err := w.Add(target)
	if err == nil {
		t.Fatal("expected Add() to fail for missing recursive directory")
	}
	if _, ok := w.roots[target.Path]; ok {
		t.Fatal("expected failed directory add to remove root registration")
	}
}

func TestDarwinWatcherEnrollDirectoryTargetReadDirError(t *testing.T) {
	stubDarwinSyscalls(t)
	root := t.TempDir()
	w := newDarwinWatcherForTests(Config{})
	w.roots[root] = Target{Path: root, WatchPath: root, IsDir: true}
	w.rootNodes[root] = map[string]struct{}{}
	want := errors.New("readdir boom")
	darwinReadDir = func(string) ([]os.DirEntry, error) { return nil, want }

	err := w.enrollDirectoryTarget(Target{Path: root, WatchPath: root, IsDir: true})
	if !errors.Is(err, want) {
		t.Fatalf("enrollDirectoryTarget() error = %v, want %v", err, want)
	}
}

func TestDarwinWatcherEnrollDirectoryTargetFlatDirectory(t *testing.T) {
	stubDarwinSyscalls(t)
	root := t.TempDir()
	mustMkdirDarwinTest(t, filepath.Join(root, "childdir"))
	mustWriteFileDarwinTest(t, filepath.Join(root, "keep.txt"))
	mustWriteFileDarwinTest(t, filepath.Join(root, "skip.txt"))

	w := newDarwinWatcherForTests(Config{
		ShouldExclude: func(path string, isDir bool) bool {
			return filepath.Base(path) == "skip.txt"
		},
	})
	w.roots[root] = Target{Path: root, WatchPath: root, IsDir: true}
	w.rootNodes[root] = map[string]struct{}{}

	if err := w.enrollDirectoryTarget(Target{Path: root, WatchPath: root, IsDir: true, Recursive: false}); err != nil {
		t.Fatalf("enrollDirectoryTarget() error = %v", err)
	}
	if _, ok := w.watched[root]; !ok {
		t.Fatal("expected root directory to be watched")
	}
	if _, ok := w.watched[filepath.Join(root, "keep.txt")]; !ok {
		t.Fatal("expected flat file child to be watched")
	}
	if _, ok := w.watched[filepath.Join(root, "skip.txt")]; ok {
		t.Fatal("did not expect excluded child file to be watched")
	}
	if _, ok := w.watched[filepath.Join(root, "childdir")]; ok {
		t.Fatal("did not expect child directory to be watched for non-recursive target")
	}
}

func TestDarwinWatcherAddPathBranches(t *testing.T) {
	t.Run("open error", func(t *testing.T) {
		stubDarwinSyscalls(t)
		w := newDarwinWatcherForTests(Config{})
		root := "/tmp/root"
		path := "/tmp/root/file.txt"
		w.roots[root] = Target{Path: root}
		w.rootNodes[root] = map[string]struct{}{}
		darwinOpen = func(string, int, uint32) (int, error) { return 0, errors.New("open boom") }

		err := w.addPath(root, path, false)
		if err == nil || !strings.Contains(err.Error(), "open") {
			t.Fatalf("addPath() error = %v, want open error", err)
		}
	})

	t.Run("excluded non-root", func(t *testing.T) {
		stubDarwinSyscalls(t)
		root := "/tmp/root"
		path := "/tmp/root/skip.txt"
		w := newDarwinWatcherForTests(Config{
			ShouldExclude: func(candidate string, isDir bool) bool { return candidate == path },
		})
		w.roots[root] = Target{Path: root}
		w.rootNodes[root] = map[string]struct{}{}

		if err := w.addPath(root, path, false); err != nil {
			t.Fatalf("addPath() error = %v", err)
		}
		if _, ok := w.watched[path]; ok {
			t.Fatal("did not expect excluded non-root path to be watched")
		}
	})

	t.Run("register error", func(t *testing.T) {
		stubDarwinSyscalls(t)
		w := newDarwinWatcherForTests(Config{})
		root := "/tmp/root"
		path := "/tmp/root/file.txt"
		w.roots[root] = Target{Path: root}
		w.rootNodes[root] = map[string]struct{}{}
		darwinKevent = func(int, []unix.Kevent_t, []unix.Kevent_t, *unix.Timespec) (int, error) {
			return 0, errors.New("register boom")
		}

		err := w.addPath(root, path, false)
		if err == nil || !strings.Contains(err.Error(), "register") {
			t.Fatalf("addPath() error = %v, want register error", err)
		}
	})

	t.Run("closed after register", func(t *testing.T) {
		stubDarwinSyscalls(t)
		w := newDarwinWatcherForTests(Config{})
		root := "/tmp/root"
		path := "/tmp/root/file.txt"
		w.roots[root] = Target{Path: root}
		w.rootNodes[root] = map[string]struct{}{}
		w.closed = true

		err := w.addPath(root, path, false)
		if !errors.Is(err, os.ErrClosed) {
			t.Fatalf("addPath() error = %v, want %v", err, os.ErrClosed)
		}
	})

	t.Run("existing node", func(t *testing.T) {
		stubDarwinSyscalls(t)
		w := newDarwinWatcherForTests(Config{})
		root := "/tmp/root"
		path := "/tmp/root/file.txt"
		w.roots[root] = Target{Path: root}
		w.rootNodes[root] = map[string]struct{}{}
		w.watched[path] = &darwinNode{path: path, roots: map[string]struct{}{"other": {}}}

		if err := w.addPath(root, path, false); err != nil {
			t.Fatalf("addPath() error = %v", err)
		}
		if _, ok := w.watched[path].roots[root]; !ok {
			t.Fatal("expected root to be attached to existing node")
		}
		if _, ok := w.rootNodes[root][path]; !ok {
			t.Fatal("expected rootNodes to record the existing node")
		}
	})

	t.Run("existing node after register", func(t *testing.T) {
		stubDarwinSyscalls(t)
		w := newDarwinWatcherForTests(Config{})
		root := "/tmp/root"
		path := "/tmp/root/file.txt"
		w.roots[root] = Target{Path: root}
		w.rootNodes[root] = map[string]struct{}{}
		darwinKevent = func(int, []unix.Kevent_t, []unix.Kevent_t, *unix.Timespec) (int, error) {
			w.mu.Lock()
			w.watched[path] = &darwinNode{path: path, roots: map[string]struct{}{"other": {}}}
			w.mu.Unlock()
			return 0, nil
		}

		if err := w.addPath(root, path, false); err != nil {
			t.Fatalf("addPath() error = %v", err)
		}
		if _, ok := w.watched[path].roots[root]; !ok {
			t.Fatal("expected root to be attached after late duplicate detection")
		}
	})
}

func TestDarwinWatcherRemoveBranches(t *testing.T) {
	stubDarwinSyscalls(t)
	w := newDarwinWatcherForTests(Config{})

	if err := w.Remove("/tmp/missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Remove() error = %v, want %v", err, os.ErrNotExist)
	}

	root := "/tmp/root"
	path := "/tmp/root/file.txt"
	w.roots[root] = Target{Path: root}
	w.rootNodes[root] = map[string]struct{}{path: {}, "/tmp/root/missing-node": {}}
	w.watched[path] = &darwinNode{fd: 1, path: path, roots: map[string]struct{}{root: {}}}

	if err := w.Remove(root); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, ok := w.roots[root]; ok {
		t.Fatal("expected root to be removed")
	}
}

func TestDarwinWatcherRenamePrefixMovesChildrenAndSnapshots(t *testing.T) {
	stubDarwinSyscalls(t)
	w := newDarwinWatcherForTests(Config{})
	root := "/tmp/root"
	oldDir := filepath.Join(root, "old")
	oldChild := filepath.Join(oldDir, "child.txt")
	newDir := filepath.Join(root, "new")
	newChild := filepath.Join(newDir, "child.txt")
	w.rootNodes[root] = map[string]struct{}{oldDir: {}, oldChild: {}}
	w.watched[oldDir] = &darwinNode{fd: 1, path: oldDir, isDir: true, roots: map[string]struct{}{root: {}}}
	w.watched[oldChild] = &darwinNode{fd: 2, path: oldChild, isDir: false, roots: map[string]struct{}{root: {}}}
	w.fdToPath[1] = oldDir
	w.fdToPath[2] = oldChild
	w.snapshots[oldDir] = map[string]entryMeta{"child.txt": {inode: 1}}

	w.renamePrefix(oldDir, newDir)

	if _, ok := w.watched[newDir]; !ok {
		t.Fatal("expected renamed directory node")
	}
	if _, ok := w.watched[newChild]; !ok {
		t.Fatal("expected renamed child node")
	}
	if _, ok := w.snapshots[newDir]; !ok {
		t.Fatal("expected snapshot to move with renamed directory")
	}
	if _, ok := w.rootNodes[root][newChild]; !ok {
		t.Fatal("expected rootNodes to track renamed child")
	}
}

func TestDarwinWatcherRescanDirUpdatesState(t *testing.T) {
	stubDarwinSyscalls(t)
	root := t.TempDir()
	oldPath := filepath.Join(root, "old.txt")
	renamedPath := filepath.Join(root, "renamed.txt")
	removedPath := filepath.Join(root, "gone.txt")
	addedPath := filepath.Join(root, "added.txt")
	dirAdded := filepath.Join(root, "diradded")
	dirAddedChild := filepath.Join(dirAdded, "nested.txt")

	mustWriteFileDarwinTest(t, oldPath)
	mustWriteFileDarwinTest(t, removedPath)
	if err := os.Rename(oldPath, renamedPath); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if err := os.Remove(removedPath); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	mustWriteFileDarwinTest(t, addedPath)
	mustMkdirDarwinTest(t, dirAdded)
	mustWriteFileDarwinTest(t, dirAddedChild)

	w := newDarwinWatcherForTests(Config{})
	w.roots[root] = Target{Path: root, WatchPath: root, IsDir: true, Recursive: true}
	w.rootNodes[root] = map[string]struct{}{root: {}, oldPath: {}, removedPath: {}}
	w.watched[root] = &darwinNode{fd: 1, path: root, isDir: true, roots: map[string]struct{}{root: {}}}
	w.watched[oldPath] = &darwinNode{fd: 2, path: oldPath, roots: map[string]struct{}{root: {}}}
	w.watched[removedPath] = &darwinNode{fd: 3, path: removedPath, roots: map[string]struct{}{root: {}}}
	w.fdToPath[1] = root
	w.fdToPath[2] = oldPath
	w.fdToPath[3] = removedPath
	w.snapshots[root] = map[string]entryMeta{
		"old.txt":  {inode: inodeForPath(renamedPath)},
		"gone.txt": {inode: 999},
	}

	w.rescanDir(root)

	if _, ok := w.watched[renamedPath]; !ok {
		t.Fatal("expected renamed file to be tracked")
	}
	if _, ok := w.watched[removedPath]; ok {
		t.Fatal("did not expect removed file to remain tracked")
	}
	if _, ok := w.watched[addedPath]; !ok {
		t.Fatal("expected added file to be tracked")
	}
	if _, ok := w.watched[dirAdded]; !ok {
		t.Fatal("expected added directory to be tracked for recursive root")
	}
	if _, ok := w.watched[dirAddedChild]; !ok {
		t.Fatal("expected nested added file to be tracked for recursive root")
	}
}

func TestDarwinWatcherRenamePathMissingIsNoop(t *testing.T) {
	stubDarwinSyscalls(t)
	w := newDarwinWatcherForTests(Config{})
	w.renamePath("/tmp/missing", "/tmp/new")
}

func TestDarwinWatcherReadDirSnapshotReadDirError(t *testing.T) {
	stubDarwinSyscalls(t)
	w := newDarwinWatcherForTests(Config{})
	darwinReadDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("boom") }

	snapshot := w.readDirSnapshot("/tmp/missing")
	if len(snapshot) != 0 {
		t.Fatalf("readDirSnapshot() = %v, want empty snapshot", snapshot)
	}
}

func TestDarwinWatcherReadDirSnapshotSkipsExcludedEntries(t *testing.T) {
	stubDarwinSyscalls(t)
	root := t.TempDir()
	mustWriteFileDarwinTest(t, filepath.Join(root, "keep.txt"))
	mustWriteFileDarwinTest(t, filepath.Join(root, "skip.txt"))
	w := newDarwinWatcherForTests(Config{
		ShouldExclude: func(path string, isDir bool) bool {
			return filepath.Base(path) == "skip.txt"
		},
	})

	snapshot := w.readDirSnapshot(root)
	if _, ok := snapshot["keep.txt"]; !ok {
		t.Fatal("expected keep.txt in snapshot")
	}
	if _, ok := snapshot["skip.txt"]; ok {
		t.Fatal("did not expect excluded file in snapshot")
	}
}

func TestDarwinWatcherAddRecursiveDirectorySkipsExcludedEntries(t *testing.T) {
	stubDarwinSyscalls(t)
	root := t.TempDir()
	mustMkdirDarwinTest(t, filepath.Join(root, "keepdir"))
	mustMkdirDarwinTest(t, filepath.Join(root, "skipdir"))
	mustWriteFileDarwinTest(t, filepath.Join(root, "keep.txt"))
	mustWriteFileDarwinTest(t, filepath.Join(root, "skip.txt"))
	mustWriteFileDarwinTest(t, filepath.Join(root, "keepdir", "nested.txt"))
	mustWriteFileDarwinTest(t, filepath.Join(root, "skipdir", "nested.txt"))

	w := newDarwinWatcherForTests(Config{
		ShouldExclude: func(path string, isDir bool) bool {
			base := filepath.Base(path)
			return base == "skipdir" || base == "skip.txt"
		},
	})

	target := Target{Path: root, WatchPath: root, IsDir: true, Recursive: true}
	if err := w.Add(target); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if _, ok := w.watched[filepath.Join(root, "keep.txt")]; !ok {
		t.Fatal("expected keep file to be watched")
	}
	if _, ok := w.watched[filepath.Join(root, "skip.txt")]; ok {
		t.Fatal("did not expect excluded file to be watched")
	}
	if _, ok := w.watched[filepath.Join(root, "skipdir")]; ok {
		t.Fatal("did not expect excluded directory to be watched")
	}
}
