package backend

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestWalkPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	keepDir := filepath.Join(root, "keep")
	skipDir := filepath.Join(root, "skip")
	if err := os.Mkdir(keepDir, 0o755); err != nil {
		t.Fatalf("Mkdir(keep) error = %v", err)
	}
	if err := os.Mkdir(skipDir, 0o755); err != nil {
		t.Fatalf("Mkdir(skip) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(keepDir, "file.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile(keep) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skipDir, "ignored.txt"), []byte("no"), 0o644); err != nil {
		t.Fatalf("WriteFile(skip) error = %v", err)
	}

	var recursive []string
	err := walkPath(root, true, func(path string, isDir bool) bool {
		return isDir && filepath.Base(path) == "skip"
	}, func(path string, d fs.DirEntry) error {
		recursive = append(recursive, filepath.Base(path))
		return nil
	})
	if err != nil {
		t.Fatalf("walkPath(recursive) error = %v", err)
	}
	if !slices.Contains(recursive, "keep") || slices.Contains(recursive, "ignored.txt") {
		t.Fatalf("recursive walk = %v, want keep and no ignored.txt", recursive)
	}

	var nonRecursive []string
	err = walkPath(root, false, nil, func(path string, d fs.DirEntry) error {
		nonRecursive = append(nonRecursive, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walkPath(non-recursive) error = %v", err)
	}
	if len(nonRecursive) != 1 || nonRecursive[0] != root {
		t.Fatalf("non-recursive walk = %v, want [%q]", nonRecursive, root)
	}

	wantErr := errors.New("stop")
	err = walkPath(root, true, nil, func(string, fs.DirEntry) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("walkPath(callback error) = %v, want %v", err, wantErr)
	}
}

func TestHasPathPrefix(t *testing.T) {
	t.Parallel()

	root := filepath.Join(string(filepath.Separator), "tmp", "project")
	descendant := filepath.Join(root, "file.txt")
	other := filepath.Join(string(filepath.Separator), "tmp", "project-other")

	if !hasPathPrefix(descendant, root) {
		t.Fatal("hasPathPrefix(descendant) = false, want true")
	}
	if !hasPathPrefix(root, root) {
		t.Fatal("hasPathPrefix(equal) = false, want true")
	}
	if hasPathPrefix(other, root) {
		t.Fatal("hasPathPrefix(non-descendant) = true, want false")
	}
}

func mustMkdirHelperTest(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir(%q) error = %v", path, err)
	}
}

func mustWriteFileHelperTest(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func TestWalkPathNonRecursiveMissingPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if err := walkPath(missing, false, nil, func(string, fs.DirEntry) error { return nil }); err == nil {
		t.Fatal("expected walkPath to fail for a missing root")
	}
}

func TestWalkPathRecursiveHonorsExclude(t *testing.T) {
	root := t.TempDir()
	keepDir := filepath.Join(root, "keep")
	skipDir := filepath.Join(root, "skipdir")
	mustMkdirHelperTest(t, keepDir)
	mustMkdirHelperTest(t, skipDir)
	mustWriteFileHelperTest(t, filepath.Join(root, "keep.txt"))
	mustWriteFileHelperTest(t, filepath.Join(root, "skip.txt"))
	mustWriteFileHelperTest(t, filepath.Join(skipDir, "nested.txt"))

	var seen []string
	err := walkPath(root, true, func(path string, isDir bool) bool {
		base := filepath.Base(path)
		return base == "skipdir" || base == "skip.txt"
	}, func(path string, d fs.DirEntry) error {
		seen = append(seen, filepath.Base(path))
		return nil
	})
	if err != nil {
		t.Fatalf("walkPath() error = %v", err)
	}

	if !slices.Contains(seen, filepath.Base(root)) {
		t.Fatal("expected root to be visited")
	}
	if !slices.Contains(seen, "keep") || !slices.Contains(seen, "keep.txt") {
		t.Fatalf("expected keep entries to be visited, seen=%v", seen)
	}
	if slices.Contains(seen, "skip.txt") || slices.Contains(seen, "skipdir") || slices.Contains(seen, "nested.txt") {
		t.Fatalf("expected excluded entries to be skipped, seen=%v", seen)
	}
}

func TestWalkPathRecursivePropagatesWalkError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based walk failures are not reliable on windows")
	}

	root := t.TempDir()
	restricted := filepath.Join(root, "restricted")
	mustMkdirHelperTest(t, restricted)
	if err := os.Chmod(restricted, 0); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(restricted, 0o755) })

	err := walkPath(root, true, nil, func(string, fs.DirEntry) error { return nil })
	if err == nil {
		t.Fatal("expected walkPath to propagate walk error")
	}
}
