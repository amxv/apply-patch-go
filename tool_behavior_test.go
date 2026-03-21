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


func TestToolAppliesMultipleOperationsSummary(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "modify.txt"), []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "delete.txt"), []byte("obsolete\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runApplyPatchInDir(t, dir, "*** Begin Patch\n*** Add File: nested/new.txt\n+created\n*** Delete File: delete.txt\n*** Update File: modify.txt\n@@\n-line2\n+changed\n*** End Patch")
	if err != nil {
		t.Fatalf("unexpected error: %v stderr=%q", err, stderr)
	}
	if stdout != "Success. Updated the following files:\nA nested/new.txt\nM modify.txt\nD delete.txt\n" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
}

func TestToolAppliesMultipleChunksSummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3\nline4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runApplyPatchInDir(t, dir, "*** Begin Patch\n*** Update File: multi.txt\n@@\n-line2\n+changed2\n@@\n-line4\n+changed4\n*** End Patch")
	if err != nil {
		t.Fatalf("unexpected error: %v stderr=%q", err, stderr)
	}
	if stdout != "Success. Updated the following files:\nM multi.txt\n" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
}

func TestToolRejectsInvalidHunkHeaderMessage(t *testing.T) {
	dir := t.TempDir()
	_, stderr, err := runApplyPatchInDir(t, dir, "*** Begin Patch\n*** Frobnicate File: foo\n*** End Patch")
	if err == nil {
		t.Fatal("expected error")
	}
	expected := "Invalid patch hunk on line 2: '*** Frobnicate File: foo' is not a valid hunk header. Valid hunk headers: '*** Add File: {path}', '*** Delete File: {path}', '*** Update File: {path}'\n"
	if stderr != expected {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestToolUpdateAppendsTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no_newline.txt")
	if err := os.WriteFile(path, []byte("no newline at end"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runApplyPatchInDir(t, dir, "*** Begin Patch\n*** Update File: no_newline.txt\n@@\n-no newline at end\n+first line\n+second line\n*** End Patch")
	if err != nil {
		t.Fatalf("unexpected error: %v stderr=%q", err, stderr)
	}
	if stdout != "Success. Updated the following files:\nM no_newline.txt\n" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "first line\nsecond line\n" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestToolFailureAfterPartialSuccessLeavesChanges(t *testing.T) {
	dir := t.TempDir()
	created := filepath.Join(dir, "created.txt")
	stdout, stderr, err := runApplyPatchInDir(t, dir, "*** Begin Patch\n*** Add File: created.txt\n+hello\n*** Update File: missing.txt\n@@\n-old\n+new\n*** End Patch")
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout != "" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
	if stderr != "Failed to read file to update missing.txt: No such file or directory (os error 2)\n" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	data, err := os.ReadFile(created)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("unexpected created file: %q", string(data))
	}
}
