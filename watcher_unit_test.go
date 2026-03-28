package gaze

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"go.ofkm.dev/gaze/internal/backend"
	"go.ofkm.dev/gaze/internal/filter"
	"go.ofkm.dev/gaze/internal/queue"
	"go.ofkm.dev/gaze/internal/tree"
)

type stubDriver struct {
	addErr    error
	removeErr error
	closeErr  error

	added   []backend.Target
	removed []string

	events chan backend.Event
	errors chan error

	mu         sync.Mutex
	closeCalls int
}

func newStubDriver() *stubDriver {
	return &stubDriver{
		events: make(chan backend.Event, 16),
		errors: make(chan error, 16),
	}
}

func (d *stubDriver) Add(target backend.Target) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.added = append(d.added, target)
	return d.addErr
}

func (d *stubDriver) Remove(path string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.removed = append(d.removed, path)
	return d.removeErr
}

func (d *stubDriver) Events() <-chan backend.Event {
	return d.events
}

func (d *stubDriver) Errors() <-chan error {
	return d.errors
}

func (d *stubDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closeCalls++
	return d.closeErr
}

func newUnitWatcher(t *testing.T, cfg Config, driver backend.Watcher) *Watcher {
	t.Helper()

	matcher, err := filter.New(filter.Config{
		Prefixes: cfg.ExcludePrefixes,
		Globs:    cfg.ExcludeGlobs,
		Exclude: func(path string, isDir bool) bool {
			if cfg.Exclude == nil {
				return false
			}
			return cfg.Exclude(PathInfo{
				Path:  path,
				Base:  filepath.Base(path),
				IsDir: isDir,
			})
		},
	})
	if err != nil {
		t.Fatalf("filter.New() error = %v", err)
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	}

	return &Watcher{
		cfg:     resolveConfig(cfg),
		matcher: matcher,
		index:   tree.New(),
		driver:  driver,
		queue:   queue.New[Event](8),
		logger:  logger,
		done:    make(chan struct{}),
	}
}

func TestOpString(t *testing.T) {
	t.Parallel()

	if got := (Op(0)).String(); got != "none" {
		t.Fatalf("Op(0).String() = %q, want %q", got, "none")
	}

	got := (OpCreate | OpWrite | OpRemove | OpRename | OpChmod | OpOverflow).String()
	want := "create|write|remove|rename|chmod|overflow"
	if got != want {
		t.Fatalf("combined Op.String() = %q, want %q", got, want)
	}
}

func TestResolveConfigAndRecursionMode(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	cfg := resolveConfig(Config{
		Recursion:       RecursionEnabled,
		ExcludeGlobs:    []string{"*.tmp"},
		ExcludePrefixes: []string{"/tmp/cache"},
		OnEvent:         func(Event) {},
		OnError:         func(error) {},
		Logger:          logger,
		Ops:             OpCreate,
		QueueCapacity:   0,
		FollowSymlinks:  true,
	})

	if !cfg.recursiveEnabled(false) {
		t.Fatal("Config{Recursion: RecursionEnabled}.recursiveEnabled(false) = false, want true")
	}
	if cfg.QueueCapacity != 1024 {
		t.Fatalf("QueueCapacity = %d, want 1024", cfg.QueueCapacity)
	}
	if !cfg.Ops.Has(OpCreate) || !cfg.Ops.Has(OpOverflow) {
		t.Fatalf("Ops = %v, want create plus overflow", cfg.Ops)
	}
	if cfg.Logger != logger {
		t.Fatal("Logger was not preserved")
	}
	if (Config{Recursion: RecursionDisabled}).recursiveEnabled(true) {
		t.Fatal("RecursionDisabled should override default")
	}

	defaulted := resolveConfig(Config{})
	if defaulted.Logger == nil {
		t.Fatal("default logger = nil, want non-nil")
	}
	if defaulted.Ops != allOps {
		t.Fatalf("default Ops = %v, want %v", defaulted.Ops, allOps)
	}
	if !(Config{}).recursiveEnabled(true) {
		t.Fatal("default recursion should preserve provided default")
	}
}

