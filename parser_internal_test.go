package applypatch

import "testing"

func TestSplitPatchLinesEmptyAndCRLF(t *testing.T) {
	if got := splitPatchLines(""); len(got) != 0 {
		t.Fatalf("unexpected split for empty string: %#v", got)
	}
	got := splitPatchLines("a\r\nb\r")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "" {
		t.Fatalf("unexpected split: %#v", got)
	}
}

func TestCheckPatchBoundariesStrictBranches(t *testing.T) {
	if _, err := checkPatchBoundariesStrict([]string{}); err == nil || err.Error() != "invalid patch: The last line of the patch must be '*** End Patch'" {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := checkPatchBoundariesStrict([]string{"oops"}); err == nil || err.Error() != "invalid patch: The first line of the patch must be '*** Begin Patch'" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseOneHunkAndChunkInternalBranches(t *testing.T) {
	hunk, consumed, err := parseOneHunk([]string{"*** Delete File: foo.txt"}, 2)
	if err != nil || consumed != 1 || hunk.Kind != HunkDeleteFile || hunk.Path != "foo.txt" {
		t.Fatalf("unexpected delete hunk: %+v consumed=%d err=%v", hunk, consumed, err)
	}

	hunk, consumed, err = parseOneHunk([]string{"*** Add File: foo.txt", "+hello", "*** End Patch"}, 2)
	if err != nil || consumed != 2 || hunk.Kind != HunkAddFile || hunk.Contents != "hello\n" {
		t.Fatalf("unexpected add hunk: %+v consumed=%d err=%v", hunk, consumed, err)
	}

	hunk, consumed, err = parseOneHunk([]string{"*** Update File: foo.txt", "*** Move to: bar.txt", "", "@@", "-old", "+new", "*** End Patch"}, 2)
	if err != nil || consumed != 6 || hunk.Kind != HunkUpdateFile || hunk.MovePath == nil || *hunk.MovePath != "bar.txt" {
		t.Fatalf("unexpected update hunk: %+v consumed=%d err=%v", hunk, consumed, err)
	}

	if _, _, err := parseOneHunk([]string{"*** Nope File: foo.txt"}, 2); err == nil {
		t.Fatal("expected invalid header error")
	}

	if _, _, err := parseUpdateFileChunk([]string{}, 5, true); err == nil {
		t.Fatal("expected empty chunk error")
	}
	if _, _, err := parseUpdateFileChunk([]string{"bad"}, 5, false); err == nil {
		t.Fatal("expected missing context marker error")
	}
	if _, _, err := parseUpdateFileChunk([]string{"@@", "*** End of File"}, 5, true); err == nil {
		t.Fatal("expected eof-only error")
	}
}

func TestParseOneHunkUpdateEmptyDirect(t *testing.T) {
	if _, _, err := parseOneHunk([]string{"*** Update File: foo.txt"}, 2); err == nil {
		t.Fatal("expected empty update hunk error")
	}
}

func TestParseOneHunkAddFileWithoutContents(t *testing.T) {
	hunk, consumed, err := parseOneHunk([]string{"*** Add File: empty.txt", "*** End Patch"}, 2)
	if err != nil || consumed != 1 || hunk.Kind != HunkAddFile || hunk.Path != "empty.txt" || hunk.Contents != "" {
		t.Fatalf("unexpected add hunk: %+v consumed=%d err=%v", hunk, consumed, err)
	}
}
