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
