package filter

import (
	"path/filepath"
	"testing"
)

func TestMatcherShouldExclude(t *testing.T) {
	cacheDir := filepath.Join(string(filepath.Separator), "tmp", "cache")
	skipDir := filepath.Join(string(filepath.Separator), "tmp", "skipdir")

	matcher, err := New(Config{
		Prefixes: []string{cacheDir, "vendor"},
		Globs:    []string{"*.tmp"},
		Exclude: func(path string, isDir bool) bool {
			return isDir && path == skipDir
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
		{path: filepath.Join(cacheDir, "a.txt"), exclude: true},
		{path: filepath.Join(string(filepath.Separator), "tmp", "project", "build.tmp"), exclude: true},
		{path: filepath.Join("vendor", "pkg", "file.go"), exclude: true},
		{path: skipDir, isDir: true, exclude: true},
		{path: filepath.Join(string(filepath.Separator), "tmp", "project", "main.go"), exclude: false},
	}

	for _, tc := range cases {
		if got := matcher.ShouldExclude(tc.path, tc.isDir); got != tc.exclude {
			t.Fatalf("ShouldExclude(%q, %v) = %v, want %v", tc.path, tc.isDir, got, tc.exclude)
		}
	}
}

func TestMatcherNewTrimsAndRejectsInvalidGlob(t *testing.T) {
	if _, err := New(Config{Globs: []string{"["}}); err == nil {
		t.Fatal("New(invalid glob) error = nil, want error")
	}

	cacheDir := filepath.Join(string(filepath.Separator), "tmp", "cache")
	matcher, err := New(Config{
		Prefixes: []string{"", " " + filepath.Join(cacheDir, "..", "cache") + " "},
		Globs:    []string{"", "*.log"},
	})
	if err != nil {
		t.Fatalf("New(trimmed config) error = %v", err)
	}
	if !matcher.ShouldExclude(filepath.Join(cacheDir, "app.log"), false) {
		t.Fatal("ShouldExclude() = false, want true for cleaned prefix/glob")
	}
}

func TestMatcherShouldExcludeDirectoryPrefixAndGlobVariants(t *testing.T) {
	m, err := New(Config{
		Prefixes: []string{"vendor", ".git"},
		Globs:    []string{"*.tmp", "*.cache"},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name  string
		path  string
		isDir bool
		want  bool
	}{
		{name: "prefix dir", path: "vendor", isDir: true, want: true},
		{name: "prefix child", path: filepath.Join("vendor", "pkg", "file.go"), want: true},
		{name: "hidden dir prefix child", path: filepath.Join(".git", "objects", "01"), want: true},
		{name: "glob file", path: filepath.Join("build", "output.tmp"), want: true},
		{name: "glob simple", path: "data.cache", want: true},
		{name: "keep source", path: filepath.Join("internal", "filter", "matcher.go"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := m.ShouldExclude(tc.path, tc.isDir); got != tc.want {
				t.Fatalf("ShouldExclude(%q, %t) = %t, want %t", tc.path, tc.isDir, got, tc.want)
			}
		})
	}
}

func TestMatcherShouldExcludeExcludeCallback(t *testing.T) {
	m, err := New(Config{
		Exclude: func(path string, isDir bool) bool {
			return isDir && path == "generated"
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if !m.ShouldExclude("generated", true) {
		t.Fatal("expected callback exclusion for generated directory")
	}
	if m.ShouldExclude(filepath.Join("generated", "file.go"), false) {
		t.Fatal("did not expect callback exclusion for file path")
	}
}

func TestMatcherNewTrimsBlankEntriesAndRejectsInvalidGlob(t *testing.T) {
	m, err := New(Config{
		Prefixes: []string{"", "  ", "vendor"},
		Globs:    []string{"", " *.tmp ", "[", "*.log"},
	})
	if err == nil {
		t.Fatalf("New() = %#v, want invalid glob error", m)
	}

	m, err = New(Config{
		Prefixes: []string{"", "vendor"},
		Globs:    []string{"", "*.log"},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if !m.ShouldExclude(filepath.Join("nested", "file.log"), false) {
		t.Fatal("expected slash-form glob to match full path")
	}
}

func TestMatcherShouldExcludeSlashedGlob(t *testing.T) {
	m, err := New(Config{Globs: []string{"nested/*.log"}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if !m.ShouldExclude(filepath.Join("nested", "file.log"), false) {
		t.Fatal("expected slashed glob to match cleaned path")
	}
}
