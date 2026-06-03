package gaze

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"go.ofkm.dev/gaze/internal/backend"
	"go.ofkm.dev/gaze/internal/filter"
	"go.ofkm.dev/gaze/internal/queue"
	"go.ofkm.dev/gaze/internal/tree"
	"go.ofkm.dev/gaze/types"
)

var (
	newMatcher = filter.New
	newBackend = backend.New
)

// Watcher observes one or more filesystem roots and delivers normalized
// [types.Event] values to the callbacks configured in its [types.Config].
//
// A Watcher owns its goroutines and must be released with [Watcher.Close]. Its
// methods are safe to call from multiple goroutines. Event and error callbacks
// run on goroutines owned by the Watcher, not the caller's (see [types.Config]).
//
// The zero value is not usable; obtain a Watcher from [New], [WatchDirectory],
// or [WatchFile].
type Watcher struct {
	cfg     types.Config
	matcher *filter.Matcher
	index   *tree.Index
	driver  backend.Watcher
	queue   *queue.Queue[types.Event]
	logger  *slog.Logger

	closeOnce sync.Once
	closeErr  error
	closed    atomic.Bool
	done      chan struct{}
}

// New creates an empty Watcher with no roots; call [Watcher.Add] to enroll
// paths. With no argument it uses default configuration; pass a single
// [types.Config] to customize. Passing more than one Config is an error.
//
// For the common single-root case, prefer [WatchDirectory] or [WatchFile].
func New(cfg ...types.Config) (*Watcher, error) {
	c, err := pickConfig(cfg)
	if err != nil {
		return nil, err
	}
	return newWatcher(resolveConfig(c))
}

// WatchDirectory creates a Watcher and starts watching path, normally a
// directory. Directories are watched recursively by default; set
// Config.Recursion to [types.RecursionDisabled] to watch only the top level.
// With no argument it uses defaults; pass a single [types.Config] to customize.
// Passing more than one Config is an error.
//
// path is normalized to an absolute path. It may also name a file, in which
// case it behaves like [WatchFile], though WatchFile states that intent more
// clearly.
func WatchDirectory(path string, cfg ...types.Config) (*Watcher, error) {
	w, err := New(cfg...)
	if err != nil {
		return nil, err
	}
	return watchRoot(w, path)
}

// WatchFile creates a Watcher and starts watching the single file at path.
//
// Gaze does not watch the file inode directly: it watches the file's parent
// directory non-recursively and delivers only events for path. Recursion is
// forced off regardless of Config. With no argument it uses defaults; pass a
// single [types.Config] to customize. Passing more than one Config is an error.
func WatchFile(path string, cfg ...types.Config) (*Watcher, error) {
	c, err := pickConfig(cfg)
	if err != nil {
		return nil, err
	}
	c = resolveConfig(c)
	c.Recursion = types.RecursionDisabled

	w, err := newWatcher(c)
	if err != nil {
		return nil, err
	}
	return watchRoot(w, path)
}

// pickConfig returns the single optional Config, or the zero Config when none
// is supplied. More than one Config is rejected so a mistaken extra argument is
// never silently dropped.
func pickConfig(cfg []types.Config) (types.Config, error) {
	switch len(cfg) {
	case 0:
		return types.Config{}, nil
	case 1:
		return cfg[0], nil
	default:
		return types.Config{}, errors.New("gaze: at most one Config may be provided")
	}
}

// watchRoot adds path to w, closing w if the add fails so the caller never
// receives a live watcher it cannot use.
func watchRoot(w *Watcher, path string) (*Watcher, error) {
	if err := w.Add(path); err != nil {
		if closeErr := w.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, err
	}
	return w, nil
}

func newWatcher(cfg types.Config) (*Watcher, error) {
	var exclude func(string, bool) bool
	if cfg.Exclude != nil {
		exclude = func(path string, isDir bool) bool {
			return cfg.Exclude(types.PathInfo{
				Path:  path,
				Base:  filepath.Base(path),
				IsDir: isDir,
			})
		}
	}

	matcher, err := newMatcher(filter.Config{
		Prefixes: cfg.ExcludePrefixes,
		Globs:    cfg.ExcludeGlobs,
		Exclude:  exclude,
	})
	if err != nil {
		return nil, err
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	driver, err := newBackend(backend.Config{
		BufferSize:     max(cfg.QueueCapacity, 64),
		FollowSymlinks: cfg.FollowSymlinks,
		ShouldExclude:  matcher.ShouldExclude,
	})
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		cfg:     cfg,
		matcher: matcher,
		index:   tree.New(),
		driver:  driver,
		queue:   queue.New[types.Event](cfg.QueueCapacity),
		logger:  cfg.Logger,
		done:    make(chan struct{}),
	}

	go w.runBackend()
	go w.runEvents()

	return w, nil
}

// Add starts watching the file or directory at path, which is normalized to an
// absolute path. Directories are watched according to Config.Recursion; a file
// is watched through its parent directory (see [WatchFile]).
//
// Add returns [ErrWatcherClosed] if the Watcher is closed, an error if path
// cannot be stat'd, an error if path is a symlink and Config.FollowSymlinks is
// false, and an error if path is excluded by the configured filters.
func (w *Watcher) Add(path string) error {
	if w.closed.Load() {
		return ErrWatcherClosed
	}

	target, err := w.prepareTarget(path)
	if err != nil {
		return err
	}

	if err := w.index.Add(tree.Root{
		Path:      target.Path,
		WatchPath: target.WatchPath,
		IsDir:     target.IsDir,
		Recursive: target.Recursive,
	}); err != nil {
		return err
	}

	if err := w.driver.Add(backend.Target{
		Path:      target.Path,
		WatchPath: target.WatchPath,
		IsDir:     target.IsDir,
		Recursive: target.Recursive,
	}); err != nil {
		_, _ = w.index.Remove(target.Path)
		if errors.Is(err, os.ErrClosed) {
			return ErrWatcherClosed
		}
		return err
	}

	return nil
}

