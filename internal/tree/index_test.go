package tree

import "testing"

func TestIndexMatchesAndMovePrefix(t *testing.T) {
	index := New()
	_ = index.Add(Root{Path: "/root", WatchPath: "/root", IsDir: true, Recursive: true})
	_ = index.Add(Root{Path: "/flat", WatchPath: "/flat", IsDir: true, Recursive: false})
	_ = index.Add(Root{Path: "/flat/file.txt", WatchPath: "/flat", IsDir: false})

	if !index.Matches("/root/a/b.txt") {
		t.Fatal("recursive root did not match descendant")
	}
	if !index.Matches("/flat/child.txt") {
		t.Fatal("flat root did not match direct child")
	}
	if index.Matches("/flat/nested/child.txt") {
		t.Fatal("flat root unexpectedly matched nested child")
	}
	if !index.Matches("/flat/file.txt") {
		t.Fatal("file root did not match exact file")
	}

	index.MovePrefix("/root", "/moved")
	if !index.Matches("/moved/a/b.txt") {
		t.Fatal("moved recursive root did not match descendant")
	}
	if index.Matches("/root/a/b.txt") {
		t.Fatal("old recursive root path still matched after move")
	}

	if removed, ok := index.Remove("/flat/file.txt"); !ok || removed.Path != "/flat/file.txt" {
		t.Fatalf("Remove(file root) = (%+v, %v), want removed file root", removed, ok)
	}
	if _, ok := index.Remove("/missing"); ok {
		t.Fatal("Remove(missing) = ok, want false")
	}
}

func TestJoinMovedPath(t *testing.T) {
	if got := joinMovedPath("/old/child/file.txt", "/old", "/new"); got != "/new/child/file.txt" {
		t.Fatalf("joinMovedPath() = %q, want %q", got, "/new/child/file.txt")
	}
}

func TestIndexMovePrefixRewritesNestedAndExactMatches(t *testing.T) {
	idx := New()
	if err := idx.Add(Root{Path: "/tmp/project", WatchPath: "/tmp/project", IsDir: true, Recursive: true}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := idx.Add(Root{Path: "/tmp/project/nested", WatchPath: "/tmp/project/nested", IsDir: true, Recursive: true}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := idx.Add(Root{Path: "/tmp/project-file"}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	idx.MovePrefix("/tmp/project", "/tmp/renamed")

	if !idx.Matches("/tmp/renamed") {
		t.Fatal("expected exact moved root to match")
	}
	if !idx.Matches("/tmp/renamed/nested/child.txt") {
		t.Fatal("expected nested moved directory to match descendants")
	}
	if idx.Matches("/tmp/project/nested/child.txt") {
		t.Fatal("did not expect old nested directory to keep matching")
	}
	if idx.Matches("/tmp/renamed-file") {
		t.Fatal("did not expect sibling path to be rewritten")
	}
	if !idx.Matches("/tmp/project-file") {
		t.Fatal("expected unrelated sibling root to remain")
	}
}

func TestIndexMovePrefixIgnoresUnrelatedPrefix(t *testing.T) {
	idx := New()
	if err := idx.Add(Root{Path: "/workspace/root", Recursive: true}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	idx.MovePrefix("/workspace/other", "/workspace/new")

	if !idx.Matches("/workspace/root") {
		t.Fatal("expected unrelated move to leave existing root intact")
	}
}

func TestJoinMovedPathExactAndNested(t *testing.T) {
	if got := joinMovedPath("/tmp/old", "/tmp/old", "/tmp/new"); got != "/tmp/new" {
		t.Fatalf("joinMovedPath() = %q, want %q", got, "/tmp/new")
	}
	if got := joinMovedPath("/tmp/old/nested/file.txt", "/tmp/old", "/tmp/new"); got != "/tmp/new/nested/file.txt" {
		t.Fatalf("joinMovedPath() = %q, want %q", got, "/tmp/new/nested/file.txt")
	}
}
