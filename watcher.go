package gaze

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.ofkm.dev/gaze/internal/backend"
	"go.ofkm.dev/gaze/internal/filter"
	"go.ofkm.dev/gaze/internal/queue"
	"go.ofkm.dev/gaze/internal/tree"
)

var (
	newMatcher = filter.New
	newBackend = backend.New
)

type Watcher struct {
	cfg     Config
	matcher *filter.Matcher
	index   *tree.Index
	driver  backend.Watcher
	queue   *queue.Queue[Event]
	logger  *slog.Logger

	closeOnce sync.Once
	closeErr  error
	done      chan struct{}
}

func WatchDirectory(path string) (*Watcher, error) {
	return WatchDirectoryWithConfig(path, Config{})
}

func WatchDirectoryWithConfig(path string, cfg Config) (*Watcher, error) {
	w, err := NewWithConfig(cfg)
	if err != nil {
		return nil, err
	}
	if err := w.Add(path); err != nil {
		if closeErr := w.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, err
	}
	return w, nil
}

func WatchFile(path string) (*Watcher, error) {
	return WatchFileWithConfig(path, Config{})
}

func WatchFileWithConfig(path string, cfg Config) (*Watcher, error) {
	cfg = resolveConfig(cfg)
	cfg.Recursion = RecursionDisabled

	w, err := newWatcher(cfg)
	if err != nil {
		return nil, err
	}
	if err := w.Add(path); err != nil {
		if closeErr := w.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, err
	}
	return w, nil
}

func New() (*Watcher, error) {
	return NewWithConfig(Config{})
}

func NewWithConfig(cfg Config) (*Watcher, error) {
	return newWatcher(resolveConfig(cfg))
}

func newWatcher(cfg Config) (*Watcher, error) {
	var exclude func(string, bool) bool
	if cfg.Exclude != nil {
		exclude = func(path string, isDir bool) bool {
			return cfg.Exclude(PathInfo{
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
		queue:   queue.New[Event](cfg.QueueCapacity),
		logger:  cfg.Logger,
		done:    make(chan struct{}),
	}

	go w.runBackend()
	go w.runEvents()

	return w, nil
}

func (w *Watcher) Add(path string) error {
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
		return err
	}

	return nil
}

func (w *Watcher) Remove(path string) error {
	normalized, err := w.normalizePath(path)
	if err != nil {
		return err
	}

	root, ok := w.index.Remove(normalized)
	if !ok {
		return os.ErrNotExist
	}

	return w.driver.Remove(root.Path)
}

func (w *Watcher) Close() error {
	w.closeOnce.Do(func() {
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
	publicOp := Op(evt.Op)

	if evt.Op.Has(backend.OpRename) && evt.Path != "" && evt.OldPath != "" {
		w.index.MovePrefix(evt.OldPath, evt.Path)
	}

	if !evt.Op.Has(backend.OpOverflow) && evt.Path != "" {
		if !w.index.Matches(evt.Path) && (evt.OldPath == "" || !w.index.Matches(evt.OldPath)) {
			return
		}
	}

	if publicOp != OpOverflow && !w.cfg.Ops.Has(publicOp) {
		return
	}

	public := Event{
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

func (w *Watcher) dispatchEvent(evt Event) {
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
		Recursive: isDir && w.cfg.recursiveEnabled(true),
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
