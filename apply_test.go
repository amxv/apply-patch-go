package applypatch

import (
	"bytes"
	"os"
	"path/filepath"
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