// Remove stops watching a root previously passed to [Watcher.Add] or a
// constructor. path is normalized the same way as Add but must name an exact
// root: Remove does not match descendants of a watched directory.
//
// Remove returns [ErrWatcherClosed] if the Watcher is closed and os.ErrNotExist
// if path is not a current root.
func (w *Watcher) Remove(path string) error {
	if w.closed.Load() {
		return ErrWatcherClosed
	}

	normalized, err := w.normalizePath(path)
	if err != nil {
		return err
	}

	root, ok := w.index.Remove(normalized)
	if !ok {
		return os.ErrNotExist
	}

	if err := w.driver.Remove(root.Path); err != nil {
		if errors.Is(err, os.ErrClosed) {
			return ErrWatcherClosed
		}
		return err
	}
	return nil
}

// Close stops all watching, releases backend resources, and waits for in-flight
// event dispatch to drain. It is idempotent and safe to call multiple times and
// from multiple goroutines; every call returns the same error.
func (w *Watcher) Close() error {
	w.closeOnce.Do(func() {
		w.closed.Store(true)
		w.closeErr = w.driver.Close()
		w.queue.Close()
		<-w.done
	})
	return w.closeErr
}

func (w *Watcher) runBackend() {
	defer w.queue.Close()
	for {
		select {
		case evt, ok := <-w.driver.Events():
			if !ok {
				return
			}
			w.handleBackendEvent(evt)
		case err, ok := <-w.driver.Errors():
			if !ok {
				return
			}
			w.emitError(err)
		}
	}
}

func (w *Watcher) runEvents() {
	defer func() {
		close(w.done)
	}()

	for {
		evt, ok := w.queue.Pop()
		if !ok {
			return
		}
		w.dispatchEvent(evt)
	}
}

func (w *Watcher) handleBackendEvent(evt backend.Event) {
	publicOp := types.Op(evt.Op)

	if evt.Op.Has(backend.OpRename) && evt.Path != "" && evt.OldPath != "" {
		w.index.MovePrefix(evt.OldPath, evt.Path)
	}

	if !evt.Op.Has(backend.OpOverflow) && evt.Path != "" {
		if !w.index.Matches(evt.Path) && (evt.OldPath == "" || !w.index.Matches(evt.OldPath)) {
			return
		}
	}

	if publicOp != types.OpOverflow && !w.cfg.Ops.Has(publicOp) {
		return
	}

	public := types.Event{
		Path:    evt.Path,
		OldPath: evt.OldPath,
		Op:      publicOp,
		IsDir:   evt.IsDir,
	}

	if public.Path != "" && w.matcher.ShouldExclude(public.Path, public.IsDir) {
		return
	}
	if public.OldPath != "" && w.matcher.ShouldExclude(public.OldPath, public.IsDir) && public.Path == "" {
		return
	}

	w.queue.Push(public)
}

func (w *Watcher) emitError(err error) {
	if err == nil {
		return
	}
	if w.cfg.OnError != nil {
		w.dispatchError(err)
		return
	}
	if w.logger != nil {
		w.logger.Error("gaze error", "err", err)
	}
}

func (w *Watcher) dispatchEvent(evt types.Event) {
	defer func() {
		if recovered := recover(); recovered != nil {
			w.emitError(fmt.Errorf("gaze: event handler panic: %v", recovered))
		}
	}()
	if w.cfg.OnEvent == nil {
		if w.logger != nil {
			attrs := []any{"op", evt.Op.String(), "path", evt.Path, "is_dir", evt.IsDir}
			if evt.OldPath != "" {
				attrs = append(attrs, "old_path", evt.OldPath)
			}
			w.logger.Info("gaze event", attrs...)
		}
		return
	}
	w.cfg.OnEvent(evt)
}

func (w *Watcher) dispatchError(err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if w.logger != nil {
				w.logger.Error("gaze error handler panic", "panic", recovered)
			}
		}
	}()
	w.cfg.OnError(err)
}

type preparedTarget struct {
	Path      string
	WatchPath string
	IsDir     bool
	Recursive bool
}

func (w *Watcher) prepareTarget(path string) (preparedTarget, error) {
	normalized, err := w.normalizePath(path)
	if err != nil {
		return preparedTarget{}, err
	}

	info, err := os.Lstat(normalized)
	if err != nil {
		return preparedTarget{}, err
	}

	if info.Mode()&os.ModeSymlink != 0 && !w.cfg.FollowSymlinks {
		return preparedTarget{}, fmt.Errorf("gaze: symlink root %q requires Config.FollowSymlinks = true", normalized)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		normalized, err = filepath.EvalSymlinks(normalized)
		if err != nil {
			return preparedTarget{}, err
		}
		info, err = os.Stat(normalized)
		if err != nil {
			return preparedTarget{}, err
		}
	}

	isDir := info.IsDir()
	if w.matcher.ShouldExclude(normalized, isDir) {
		return preparedTarget{}, fmt.Errorf("gaze: excluded root %q", normalized)
	}

	target := preparedTarget{
		Path:      normalized,
		WatchPath: normalized,
		IsDir:     isDir,
		Recursive: isDir && recursiveEnabled(w.cfg, true),
	}
	if !isDir {
		target.WatchPath = filepath.Dir(normalized)
		target.Recursive = false
	}
	return target, nil
}

func (w *Watcher) normalizePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("gaze: empty path")
	}

	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	return abs, nil
}
