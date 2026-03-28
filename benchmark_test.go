package gaze_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"go.ofkm.dev/gaze"
	"go.ofkm.dev/gaze/internal/filter"
	"go.ofkm.dev/gaze/internal/tree"
)

func BenchmarkWatchDirectoryCreateRemove(b *testing.B) {
	root := b.TempDir()
	cfg := gaze.Config{
		OnEvent: func(gaze.Event) {},
		OnError: func(error) {},
	}

	w, err := gaze.WatchDirectoryWithConfig(root, cfg)
	if err != nil {
		b.Fatalf("WatchDirectory() error = %v", err)
	}
	defer func() {
		_ = w.Close()
	}()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(root, "bench-"+strconv.Itoa(i))
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			b.Fatalf("WriteFile() error = %v", err)
		}
		if err := os.Remove(path); err != nil {
			b.Fatalf("Remove() error = %v", err)
		}
	}
}

func BenchmarkOpString(b *testing.B) {
	op := gaze.OpCreate | gaze.OpWrite | gaze.OpRename | gaze.OpOverflow

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = op.String()
	}
}

func BenchmarkFilterShouldExclude(b *testing.B) {
	matcher, err := filter.New(filter.Config{
		Prefixes: []string{"/tmp/cache", "/tmp/node_modules"},
		Globs:    []string{"*.tmp", "*.swp", ".DS_Store"},
	})
	if err != nil {
		b.Fatalf("filter.New() error = %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = matcher.ShouldExclude("/tmp/project/src/file.go", false)
		_ = matcher.ShouldExclude("/tmp/cache/build.tmp", false)
	}
}

func BenchmarkTreeMatches(b *testing.B) {
	index := tree.New()
	_ = index.Add(tree.Root{Path: "/workspace", WatchPath: "/workspace", IsDir: true, Recursive: true})
	_ = index.Add(tree.Root{Path: "/workspace/config.yaml", WatchPath: "/workspace", IsDir: false})

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = index.Matches("/workspace/src/main.go")
		_ = index.Matches("/workspace/config.yaml")
	}
}
