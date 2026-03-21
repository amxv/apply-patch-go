package applypatch

import (
	"os"
	"path/filepath"
	"strings"
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

func TestUnifiedDiffWithCustomContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ctx.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\nfour\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n-two\n+TWO\n*** End Patch"
	parsed, err := ParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := UnifiedDiffFromChunksWithContext(path, parsed.Hunks[0].Chunks, 0)
	if err != nil {
		t.Fatal(err)
	}
	if diff.UnifiedDiff == "" || !strings.Contains(diff.UnifiedDiff, "@@") {
		t.Fatalf("unexpected unified diff: %q", diff.UnifiedDiff)
	}
}

func TestUnifiedDiffWithContextZeroExact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ctx0.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\nfour\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n-two\n+TWO\n*** End Patch"
	parsed, err := ParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := UnifiedDiffFromChunksWithContext(path, parsed.Hunks[0].Chunks, 0)
	if err != nil {
		t.Fatal(err)
	}
	expected := "@@ -2 +2 @@\n-two\n+TWO\n"
	if diff.UnifiedDiff != expected {
		t.Fatalf("unexpected unified diff:\n%s", diff.UnifiedDiff)
	}
}

func TestUnifiedDiffFirstLineReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "first.txt")
	if err := os.WriteFile(path, []byte("foo\nbar\nbaz\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n-foo\n+FOO\n*** End Patch"
	parsed, err := ParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := UnifiedDiffFromChunks(path, parsed.Hunks[0].Chunks)
	if err != nil {
		t.Fatal(err)
	}
	expected := "@@ -1,2 +1,2 @@\n-foo\n+FOO\n bar\n"
	if diff.UnifiedDiff != expected {
		t.Fatalf("unexpected unified diff:\n%s", diff.UnifiedDiff)
	}
}

func TestUnifiedDiffInterleavedChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "interleaved.txt")
	if err := os.WriteFile(path, []byte("a\nb\nc\nd\ne\nf\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n a\n-b\n+B\n@@\n d\n-e\n+E\n@@\n f\n+g\n*** End of File\n*** End Patch"
	parsed, err := ParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := UnifiedDiffFromChunks(path, parsed.Hunks[0].Chunks)
	if err != nil {
		t.Fatal(err)
	}
	expected := "@@ -1,6 +1,7 @@\n a\n-b\n+B\n c\n d\n-e\n+E\n f\n+g\n"
	if diff.UnifiedDiff != expected {
		t.Fatalf("unexpected unified diff:\n%s", diff.UnifiedDiff)
	}
	if diff.Content != "a\nB\nc\nd\nE\nf\ng\n" {
		t.Fatalf("unexpected content: %q", diff.Content)
	}
}

func TestUnifiedDiffFailsWhenDiffMissingFromPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing-diff.txt")
	if err := os.WriteFile(path, []byte("foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", ""); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Setenv("PATH", oldPath) }()
	_, err := UnifiedDiffFromChunks(path, []UpdateFileChunk{{OldLines: []string{"foo"}, NewLines: []string{"bar"}}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Failed to execute diff") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnifiedDiffFailsCreatingTempDirectory(t *testing.T) {
	tmpRoot := t.TempDir()
	blocked := filepath.Join(tmpRoot, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTmp := os.Getenv("TMPDIR")
	if err := os.Setenv("TMPDIR", blocked); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Setenv("TMPDIR", oldTmp) }()
	path := filepath.Join(t.TempDir(), "tmpdir-fail.txt")
	if err := os.WriteFile(path, []byte("foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := UnifiedDiffFromChunks(path, []UpdateFileChunk{{OldLines: []string{"foo"}, NewLines: []string{"bar"}}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Failed to create temp directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnifiedDiffReturnsErrorForDiffExitCodeGreaterThanOne(t *testing.T) {
	binDir := t.TempDir()
	diffPath := filepath.Join(binDir, "diff")
	if err := os.WriteFile(diffPath, []byte("#!/bin/sh\necho boom\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", binDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Setenv("PATH", oldPath) }()
	path := filepath.Join(t.TempDir(), "diff-exit2.txt")
	if err := os.WriteFile(path, []byte("foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := UnifiedDiffFromChunks(path, []UpdateFileChunk{{OldLines: []string{"foo"}, NewLines: []string{"bar"}}})
	if err == nil || err.Error() != "boom\n" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnifiedDiffUsesStderrWhenDiffExitsOne(t *testing.T) {
	binDir := t.TempDir()
	diffPath := filepath.Join(binDir, "diff")
	script := "#!/bin/sh\necho '--- old' 1>&2\necho '+++ new' 1>&2\necho '@@ -1 +1 @@' 1>&2\necho '-foo' 1>&2\necho '+bar' 1>&2\nexit 1\n"
	if err := os.WriteFile(diffPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", binDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Setenv("PATH", oldPath) }()
	path := filepath.Join(t.TempDir(), "diff-exit1.txt")
	if err := os.WriteFile(path, []byte("foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diff, err := UnifiedDiffFromChunks(path, []UpdateFileChunk{{OldLines: []string{"foo"}, NewLines: []string{"bar"}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff.UnifiedDiff != "@@ -1 +1 @@\n-foo\n+bar\n" {
		t.Fatalf("unexpected unified diff: %q", diff.UnifiedDiff)
	}
}
