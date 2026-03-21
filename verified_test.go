package applypatch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaybeParseApplyPatchVerifiedDeleteReadsContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "del.txt")
	if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := []string{"apply_patch", "*** Begin Patch\n*** Delete File: del.txt\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, dir)
	if got.Kind != MaybeApplyPatchVerifiedBody || got.Action == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	change, ok := got.Action.Changes[path]
	if !ok {
		t.Fatalf("missing change for %s", path)
	}
	if change.Kind != ApplyPatchFileChangeDelete || change.Content != "x\n" {
		t.Fatalf("unexpected change: %+v", change)
	}
}

func TestMaybeParseApplyPatchVerifiedUpdateComputesNewContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "u.txt")
	if err := os.WriteFile(path, []byte("foo\nbar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := []string{"apply_patch", "*** Begin Patch\n*** Update File: u.txt\n@@\n foo\n-bar\n+baz\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, dir)
	if got.Kind != MaybeApplyPatchVerifiedBody || got.Action == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	change, ok := got.Action.Changes[path]
	if !ok {
		t.Fatalf("missing change for %s", path)
	}
	if change.Kind != ApplyPatchFileChangeUpdate || change.NewContent != "foo\nbaz\n" {
		t.Fatalf("unexpected change: %+v", change)
	}
}