func TestNormalizePathAndPrepareTarget(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	filePath := filepath.Join(root, "config.yaml")
	if err := os.WriteFile(filePath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	w := newUnitWatcher(t, Config{}, newStubDriver())

	if _, err := w.normalizePath("   "); err == nil {
		t.Fatal("normalizePath(empty) error = nil, want error")
	}

	normalized, err := w.normalizePath(filepath.Join(root, ".", "config.yaml"))
	if err != nil {
		t.Fatalf("normalizePath() error = %v", err)
	}
	if normalized != filePath {
		t.Fatalf("normalizePath() = %q, want %q", normalized, filePath)
	}

	target, err := w.prepareTarget(filePath)
	if err != nil {
		t.Fatalf("prepareTarget(file) error = %v", err)
	}
	if target.Path != filePath || target.WatchPath != root || target.IsDir || target.Recursive {
		t.Fatalf("prepareTarget(file) = %+v, want file target", target)
	}
}

func TestPrepareTargetExcludeAndSymlinkHandling(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatalf("Mkdir(real) error = %v", err)
	}

	excludedDir := filepath.Join(root, "skip.tmp")
	if err := os.Mkdir(excludedDir, 0o755); err != nil {
		t.Fatalf("Mkdir(excluded) error = %v", err)
	}

	link := filepath.Join(root, "linked")
	if err := os.Symlink(realDir, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink setup unavailable on windows: %v", err)
		}
		t.Fatalf("Symlink() error = %v", err)
	}

	w := newUnitWatcher(t, Config{ExcludeGlobs: []string{"*.tmp"}}, newStubDriver())
	if _, err := w.prepareTarget(excludedDir); err == nil || !strings.Contains(err.Error(), "excluded root") {
		t.Fatalf("prepareTarget(excluded) error = %v, want excluded root error", err)
	}
	if _, err := w.prepareTarget(link); err == nil || !strings.Contains(err.Error(), "FollowSymlinks") {
		t.Fatalf("prepareTarget(symlink) error = %v, want follow symlinks error", err)
	}

	wFollow := newUnitWatcher(t, Config{FollowSymlinks: true}, newStubDriver())
	target, err := wFollow.prepareTarget(link)
	if err != nil {
		t.Fatalf("prepareTarget(symlink follow) error = %v", err)
	}
	resolvedRealDir, err := filepath.EvalSymlinks(realDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(realDir) error = %v", err)
	}
	if target.Path != resolvedRealDir || target.WatchPath != resolvedRealDir || !target.IsDir || !target.Recursive {
		t.Fatalf("prepareTarget(symlink follow) = %+v, want resolved directory target", target)
	}
}

func TestWatcherAddRemoveAndClose(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	driver := newStubDriver()
	w := newUnitWatcher(t, Config{}, driver)
	go w.runEvents()

	if err := w.Add(root); err != nil {
		t.Fatalf("Add(directory) error = %v", err)
	}
	if len(driver.added) != 1 || driver.added[0].Path != root {
		t.Fatalf("driver added = %+v, want target rooted at %q", driver.added, root)
	}

	driver.addErr = errors.New("driver add failed")
	otherRoot := filepath.Join(t.TempDir(), "other")
	if err := os.Mkdir(otherRoot, 0o755); err != nil {
		t.Fatalf("Mkdir(otherRoot) error = %v", err)
	}
	if err := w.Add(otherRoot); !errors.Is(err, driver.addErr) {
		t.Fatalf("Add(driver failure) error = %v, want %v", err, driver.addErr)
	}
	if _, ok := w.index.Remove(otherRoot); ok {
		t.Fatal("index still contained target after Add failure")
	}
	driver.addErr = nil

	if err := w.Remove(root); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if len(driver.removed) != 1 || driver.removed[0] != root {
		t.Fatalf("driver removed = %+v, want %q", driver.removed, root)
	}

	if err := w.Remove(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("second Remove() error = %v, want %v", err, os.ErrNotExist)
	}

	if err := w.Add(root); err != nil {
		t.Fatalf("Add(directory again) error = %v", err)
	}
	driver.removeErr = errors.New("driver remove failed")
	if err := w.Remove(root); !errors.Is(err, driver.removeErr) {
		t.Fatalf("Remove(driver failure) error = %v, want %v", err, driver.removeErr)
	}
	driver.removeErr = nil

	driver.closeErr = errors.New("close failed")
	if err := w.Close(); !errors.Is(err, driver.closeErr) {
		t.Fatalf("Close() error = %v, want %v", err, driver.closeErr)
	}
	if err := w.Close(); !errors.Is(err, driver.closeErr) {
		t.Fatalf("second Close() error = %v, want %v", err, driver.closeErr)
	}
	if driver.closeCalls != 1 {
		t.Fatalf("driver close calls = %d, want 1", driver.closeCalls)
	}
}

