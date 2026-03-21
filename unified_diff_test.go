package applypatch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnifiedDiffLastLineReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "last.txt")
	if err := os.WriteFile(path, []byte("foo\nbar\nbaz\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n foo\n bar\n-baz\n+BAZ\n*** End Patch"
	parsed, err := ParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	chunks := parsed.Hunks[0].Chunks
	diff, err := UnifiedDiffFromChunks(path, chunks)
	if err != nil {
		t.Fatal(err)
	}
	expectedDiff := "@@ -2,2 +2,2 @@\n bar\n-baz\n+BAZ\n"
	if diff.UnifiedDiff != expectedDiff {
		t.Fatalf("unexpected unified diff:\n%s", diff.UnifiedDiff)
	}
	if diff.Content != "foo\nbar\nBAZ\n" {
		t.Fatalf("unexpected content: %q", diff.Content)
	}
}

func TestUnifiedDiffInsertAtEOF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "insert.txt")
	if err := os.WriteFile(path, []byte("foo\nbar\nbaz\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n+quux\n*** End of File\n*** End Patch"
	parsed, err := ParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	chunks := parsed.Hunks[0].Chunks
	diff, err := UnifiedDiffFromChunks(path, chunks)
	if err != nil {
		t.Fatal(err)
	}
	expectedDiff := "@@ -3 +3,2 @@\n baz\n+quux\n"
	if diff.UnifiedDiff != expectedDiff {
		t.Fatalf("unexpected unified diff:\n%s", diff.UnifiedDiff)
	}
	if diff.Content != "foo\nbar\nbaz\nquux\n" {
		t.Fatalf("unexpected content: %q", diff.Content)
	}
}
