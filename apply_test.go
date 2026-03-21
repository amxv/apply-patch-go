package applypatch

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPatchMoveFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + src + "\n*** Move to: " + dst + "\n@@\n-line\n+line2\n*** End Patch"
	var stdout, stderr bytes.Buffer
	if err := ApplyPatch(patch, &stdout, &stderr); err != nil {
		t.Fatalf("ApplyPatch returned error: %v stderr=%q", err, stderr.String())
	}
	if stdout.String() != "Success. Updated the following files:\nM "+dst+"\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected source to be removed, stat err=%v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "line2\n" {
		t.Fatalf("unexpected destination content: %q", string(got))
	}
}

func TestApplyPatchUpdateWithUnicodeDash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unicode.py")
	original := "import asyncio  # local import \u2013 avoids top\u2011level dep\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n-import asyncio  # local import - avoids top-level dep\n+import asyncio  # HELLO\n*** End Patch"
	var stdout, stderr bytes.Buffer
	if err := ApplyPatch(patch, &stdout, &stderr); err != nil {
		t.Fatalf("ApplyPatch returned error: %v stderr=%q", err, stderr.String())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "import asyncio  # HELLO\n" {
		t.Fatalf("unexpected file content: %q", string(got))
	}
}

func TestApplyHunksWritesSummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	patch := "*** Begin Patch\n*** Add File: " + path + "\n+hello\n*** End Patch"
	parsed, err := ParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := ApplyHunks(parsed.Hunks, &stdout, &stderr); err != nil {
		t.Fatalf("ApplyHunks returned error: %v stderr=%q", err, stderr.String())
	}
	if stdout.String() != "Success. Updated the following files:\nA "+path+"\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestApplyHunksErrorWritesStderr(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.txt")
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n-old\n+new\n*** End Patch"
	parsed, err := ParsePatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = ApplyHunks(parsed.Hunks, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	expected := "Failed to read file to update " + path + ": No such file or directory (os error 2)\n"
	if stderr.String() != expected {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestApplyPatchAddFileCreatesContents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "add.txt")
	patch := "*** Begin Patch\n*** Add File: " + path + "\n+ab\n+cd\n*** End Patch"
	var stdout, stderr bytes.Buffer
	if err := ApplyPatch(patch, &stdout, &stderr); err != nil {
		t.Fatalf("ApplyPatch returned error: %v stderr=%q", err, stderr.String())
	}
	if stdout.String() != "Success. Updated the following files:\nA "+path+"\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ab\ncd\n" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestApplyPatchDeleteFileRemovesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "del.txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Delete File: " + path + "\n*** End Patch"
	var stdout, stderr bytes.Buffer
	if err := ApplyPatch(patch, &stdout, &stderr); err != nil {
		t.Fatalf("ApplyPatch returned error: %v stderr=%q", err, stderr.String())
	}
	if stdout.String() != "Success. Updated the following files:\nD "+path+"\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, stat err=%v", err)
	}
}

func TestApplyPatchUpdateFileModifiesContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "update.txt")
	if err := os.WriteFile(path, []byte("foo\nbar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n foo\n-bar\n+baz\n*** End Patch"
	var stdout, stderr bytes.Buffer
	if err := ApplyPatch(patch, &stdout, &stderr); err != nil {
		t.Fatalf("ApplyPatch returned error: %v stderr=%q", err, stderr.String())
	}
	if stdout.String() != "Success. Updated the following files:\nM "+path+"\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "foo\nbaz\n" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestApplyPatchInterleavedChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "interleaved.txt")
	if err := os.WriteFile(path, []byte("a\nb\nc\nd\ne\nf\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n a\n-b\n+B\n@@\n c\n d\n-e\n+E\n@@\n f\n+g\n*** End of File\n*** End Patch"
	var stdout, stderr bytes.Buffer
	if err := ApplyPatch(patch, &stdout, &stderr); err != nil {
		t.Fatalf("ApplyPatch returned error: %v stderr=%q", err, stderr.String())
	}
	if stdout.String() != "Success. Updated the following files:\nM "+path+"\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "a\nB\nc\nd\nE\nf\ng\n" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestApplyPatchPureAdditionChunkFollowedByRemoval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "panic.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n+after-context\n+second-line\n@@\n line1\n-line2\n-line3\n+line2-replacement\n*** End Patch"
	var stdout, stderr bytes.Buffer
	if err := ApplyPatch(patch, &stdout, &stderr); err != nil {
		t.Fatalf("ApplyPatch returned error: %v stderr=%q", err, stderr.String())
	}
	if stdout.String() != "Success. Updated the following files:\nM "+path+"\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "line1\nline2-replacement\nafter-context\nsecond-line\n" {
		t.Fatalf("unexpected file content: %q", string(data))
	}
}

func TestApplyPatchFailsOnWriteError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readonly.txt")
	if err := os.WriteFile(path, []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chmod(path, 0o644)
	}()
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n-before\n+after\n*** End Patch"
	var stdout, stderr bytes.Buffer
	err := ApplyPatch(patch, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() == "" {
		t.Fatal("expected stderr output")
	}
}

func TestApplyPatchMoveFailsRemovingOriginal(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission failure is unreliable as root")
	}
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	dstDir := filepath.Join(dir, "dst")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(srcDir, "move.txt")
	dst := filepath.Join(dstDir, "move.txt")
	if err := os.WriteFile(src, []byte("from\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(srcDir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(srcDir, 0o755) }()
	patch := "*** Begin Patch\n*** Update File: " + src + "\n*** Move to: " + dst + "\n@@\n-from\n+to\n*** End Patch"
	var stdout, stderr bytes.Buffer
	err := ApplyPatch(patch, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Failed to remove original "+src) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestApplyPatchUpdateDirectoryReadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + path + "\n@@\n-old\n+new\n*** End Patch"
	var stdout, stderr bytes.Buffer
	err := ApplyPatch(patch, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Failed to read file to update "+path) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestApplyPatchMoveWriteDestinationDirectoryError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst")
	if err := os.WriteFile(src, []byte("from\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	patch := "*** Begin Patch\n*** Update File: " + src + "\n*** Move to: " + dst + "\n@@\n-from\n+to\n*** End Patch"
	var stdout, stderr bytes.Buffer
	err := ApplyPatch(patch, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Failed to write file "+dst) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}
