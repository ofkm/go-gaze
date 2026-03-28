package tree

import (
	"path/filepath"
	"testing"
)

func TestIndexMatchesAndMovePrefix(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "root")
	flat := filepath.Join(string(filepath.Separator), "flat")
	file := filepath.Join(flat, "file.txt")
	moved := filepath.Join(string(filepath.Separator), "moved")

	index := New()
	_ = index.Add(Root{Path: root, WatchPath: root, IsDir: true, Recursive: true})
	_ = index.Add(Root{Path: flat, WatchPath: flat, IsDir: true, Recursive: false})
	_ = index.Add(Root{Path: file, WatchPath: flat, IsDir: false})

	if !index.Matches(filepath.Join(root, "a", "b.txt")) {
		t.Fatal("recursive root did not match descendant")
	}
	if !index.Matches(filepath.Join(flat, "child.txt")) {
		t.Fatal("flat root did not match direct child")
	}
	if index.Matches(filepath.Join(flat, "nested", "child.txt")) {
		t.Fatal("flat root unexpectedly matched nested child")
	}
	if !index.Matches(file) {
		t.Fatal("file root did not match exact file")
	}

	index.MovePrefix(root, moved)
	if !index.Matches(filepath.Join(moved, "a", "b.txt")) {
		t.Fatal("moved recursive root did not match descendant")
	}
	if index.Matches(filepath.Join(root, "a", "b.txt")) {
		t.Fatal("old recursive root path still matched after move")
	}

	if removed, ok := index.Remove(file); !ok || removed.Path != file {
		t.Fatalf("Remove(file root) = (%+v, %v), want removed file root", removed, ok)
	}
	if _, ok := index.Remove(filepath.Join(string(filepath.Separator), "missing")); ok {
		t.Fatal("Remove(missing) = ok, want false")
	}
}

func TestJoinMovedPath(t *testing.T) {
	oldRoot := filepath.Join(string(filepath.Separator), "old")
	newRoot := filepath.Join(string(filepath.Separator), "new")
	want := filepath.Join(newRoot, "child", "file.txt")
	if got := joinMovedPath(filepath.Join(oldRoot, "child", "file.txt"), oldRoot, newRoot); got != want {
		t.Fatalf("joinMovedPath() = %q, want %q", got, want)
	}
}

func TestIndexMovePrefixRewritesNestedAndExactMatches(t *testing.T) {
	project := filepath.Join(string(filepath.Separator), "tmp", "project")
	projectNested := filepath.Join(project, "nested")
	projectFile := filepath.Join(string(filepath.Separator), "tmp", "project-file")
	renamed := filepath.Join(string(filepath.Separator), "tmp", "renamed")

	idx := New()
	if err := idx.Add(Root{Path: project, WatchPath: project, IsDir: true, Recursive: true}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := idx.Add(Root{Path: projectNested, WatchPath: projectNested, IsDir: true, Recursive: true}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := idx.Add(Root{Path: projectFile}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	idx.MovePrefix(project, renamed)

	if !idx.Matches(renamed) {
		t.Fatal("expected exact moved root to match")
	}
	if !idx.Matches(filepath.Join(renamed, "nested", "child.txt")) {
		t.Fatal("expected nested moved directory to match descendants")
	}
	if idx.Matches(filepath.Join(project, "nested", "child.txt")) {
		t.Fatal("did not expect old nested directory to keep matching")
	}
	if idx.Matches(filepath.Join(string(filepath.Separator), "tmp", "renamed-file")) {
		t.Fatal("did not expect sibling path to be rewritten")
	}
	if !idx.Matches(projectFile) {
		t.Fatal("expected unrelated sibling root to remain")
	}
}

func TestIndexMovePrefixIgnoresUnrelatedPrefix(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "workspace", "root")

	idx := New()
	if err := idx.Add(Root{Path: root, Recursive: true}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	idx.MovePrefix(filepath.Join(string(filepath.Separator), "workspace", "other"), filepath.Join(string(filepath.Separator), "workspace", "new"))

	if !idx.Matches(root) {
		t.Fatal("expected unrelated move to leave existing root intact")
	}
}

func TestJoinMovedPathExactAndNested(t *testing.T) {
	oldRoot := filepath.Join(string(filepath.Separator), "tmp", "old")
	newRoot := filepath.Join(string(filepath.Separator), "tmp", "new")
	if got := joinMovedPath(oldRoot, oldRoot, newRoot); got != newRoot {
		t.Fatalf("joinMovedPath() = %q, want %q", got, newRoot)
	}
	want := filepath.Join(newRoot, "nested", "file.txt")
	if got := joinMovedPath(filepath.Join(oldRoot, "nested", "file.txt"), oldRoot, newRoot); got != want {
		t.Fatalf("joinMovedPath() = %q, want %q", got, want)
	}
}
