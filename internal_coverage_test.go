package applypatch

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPatchInvalidPatchWritesPrefixedError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := ApplyPatch("*** Begin Patch\n", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.HasPrefix(stderr.String(), "Invalid patch: ") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestApplyPatchInvalidHunkWritesPrefixedError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := ApplyPatch("*** Begin Patch\n*** Frobnicate File: foo\n*** End Patch", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.HasPrefix(stderr.String(), "Invalid patch hunk on line 2: ") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestApplyPatchAddFileToDirectoryFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := ApplyPatch("*** Begin Patch\n*** Add File: "+target+"\n+hello\n*** End Patch", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Failed to write file "+target) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestApplyPatchAddFileParentPathIsFileFails(t *testing.T) {
	dir := t.TempDir()
	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(blocked, "child.txt")
	var stdout, stderr bytes.Buffer
	err := ApplyPatch("*** Begin Patch\n*** Add File: "+target+"\n+hello\n*** End Patch", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Failed to create parent directories for "+target) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestApplyPatchDeleteFileRemovePermissionFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission failure is unreliable as root")
	}
	dir := t.TempDir()
	parent := filepath.Join(dir, "parent")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(parent, "delete.txt")
	if err := os.WriteFile(target, []byte("gone\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(parent, 0o755) }()
	var stdout, stderr bytes.Buffer
	err := ApplyPatch("*** Begin Patch\n*** Delete File: "+target+"\n*** End Patch", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "Failed to delete file "+target+"\n" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestUnifiedDiffFromMissingFileFails(t *testing.T) {
	_, err := UnifiedDiffFromChunks(filepath.Join(t.TempDir(), "missing.txt"), []UpdateFileChunk{{OldLines: []string{"old"}, NewLines: []string{"new"}}})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Failed to read file to update") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestComputeReplacementsChangeContextBranches(t *testing.T) {
	ctx := "line1"
	repls, err := computeReplacements([]string{"line1", "line2"}, "ctx.txt", []UpdateFileChunk{{ChangeContext: &ctx, OldLines: []string{"line2"}, NewLines: []string{"LINE2"}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repls) != 1 || repls[0].Start != 1 || repls[0].OldLen != 1 || len(repls[0].NewLines) != 1 || repls[0].NewLines[0] != "LINE2" {
		t.Fatalf("unexpected replacements: %+v", repls)
	}

	missing := "missing"
	_, err = computeReplacements([]string{"line1", "line2"}, "ctx.txt", []UpdateFileChunk{{ChangeContext: &missing, OldLines: []string{"line2"}, NewLines: []string{"LINE2"}}})
	if err == nil || !strings.Contains(err.Error(), "Failed to find context 'missing' in ctx.txt") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestComputeReplacementsTrailingEmptyPatternRetry(t *testing.T) {
	repls, err := computeReplacements([]string{"line1", "line2"}, "retry.txt", []UpdateFileChunk{{OldLines: []string{"line2", ""}, NewLines: []string{"LINE2", ""}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repls) != 1 || repls[0].Start != 1 || repls[0].OldLen != 1 || len(repls[0].NewLines) != 1 || repls[0].NewLines[0] != "LINE2" {
		t.Fatalf("unexpected replacements: %+v", repls)
	}
}

func TestApplyReplacementsCapsSuffixStart(t *testing.T) {
	updated := applyReplacements([]string{"a"}, []replacement{{Start: 1, OldLen: 2, NewLines: []string{"b"}}})
	if strings.Join(updated, "\n") != "a\nb" {
		t.Fatalf("unexpected updated lines: %#v", updated)
	}
}
