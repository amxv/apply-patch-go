package applypatch

import "testing"

func TestParsePatchBasicAdd(t *testing.T) {
	patch := "*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch"
	parsed, err := ParsePatch(patch)
	if err != nil {
		t.Fatalf("ParsePatch returned error: %v", err)
	}
	if len(parsed.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(parsed.Hunks))
	}
	h := parsed.Hunks[0]
	if h.Kind != HunkAddFile || h.Path != "foo" || h.Contents != "hi\n" {
		t.Fatalf("unexpected hunk: %+v", h)
	}
}

func TestParsePatchLenientHeredoc(t *testing.T) {
	patch := "<<EOF\n*** Begin Patch\n*** Update File: file2.py\n import foo\n+bar\n*** End Patch\nEOF\n"
	parsed, err := ParsePatch(patch)
	if err != nil {
		t.Fatalf("ParsePatch returned error: %v", err)
	}
	if len(parsed.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(parsed.Hunks))
	}
	h := parsed.Hunks[0]
	if h.Kind != HunkUpdateFile || h.Path != "file2.py" || len(h.Chunks) != 1 {
		t.Fatalf("unexpected update hunk: %+v", h)
	}
	chunk := h.Chunks[0]
	if len(chunk.OldLines) != 1 || chunk.OldLines[0] != "import foo" {
		t.Fatalf("unexpected old lines: %#v", chunk.OldLines)
	}
	if len(chunk.NewLines) != 2 || chunk.NewLines[1] != "bar" {
		t.Fatalf("unexpected new lines: %#v", chunk.NewLines)
	}
}

func TestParsePatchWhitespacePaddedMarkers(t *testing.T) {
	patch := " *** Begin Patch \n*** Add File: foo\n+hi\n *** End Patch "
	parsed, err := ParsePatch(patch)
	if err != nil {
		t.Fatalf("ParsePatch returned error: %v", err)
	}
	if len(parsed.Hunks) != 1 || parsed.Hunks[0].Kind != HunkAddFile {
		t.Fatalf("unexpected hunks: %+v", parsed.Hunks)
	}
}

func TestParsePatchUpdateWithoutExplicitFirstContextMarker(t *testing.T) {
	patch := "*** Begin Patch\n*** Update File: file2.py\n import foo\n+bar\n*** End Patch"
	parsed, err := ParsePatch(patch)
	if err != nil {
		t.Fatalf("ParsePatch returned error: %v", err)
	}
	if len(parsed.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(parsed.Hunks))
	}
	h := parsed.Hunks[0]
	if h.Kind != HunkUpdateFile || len(h.Chunks) != 1 {
		t.Fatalf("unexpected hunk: %+v", h)
	}
	chunk := h.Chunks[0]
	if len(chunk.NewLines) != 2 || chunk.NewLines[1] != "bar" {
		t.Fatalf("unexpected chunk: %+v", chunk)
	}
}


func TestParsePatchRejectsBadFirstLine(t *testing.T) {
	_, err := ParsePatch("bad")
	if err == nil {
		t.Fatal("expected error")
	}
	perr, ok := err.(*ParseError)
	if !ok || perr.Kind != ParseErrorInvalidPatch || perr.Message != "The first line of the patch must be '*** Begin Patch'" {
		t.Fatalf("unexpected parse error: %#v", err)
	}
}

func TestParsePatchRejectsMissingEndMarker(t *testing.T) {
	_, err := ParsePatch("*** Begin Patch\nbad")
	if err == nil {
		t.Fatal("expected error")
	}
	perr, ok := err.(*ParseError)
	if !ok || perr.Kind != ParseErrorInvalidPatch || perr.Message != "The last line of the patch must be '*** End Patch'" {
		t.Fatalf("unexpected parse error: %#v", err)
	}
}
