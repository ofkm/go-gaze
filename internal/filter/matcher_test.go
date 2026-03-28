package filter

import "testing"

func TestMatcherShouldExclude(t *testing.T) {
	matcher, err := New(Config{
		Prefixes: []string{"/tmp/cache", "vendor"},
		Globs:    []string{"*.tmp"},
		Exclude: func(path string, isDir bool) bool {
			return isDir && path == "/tmp/skipdir"
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	cases := []struct {
		path    string
		isDir   bool
		exclude bool
	}{
		{path: "/tmp/cache/a.txt", exclude: true},
		{path: "/tmp/project/build.tmp", exclude: true},
		{path: "vendor/pkg/file.go", exclude: true},
		{path: "/tmp/skipdir", isDir: true, exclude: true},
		{path: "/tmp/project/main.go", exclude: false},
	}

	for _, tc := range cases {
		if got := matcher.ShouldExclude(tc.path, tc.isDir); got != tc.exclude {
			t.Fatalf("ShouldExclude(%q, %v) = %v, want %v", tc.path, tc.isDir, got, tc.exclude)
		}
	}
}