func TestWatcherRunBackendAndDispatch(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")

	driver := newStubDriver()
	events := make(chan Event, 4)
	errs := make(chan error, 4)
	w := newUnitWatcher(t, Config{
		OnEvent: func(evt Event) {
			events <- evt
		},
		OnError: func(err error) {
			errs <- err
		},
	}, driver)
	if err := w.index.Add(tree.Root{Path: root, WatchPath: root, IsDir: true, Recursive: true}); err != nil {
		t.Fatalf("index.Add(%q) error = %v", root, err)
	}

	go w.runBackend()
	go w.runEvents()

	driver.events <- backend.Event{Path: path, Op: backend.OpCreate}

	select {
	case evt := <-events:
		if evt.Path != path || !evt.Op.Has(OpCreate) {
			t.Fatalf("event = %+v, want create on %q", evt, path)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for dispatched event")
	}

	driver.errors <- errors.New("backend failed")
	close(driver.errors)
	close(driver.events)

	select {
	case err := <-errs:
		if err == nil || err.Error() != "backend failed" {
			t.Fatalf("error = %v, want backend failed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for dispatched error")
	}

	select {
	case <-w.done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for watcher shutdown")
	}
}

func TestHandleBackendEventFilteringAndRename(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	oldDir := filepath.Join(root, "old")
	newDir := filepath.Join(root, "new")
	excluded := filepath.Join(root, "skip.tmp")

	w := newUnitWatcher(t, Config{
		ExcludeGlobs: []string{"*.tmp"},
		Ops:          OpCreate | OpRename,
	}, newStubDriver())
	if err := w.index.Add(tree.Root{Path: root, WatchPath: root, IsDir: true, Recursive: true}); err != nil {
		t.Fatalf("index.Add(root) error = %v", err)
	}
	if err := w.index.Add(tree.Root{Path: oldDir, WatchPath: oldDir, IsDir: true, Recursive: true}); err != nil {
		t.Fatalf("index.Add(oldDir) error = %v", err)
	}

	w.handleBackendEvent(backend.Event{
		Path:    newDir,
		OldPath: oldDir,
		Op:      backend.OpRename,
		IsDir:   true,
	})
	evt, ok := w.queue.Pop()
	if !ok {
		t.Fatal("queue.Pop() = closed, want rename event")
	}
	if evt.Path != newDir || evt.OldPath != oldDir || !evt.Op.Has(OpRename) {
		t.Fatalf("rename event = %+v, want rename %q -> %q", evt, oldDir, newDir)
	}
	if !w.index.Matches(filepath.Join(newDir, "child.txt")) {
		t.Fatal("index was not updated after rename")
	}

	w.handleBackendEvent(backend.Event{
		Path: filepath.Join(root, "ignored.txt"),
		Op:   backend.OpWrite,
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = w.queue.Pop()
	}()
	select {
	case <-done:
		t.Fatal("unexpected queued event for unmatched op/path")
	case <-time.After(100 * time.Millisecond):
	}
	w.queue.Close()
	<-done

	w.queue = queue.New[Event](8)
	w.handleBackendEvent(backend.Event{Op: backend.OpOverflow})
	overflow, ok := w.queue.Pop()
	if !ok || overflow.Op != OpOverflow {
		t.Fatalf("overflow event = %+v, %v, want OpOverflow", overflow, ok)
	}

	w.queue = queue.New[Event](8)
	w.handleBackendEvent(backend.Event{
		Path: excluded,
		Op:   backend.OpCreate,
	})
	blocked := make(chan struct{})
	go func() {
		defer close(blocked)
		_, _ = w.queue.Pop()
	}()
	select {
	case <-blocked:
		t.Fatal("unexpected queued event for excluded path")
	case <-time.After(100 * time.Millisecond):
	}
	w.queue.Close()
	<-blocked

	w.queue = queue.New[Event](8)
	w.handleBackendEvent(backend.Event{
		OldPath: excluded,
		Op:      backend.OpRename,
	})
	oldOnly := make(chan struct{})
	go func() {
		defer close(oldOnly)
		_, _ = w.queue.Pop()
	}()
	select {
	case <-oldOnly:
		t.Fatal("unexpected queued event for excluded old path without new path")
	case <-time.After(100 * time.Millisecond):
	}
	w.queue.Close()
	<-oldOnly
}

func TestDispatchEventAndErrorLogging(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	w := newUnitWatcher(t, Config{Logger: logger}, newStubDriver())

	w.dispatchEvent(Event{
		Path:    "/tmp/file.txt",
		OldPath: filepath.Join(string(filepath.Separator), "tmp", "old.txt"),
		Op:      OpRename,
		IsDir:   false,
	})
	if got := buf.String(); !strings.Contains(got, "gaze event") || !strings.Contains(got, "old_path="+filepath.Join(string(filepath.Separator), "tmp", "old.txt")) {
		t.Fatalf("dispatchEvent log = %q, want event log with old_path", got)
	}

	buf.Reset()
	w.cfg.OnError = func(error) {
		panic("error handler boom")
	}
	w.dispatchError(errors.New("boom"))
	if got := buf.String(); !strings.Contains(got, "gaze error handler panic") {
		t.Fatalf("dispatchError log = %q, want panic log", got)
	}
}

func TestEmitErrorAndDispatchEventPanic(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	errs := make(chan error, 2)
	w := newUnitWatcher(t, Config{
		Logger: logger,
		OnError: func(err error) {
			errs <- err
		},
	}, newStubDriver())

	w.emitError(nil)
	select {
	case err := <-errs:
		t.Fatalf("emitError(nil) produced error %v", err)
	default:
	}

	w.cfg.OnError = nil
	w.emitError(errors.New("logged error"))
	if got := buf.String(); !strings.Contains(got, "gaze error") {
		t.Fatalf("emitError logger output = %q, want gaze error log", got)
	}
	buf.Reset()
	w.cfg.OnError = func(err error) {
		errs <- err
	}

	w.dispatchEvent(Event{Path: "/tmp/file.txt", Op: OpCreate})
	if got := buf.String(); !strings.Contains(got, "gaze event") {
		t.Fatalf("dispatchEvent with nil handler wrote %q, want event log", got)
	}

	w.cfg.OnEvent = func(Event) {
		panic("event handler boom")
	}
	w.dispatchEvent(Event{Path: "/tmp/file.txt", Op: OpCreate})
	select {
	case err := <-errs:
		if err == nil || !strings.Contains(err.Error(), "event handler panic") {
			t.Fatalf("panic error = %v, want handler panic error", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for panic error")
	}
}

func TestWatchDirectoryWithConfigAcceptsFilePath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	w, err := WatchDirectoryWithConfig(file, Config{})
	if err != nil {
		t.Fatalf("WatchDirectoryWithConfig() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestWatchFileWithConfigAcceptsDirectoryPath(t *testing.T) {
	dir := t.TempDir()

	w, err := WatchFileWithConfig(dir, Config{})
	if err != nil {
		t.Fatalf("WatchFileWithConfig() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestWatchDirectoryWithConfigFollowsDirectorySymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "link")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink setup unavailable on windows: %v", err)
		}
		t.Fatalf("Symlink() error = %v", err)
	}

	w, err := WatchDirectoryWithConfig(link, Config{FollowSymlinks: true})
	if err != nil {
		t.Fatalf("WatchDirectoryWithConfig() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestWatchFileWithConfigBrokenSymlinkFails(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "broken.txt")
	if err := os.Symlink(filepath.Join(root, "missing.txt"), link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink setup unavailable on windows: %v", err)
		}
		t.Fatalf("Symlink() error = %v", err)
	}

	w, err := WatchFileWithConfig(link, Config{FollowSymlinks: true})
	if err == nil {
		if w != nil {
			_ = w.Close()
		}
		t.Fatal("expected WatchFileWithConfig to reject a broken symlink")
	}
}

func TestNewWithConfigReturnsMatcherError(t *testing.T) {
	_, err := NewWithConfig(Config{ExcludeGlobs: []string{"["}})
	if err == nil {
		t.Fatal("expected invalid glob to fail")
	}
}

func TestNewWithConfigReturnsBackendError(t *testing.T) {
	prev := newBackend
	t.Cleanup(func() { newBackend = prev })

	want := errors.New("backend boom")
	newBackend = func(backend.Config) (backend.Watcher, error) {
		return nil, want
	}

	_, err := NewWithConfig(Config{})
	if !errors.Is(err, want) {
		t.Fatalf("NewWithConfig() error = %v, want %v", err, want)
	}
}

func TestWatchConstructorsReturnNewWatcherErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := WatchDirectoryWithConfig(path, Config{ExcludeGlobs: []string{"["}}); err == nil {
		t.Fatal("expected WatchDirectoryWithConfig to return constructor error")
	}
	if _, err := WatchFileWithConfig(path, Config{ExcludeGlobs: []string{"["}}); err == nil {
		t.Fatal("expected WatchFileWithConfig to return constructor error")
	}
}

func TestWatchDirectoryWithConfigJoinsAddAndCloseErrors(t *testing.T) {
	prev := newBackend
	t.Cleanup(func() { newBackend = prev })

	addErr := errors.New("add boom")
	closeErr := errors.New("close boom")
	newBackend = func(backend.Config) (backend.Watcher, error) {
		driver := newStubDriver()
		driver.addErr = addErr
		driver.closeErr = closeErr
		return driver, nil
	}

	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := WatchDirectoryWithConfig(path, Config{})
	if err == nil || !errors.Is(err, addErr) || !errors.Is(err, closeErr) {
		t.Fatalf("WatchDirectoryWithConfig() error = %v, want joined add+close error", err)
	}
}

func TestWatchFileWithConfigJoinsAddAndCloseErrors(t *testing.T) {
	prev := newBackend
	t.Cleanup(func() { newBackend = prev })

	addErr := errors.New("add boom")
	closeErr := errors.New("close boom")
	newBackend = func(backend.Config) (backend.Watcher, error) {
		driver := newStubDriver()
		driver.addErr = addErr
		driver.closeErr = closeErr
		return driver, nil
	}

	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := WatchFileWithConfig(path, Config{})
	if err == nil || !errors.Is(err, addErr) || !errors.Is(err, closeErr) {
		t.Fatalf("WatchFileWithConfig() error = %v, want joined add+close error", err)
	}
}

func TestPrepareTargetRejectsSymlinkWithoutFollow(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "link")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink setup unavailable on windows: %v", err)
		}
		t.Fatalf("Symlink() error = %v", err)
	}

	matcher, err := filter.New(filter.Config{})
	if err != nil {
		t.Fatalf("filter.New() error = %v", err)
	}

	w := &Watcher{cfg: resolveConfig(Config{}), matcher: matcher}
	_, err = w.prepareTarget(link)
	if err == nil || !strings.Contains(err.Error(), "FollowSymlinks") {
		t.Fatalf("prepareTarget() error = %v, want symlink follow error", err)
	}
}

func TestPrepareTargetRejectsExcludedRoot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "excluded")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	matcher, err := filter.New(filter.Config{
		Exclude: func(candidate string, isDir bool) bool { return candidate == path },
	})
	if err != nil {
		t.Fatalf("filter.New() error = %v", err)
	}

	w := &Watcher{cfg: resolveConfig(Config{}), matcher: matcher}
	_, err = w.prepareTarget(path)
	if err == nil || !strings.Contains(err.Error(), "excluded root") {
		t.Fatalf("prepareTarget() error = %v, want excluded root error", err)
	}
}

