//go:build windows

package backend

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const windowsNotifyMask = windows.FILE_NOTIFY_CHANGE_FILE_NAME |
	windows.FILE_NOTIFY_CHANGE_DIR_NAME |
	windows.FILE_NOTIFY_CHANGE_ATTRIBUTES |
	windows.FILE_NOTIFY_CHANGE_SIZE |
	windows.FILE_NOTIFY_CHANGE_LAST_WRITE |
	windows.FILE_NOTIFY_CHANGE_CREATION |
	windows.FILE_NOTIFY_CHANGE_SECURITY

type windowsWatcher struct {
	cfg Config

	mu     sync.Mutex
	closed bool
	roots  map[string]*windowsRoot

	events chan Event
	errors chan error
	done   chan struct{}
	wg     sync.WaitGroup
}

type windowsRoot struct {
	target Target
	handle windows.Handle
	ready  chan struct{}
}

func New(cfg Config) (Watcher, error) {
	return &windowsWatcher{
		cfg:    cfg,
		roots:  make(map[string]*windowsRoot),
		events: make(chan Event, cfg.BufferSize),
		errors: make(chan error, 64),
		done:   make(chan struct{}),
	}, nil
}

func (w *windowsWatcher) Add(target Target) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return os.ErrClosed
	}
	if _, ok := w.roots[target.Path]; ok {
		return nil
	}

	ptr, err := windows.UTF16PtrFromString(target.WatchPath)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(
		ptr,
		windows.FILE_LIST_DIRECTORY,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return fmt.Errorf("gaze: open %q: %w", target.WatchPath, err)
	}

	root := &windowsRoot{
		target: target,
		handle: handle,
		ready:  make(chan struct{}),
	}
	w.roots[target.Path] = root
	w.wg.Add(1)
	go w.runRoot(root)
	<-root.ready
	return nil
}

func (w *windowsWatcher) Remove(path string) error {
	w.mu.Lock()
	root, ok := w.roots[path]
	if ok {
		delete(w.roots, path)
	}
	w.mu.Unlock()
	if !ok {
		return os.ErrNotExist
	}
	return closeWindowsHandle(root.handle)
}

func (w *windowsWatcher) Events() <-chan Event {
	return w.events
}

func (w *windowsWatcher) Errors() <-chan error {
	return w.errors
}

func (w *windowsWatcher) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	roots := make([]*windowsRoot, 0, len(w.roots))
	for _, root := range w.roots {
		roots = append(roots, root)
	}
	w.roots = map[string]*windowsRoot{}
	w.mu.Unlock()

	for _, root := range roots {
		_ = closeWindowsHandle(root.handle)
	}
	w.wg.Wait()
	close(w.events)
	close(w.errors)
	close(w.done)
	return nil
}

func (w *windowsWatcher) runRoot(root *windowsRoot) {
	defer w.wg.Done()
	close(root.ready)

	buf := make([]byte, 64*1024)
	var pendingOld string
	for {
		var bytes uint32
		err := windows.ReadDirectoryChanges(
			root.handle,
			&buf[0],
			uint32(len(buf)),
			root.target.IsDir && root.target.Recursive,
			windowsNotifyMask,
			&bytes,
			nil,
			0,
		)
		if err != nil {
			if w.isClosed() || errors.Is(err, windows.ERROR_INVALID_HANDLE) || errors.Is(err, windows.ERROR_OPERATION_ABORTED) {
				return
			}
			w.emitError(fmt.Errorf("gaze: ReadDirectoryChangesW %q: %w", root.target.WatchPath, err))
			return
		}

		offset := uint32(0)
		for {
			info := (*windows.FileNotifyInformation)(unsafe.Pointer(&buf[offset]))
			nameLen := info.FileNameLength / 2
			nameSlice := unsafe.Slice((*uint16)(unsafe.Pointer(&info.FileName)), nameLen)
			name := string(utf16.Decode(nameSlice))
			fullPath := filepath.Clean(filepath.Join(root.target.WatchPath, filepath.FromSlash(name)))

			switch info.Action {
			case windows.FILE_ACTION_ADDED:
				w.emitEvent(Event{Path: fullPath, Op: OpCreate, IsDir: probeDir(fullPath)})
			case windows.FILE_ACTION_REMOVED:
				w.emitEvent(Event{Path: fullPath, Op: OpRemove, IsDir: false})
			case windows.FILE_ACTION_MODIFIED:
				w.emitEvent(Event{Path: fullPath, Op: OpWrite, IsDir: probeDir(fullPath)})
			case windows.FILE_ACTION_RENAMED_OLD_NAME:
				pendingOld = fullPath
			case windows.FILE_ACTION_RENAMED_NEW_NAME:
				isDir := probeDir(fullPath)
				if pendingOld != "" {
					w.emitEvent(Event{Path: fullPath, OldPath: pendingOld, Op: OpRename, IsDir: isDir})
					pendingOld = ""
				} else {
					w.emitEvent(Event{Path: fullPath, Op: OpCreate, IsDir: isDir})
				}
			}

			if info.NextEntryOffset == 0 {
				break
			}
			offset += info.NextEntryOffset
		}
	}
}

func (w *windowsWatcher) emitEvent(evt Event) {
	w.events <- evt
}

func (w *windowsWatcher) emitError(err error) {
	select {
	case w.errors <- err:
	default:
	}
}

func (w *windowsWatcher) isClosed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}

func probeDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func closeWindowsHandle(handle windows.Handle) error {
	if err := windows.CancelIoEx(handle, nil); err != nil &&
		!errors.Is(err, windows.ERROR_NOT_FOUND) &&
		!errors.Is(err, windows.ERROR_INVALID_HANDLE) {
		_ = windows.CancelIo(handle)
	}
	return windows.CloseHandle(handle)
}
