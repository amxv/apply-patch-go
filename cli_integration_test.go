package applypatch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildApplyPatchBinary(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "apply_patch")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/apply_patch")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, string(out))
	}
	return bin
}

func TestCLIArgAddAndUpdate(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	file := "cli_test.txt"
	abs := filepath.Join(tmp, file)

	addPatch := "*** Begin Patch\n*** Add File: " + file + "\n+hello\n*** End Patch"
	cmd := exec.Command(bin, addPatch)
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("add command failed: %v\n%s", err, string(out))
	}
	if string(out) != "Success. Updated the following files:\nA cli_test.txt\n" {
		t.Fatalf("unexpected add output: %q", string(out))
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("unexpected add file: %q", string(data))
	}

	updatePatch := "*** Begin Patch\n*** Update File: " + file + "\n@@\n-hello\n+world\n*** End Patch"
	cmd = exec.Command(bin, updatePatch)
	cmd.Dir = tmp
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("update command failed: %v\n%s", err, string(out))
	}
	if string(out) != "Success. Updated the following files:\nM cli_test.txt\n" {
		t.Fatalf("unexpected update output: %q", string(out))
	}
	data, err = os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "world\n" {
		t.Fatalf("unexpected update file: %q", string(data))
	}
}

func TestCLIStdinAddAndUpdate(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	file := "cli_test_stdin.txt"
	abs := filepath.Join(tmp, file)

	addPatch := "*** Begin Patch\n*** Add File: " + file + "\n+hello\n*** End Patch"
	cmd := exec.Command(bin)
	cmd.Dir = tmp
	cmd.Stdin = strings.NewReader(addPatch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stdin add command failed: %v\n%s", err, string(out))
	}
	if string(out) != "Success. Updated the following files:\nA cli_test_stdin.txt\n" {
		t.Fatalf("unexpected stdin add output: %q", string(out))
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("unexpected stdin add file: %q", string(data))
	}

	updatePatch := "*** Begin Patch\n*** Update File: " + file + "\n@@\n-hello\n+world\n*** End Patch"
	cmd = exec.Command(bin)
	cmd.Dir = tmp
	cmd.Stdin = strings.NewReader(updatePatch)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stdin update command failed: %v\n%s", err, string(out))
	}
	if string(out) != "Success. Updated the following files:\nM cli_test_stdin.txt\n" {
		t.Fatalf("unexpected stdin update output: %q", string(out))
	}
	data, err = os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "world\n" {
		t.Fatalf("unexpected stdin update file: %q", string(data))
	}
}

func TestCLIUsageOnEmptyStdin(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	cmd := exec.Command(bin)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 2 {
		t.Fatalf("unexpected exit error: %v", err)
	}
	if string(out) != "Usage: apply_patch 'PATCH'\n       echo 'PATCH' | apply_patch\n" {
		t.Fatalf("unexpected usage output: %q", string(out))
	}
}

func TestCLIRejectsExtraArgs(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	cmd := exec.Command(bin, "one", "two")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 2 {
		t.Fatalf("unexpected exit error: %v", err)
	}
	if string(out) != "Error: apply_patch accepts exactly one argument.\n" {
		t.Fatalf("unexpected extra-args output: %q", string(out))
	}
}

func TestCLIFailingPatchReturnsExitOne(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	cmd := exec.Command(bin, "*** Begin Patch\n*** Delete File: missing.txt\n*** End Patch")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("unexpected exit error: %v", err)
	}
	if string(out) != "Failed to delete file missing.txt\n" {
		t.Fatalf("unexpected failing-patch output: %q", string(out))
	}
}