func TestNormalizePathRejectsBlankPath(t *testing.T) {
	w := &Watcher{}
	if _, err := w.normalizePath("   "); err == nil {
		t.Fatal("expected normalizePath to reject blank input")
	}
}

func TestNormalizePathCleansRelativePath(t *testing.T) {
	w := &Watcher{}
	got, err := w.normalizePath("./testdata/../.")
	if err != nil {
		t.Fatalf("normalizePath() error = %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("normalizePath() = %q, want absolute path", got)
	}
}

func TestHandleBackendEventBranches(t *testing.T) {
	tmpRoot := filepath.Join(string(filepath.Separator), "tmp")

	matcher, err := filter.New(filter.Config{
		Exclude: func(path string, isDir bool) bool { return strings.Contains(path, "excluded") },
	})
	if err != nil {
		t.Fatalf("filter.New() error = %v", err)
	}

	t.Run("skips unmatched non-overflow", func(t *testing.T) {
		w := &Watcher{
			cfg:     resolveConfig(Config{}),
			matcher: matcher,
			index:   tree.New(),
			queue:   queue.New[Event](1),
		}
		w.handleBackendEvent(backend.Event{Path: filepath.Join(tmpRoot, "other"), Op: backend.OpWrite})
		w.queue.Close()
		if _, ok := w.queue.Pop(); ok {
			t.Fatal("did not expect unmatched event to be queued")
		}
	})

	t.Run("queues overflow without path match", func(t *testing.T) {
		w := &Watcher{
			cfg:     resolveConfig(Config{Ops: OpOverflow}),
			matcher: matcher,
			index:   tree.New(),
			queue:   queue.New[Event](1),
		}
		w.handleBackendEvent(backend.Event{Op: backend.OpOverflow})
		w.queue.Close()
		evt, ok := w.queue.Pop()
		if !ok || evt.Op != OpOverflow {
			t.Fatalf("Pop() = (%v, %t), want overflow event", evt, ok)
		}
	})

	t.Run("skips excluded old path without new path", func(t *testing.T) {
		w := &Watcher{
			cfg:     resolveConfig(Config{}),
			matcher: matcher,
			index:   tree.New(),
			queue:   queue.New[Event](1),
		}
		excludedPath := filepath.Join(tmpRoot, "excluded.txt")
		if err := w.index.Add(tree.Root{Path: excludedPath}); err != nil {
			t.Fatalf("index.Add() error = %v", err)
		}
		w.handleBackendEvent(backend.Event{OldPath: excludedPath, Op: backend.OpRemove})
		w.queue.Close()
		if _, ok := w.queue.Pop(); ok {
			t.Fatal("did not expect excluded old-path event to be queued")
		}
	})
}

func TestNewWatcherExcludeCallbackUsesPathInfo(t *testing.T) {
	prev := newBackend
	t.Cleanup(func() { newBackend = prev })

	driver := newStubDriver()
	var captured PathInfo
	newBackend = func(cfg backend.Config) (backend.Watcher, error) {
		candidate := filepath.Join(string(filepath.Separator), "tmp", "example.txt")
		if !cfg.ShouldExclude(candidate, false) {
			t.Fatal("expected wrapped exclude callback to return true")
		}
		return driver, nil
	}

	w, err := NewWithConfig(Config{
		Exclude: func(info PathInfo) bool {
			captured = info
			return info.Base == "example.txt" && !info.IsDir
		},
	})
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}
	if captured.Path != filepath.Join(string(filepath.Separator), "tmp", "example.txt") || captured.Base != "example.txt" || captured.IsDir {
		t.Fatalf("captured PathInfo = %+v", captured)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestWatcherAddAndRemoveBranches(t *testing.T) {
	matcher, err := filter.New(filter.Config{})
	if err != nil {
		t.Fatalf("filter.New() error = %v", err)
	}

	t.Run("add driver error removes index entry", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file.txt")
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		driver := newStubDriver()
		driver.addErr = errors.New("driver boom")
		w := &Watcher{
			cfg:     resolveConfig(Config{}),
			matcher: matcher,
			index:   tree.New(),
			driver:  driver,
			queue:   queue.New[Event](1),
			done:    make(chan struct{}),
		}

		err := w.Add(path)
		if err == nil || !strings.Contains(err.Error(), "driver boom") {
			t.Fatalf("Add() error = %v, want driver error", err)
		}
		if w.index.Matches(path) {
			t.Fatal("did not expect failed add to remain indexed")
		}
	})

	t.Run("remove missing returns not exist", func(t *testing.T) {
		driver := newStubDriver()
		w := &Watcher{
			cfg:     resolveConfig(Config{}),
			matcher: matcher,
			index:   tree.New(),
			driver:  driver,
			queue:   queue.New[Event](1),
			done:    make(chan struct{}),
		}

		if err := w.Remove("/tmp/missing"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Remove() error = %v, want %v", err, os.ErrNotExist)
		}
	})

	t.Run("remove returns driver error", func(t *testing.T) {
		driver := newStubDriver()
		driver.removeErr = errors.New("remove boom")
		w := &Watcher{
			cfg:     resolveConfig(Config{}),
			matcher: matcher,
			index:   tree.New(),
			driver:  driver,
			queue:   queue.New[Event](1),
			done:    make(chan struct{}),
		}
		path := filepath.Join(t.TempDir(), "file.txt")
		if err := w.index.Add(tree.Root{Path: path}); err != nil {
			t.Fatalf("index.Add() error = %v", err)
		}

		err := w.Remove(path)
		if err == nil || !strings.Contains(err.Error(), "remove boom") {
			t.Fatalf("Remove() error = %v, want driver error", err)
		}
	})
}

func TestPrepareTargetMissingPath(t *testing.T) {
	matcher, err := filter.New(filter.Config{})
	if err != nil {
		t.Fatalf("filter.New() error = %v", err)
	}
	w := &Watcher{cfg: resolveConfig(Config{}), matcher: matcher}
	if _, err := w.prepareTarget(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected prepareTarget to fail for missing path")
	}
}

func TestNewAndWatchWrappers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	filePath := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(filePath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	w, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("New().Close() error = %v", err)
	}

	dirWatcher, err := WatchDirectory(root)
	if err != nil {
		t.Fatalf("WatchDirectory() error = %v", err)
	}
	if err := dirWatcher.Close(); err != nil {
		t.Fatalf("WatchDirectory().Close() error = %v", err)
	}

	fileWatcher, err := WatchFile(filePath)
	if err != nil {
		t.Fatalf("WatchFile() error = %v", err)
	}
	if err := fileWatcher.Close(); err != nil {
		t.Fatalf("WatchFile().Close() error = %v", err)
	}

	if _, err := NewWithConfig(Config{ExcludeGlobs: []string{"["}}); err == nil {
		t.Fatal("NewWithConfig(invalid glob) error = nil, want error")
	}
	if _, err := WatchDirectoryWithConfig(filepath.Join(root, "missing"), Config{}); err == nil {
		t.Fatal("WatchDirectoryWithConfig(missing path) error = nil, want error")
	}
	if _, err := WatchFileWithConfig(filepath.Join(root, "missing.txt"), Config{}); err == nil {
		t.Fatal("WatchFileWithConfig(missing path) error = nil, want error")
	}
}
