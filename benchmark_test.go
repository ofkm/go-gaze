package gaze_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"go.ofkm.dev/gaze"
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
