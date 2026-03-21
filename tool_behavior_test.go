package applypatch

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func runApplyPatchInDir(t *testing.T, dir string, patch string) (string, string, error) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = ApplyPatch(patch, &stdout, &stderr)
	return stdout.String(), stderr.String(), err
}

func TestToolRejectsEmptyPatch(t *testing.T) {
	dir := t.TempDir()
	_, stderr, err := runApplyPatchInDir(t, dir, "*** Begin Patch\n*** End Patch")
	if err == nil {
		t.Fatal("expected error")
	}
	if stderr != "No files were modified.\n" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestToolReportsMissingContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "modify.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := runApplyPatchInDir(t, dir, "*** Begin Patch\n*** Update File: modify.txt\n@@\n-missing\n+changed\n*** End Patch")
	if err == nil {
		t.Fatal("expected error")
	}
	if stderr != "Failed to find expected lines in modify.txt:\nmissing\n" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestToolRejectsMissingFileDeleteMessage(t *testing.T) {
	dir := t.TempDir()
	_, stderr, err := runApplyPatchInDir(t, dir, "*** Begin Patch\n*** Delete File: missing.txt\n*** End Patch")
	if err == nil {
		t.Fatal("expected error")
	}
	if stderr != "Failed to delete file missing.txt\n" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestToolRejectsDeleteDirectoryMessage(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := runApplyPatchInDir(t, dir, "*** Begin Patch\n*** Delete File: dir\n*** End Patch")
	if err == nil {
		t.Fatal("expected error")
	}
	if stderr != "Failed to delete file dir\n" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestToolRequiresExistingFileForUpdateMessage(t *testing.T) {
	dir := t.TempDir()
	_, stderr, err := runApplyPatchInDir(t, dir, "*** Begin Patch\n*** Update File: missing.txt\n@@\n-old\n+new\n*** End Patch")
	if err == nil {
		t.Fatal("expected error")
	}
	if stderr != "Failed to read file to update missing.txt: No such file or directory (os error 2)\n" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}
