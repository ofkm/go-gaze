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
}
