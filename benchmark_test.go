package filewatch_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	gofilewatch "go.ofkm.dev/filewatch"
)

func BenchmarkWatchDirectoryCreateRemove(b *testing.B) {
	root := b.TempDir()
	w, err := gofilewatch.WatchDirectory(
		root,
		func(cfg *gofilewatch.Config) {
			cfg.Logger = nil
			cfg.OnEvent = func(gofilewatch.Event) {}
			cfg.OnError = func(error) {}
		},
	)
	if err != nil {
		b.Fatalf("WatchDirectory() error = %v", err)
	}
	defer func() {
		_ = w.Close()
	}()

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