func TestCLIMultipleOperationsSummary(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "modify.txt"), []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "delete.txt"), []byte("obsolete\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Add File: nested/new.txt\n+created\n*** Delete File: delete.txt\n*** Update File: modify.txt\n@@\n-line2\n+changed\n*** End Patch"
	cmd := exec.Command(bin, patch)
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("multiple ops command failed: %v\n%s", err, string(out))
	}
	if string(out) != "Success. Updated the following files:\nA nested/new.txt\nM modify.txt\nD delete.txt\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func TestCLIRejectsEmptyPatchBody(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	cmd := exec.Command(bin, "*** Begin Patch\n*** End Patch")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("unexpected exit error: %v", err)
	}
	if string(out) != "No files were modified.\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func TestCLIReportsMissingContextMessage(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	path := filepath.Join(tmp, "modify.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "*** Begin Patch\n*** Update File: modify.txt\n@@\n-missing\n+changed\n*** End Patch")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("unexpected exit error: %v", err)
	}
	if string(out) != "Failed to find expected lines in modify.txt:\nmissing\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func TestCLIRequiresExistingFileForUpdateMessage(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	cmd := exec.Command(bin, "*** Begin Patch\n*** Update File: missing.txt\n@@\n-old\n+new\n*** End Patch")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("unexpected exit error: %v", err)
	}
	if string(out) != "Failed to read file to update missing.txt: No such file or directory (os error 2)\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func TestCLIRejectsInvalidHunkHeaderMessage(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	cmd := exec.Command(bin, "*** Begin Patch\n*** Frobnicate File: foo\n*** End Patch")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("unexpected exit error: %v", err)
	}
	expected := "Invalid patch hunk on line 2: '*** Frobnicate File: foo' is not a valid hunk header. Valid hunk headers: '*** Add File: {path}', '*** Delete File: {path}', '*** Update File: {path}'\n"
	if string(out) != expected {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func TestCLIRejectsEmptyUpdateHunkMessage(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	cmd := exec.Command(bin, "*** Begin Patch\n*** Update File: foo.txt\n*** End Patch")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("unexpected exit error: %v", err)
	}
	if string(out) != "Invalid patch hunk on line 2: Update file hunk for path 'foo.txt' is empty\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func TestCLIDeleteDirectoryFailsMessage(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "*** Begin Patch\n*** Delete File: dir\n*** End Patch")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("unexpected exit error: %v", err)
	}
	if string(out) != "Failed to delete file dir\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
}

func TestCLIMovesFileToNewDirectorySummary(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	oldPath := filepath.Join(tmp, "old", "name.txt")
	newPath := filepath.Join(tmp, "renamed", "dir", "name.txt")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldPath, []byte("old content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: old/name.txt\n*** Move to: renamed/dir/name.txt\n@@\n-old content\n+new content\n*** End Patch"
	cmd := exec.Command(bin, patch)
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("move command failed: %v\n%s", err, string(out))
	}
	if string(out) != "Success. Updated the following files:\nM renamed/dir/name.txt\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old path removed, stat err=%v", err)
	}
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new content\n" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestCLIAppliesMultipleChunks(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	path := filepath.Join(tmp, "multi.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3\nline4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: multi.txt\n@@\n-line2\n+changed2\n@@\n-line4\n+changed4\n*** End Patch"
	cmd := exec.Command(bin, patch)
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("multiple chunks command failed: %v\n%s", err, string(out))
	}
	if string(out) != "Success. Updated the following files:\nM multi.txt\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "line1\nchanged2\nline3\nchanged4\n" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestCLIMoveOverwritesExistingDestination(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	oldPath := filepath.Join(tmp, "old", "name.txt")
	dstPath := filepath.Join(tmp, "renamed", "dir", "name.txt")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldPath, []byte("from\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstPath, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: old/name.txt\n*** Move to: renamed/dir/name.txt\n@@\n-from\n+new\n*** End Patch"
	cmd := exec.Command(bin, patch)
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("move overwrite command failed: %v\n%s", err, string(out))
	}
	if string(out) != "Success. Updated the following files:\nM renamed/dir/name.txt\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new\n" {
		t.Fatalf("unexpected destination content: %q", string(data))
	}
}

func TestCLIAddOverwritesExistingFile(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	path := filepath.Join(tmp, "duplicate.txt")
	if err := os.WriteFile(path, []byte("old content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "*** Begin Patch\n*** Add File: duplicate.txt\n+new content\n*** End Patch")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("add overwrite command failed: %v\n%s", err, string(out))
	}
	if string(out) != "Success. Updated the following files:\nA duplicate.txt\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new content\n" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestCLIUpdateAppendsTrailingNewline(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	path := filepath.Join(tmp, "no_newline.txt")
	if err := os.WriteFile(path, []byte("no newline at end"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "*** Begin Patch\n*** Update File: no_newline.txt\n@@\n-no newline at end\n+first line\n+second line\n*** End Patch")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("trailing newline command failed: %v\n%s", err, string(out))
	}
	if string(out) != "Success. Updated the following files:\nM no_newline.txt\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "first line\nsecond line\n" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestCLIFailureAfterPartialSuccessLeavesChanges(t *testing.T) {
	bin := buildApplyPatchBinary(t)
	tmp := t.TempDir()
	created := filepath.Join(tmp, "created.txt")
	patch := "*** Begin Patch\n*** Add File: created.txt\n+hello\n*** Update File: missing.txt\n@@\n-old\n+new\n*** End Patch"
	cmd := exec.Command(bin, patch)
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("unexpected exit error: %v", err)
	}
	if string(out) != "Failed to read file to update missing.txt: No such file or directory (os error 2)\n" {
		t.Fatalf("unexpected output: %q", string(out))
	}
	data, err := os.ReadFile(created)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("unexpected created file content: %q", string(data))
	}
}
