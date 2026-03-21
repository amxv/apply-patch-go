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


func TestParseOneHunkInvalidHeader(t *testing.T) {
	_, _, err := parseOneHunk([]string{"bad"}, 234)
	if err == nil {
		t.Fatal("expected error")
	}
	perr, ok := err.(*ParseError)
	if !ok || perr.Kind != ParseErrorInvalidHunk || perr.LineNumber != 234 {
		t.Fatalf("unexpected parse error: %#v", err)
	}
}

func TestParseUpdateFileChunkErrorsAndShapes(t *testing.T) {
	_, _, err := parseUpdateFileChunk([]string{"bad"}, 123, false)
	if err == nil {
		t.Fatal("expected error")
	}
	perr, ok := err.(*ParseError)
	if !ok || perr.Kind != ParseErrorInvalidHunk || perr.LineNumber != 123 || perr.Message != "Expected update hunk to start with a @@ context marker, got: 'bad'" {
		t.Fatalf("unexpected parse error: %#v", err)
	}

	_, _, err = parseUpdateFileChunk([]string{"@@"}, 123, false)
	if err == nil {
		t.Fatal("expected error")
	}
	perr, ok = err.(*ParseError)
	if !ok || perr.Kind != ParseErrorInvalidHunk || perr.LineNumber != 124 || perr.Message != "Update hunk does not contain any lines" {
		t.Fatalf("unexpected parse error: %#v", err)
	}

	_, _, err = parseUpdateFileChunk([]string{"@@", "bad"}, 123, false)
	if err == nil {
		t.Fatal("expected error")
	}
	perr, ok = err.(*ParseError)
	if !ok || perr.Kind != ParseErrorInvalidHunk || perr.LineNumber != 124 || perr.Message != "Unexpected line found in update hunk: 'bad'. Every line should start with ' ' (context line), '+' (added line), or '-' (removed line)" {
		t.Fatalf("unexpected parse error: %#v", err)
	}

	_, _, err = parseUpdateFileChunk([]string{"@@", "*** End of File"}, 123, false)
	if err == nil {
		t.Fatal("expected error")
	}
	perr, ok = err.(*ParseError)
	if !ok || perr.Kind != ParseErrorInvalidHunk || perr.LineNumber != 124 || perr.Message != "Update hunk does not contain any lines" {
		t.Fatalf("unexpected parse error: %#v", err)
	}

	chunk, consumed, err := parseUpdateFileChunk([]string{"@@ change_context", "", " context", "-remove", "+add", " context2", "*** End Patch"}, 123, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if consumed != 6 {
		t.Fatalf("unexpected consumed lines: %d", consumed)
	}
	if chunk.ChangeContext == nil || *chunk.ChangeContext != "change_context" || chunk.IsEndOfFile {
		t.Fatalf("unexpected chunk metadata: %+v", chunk)
	}
	if len(chunk.OldLines) != 4 || len(chunk.NewLines) != 4 {
		t.Fatalf("unexpected chunk lines: %+v", chunk)
	}
	if chunk.OldLines[0] != "" || chunk.OldLines[1] != "context" || chunk.OldLines[2] != "remove" || chunk.OldLines[3] != "context2" {
		t.Fatalf("unexpected old lines: %#v", chunk.OldLines)
	}
	if chunk.NewLines[0] != "" || chunk.NewLines[1] != "context" || chunk.NewLines[2] != "add" || chunk.NewLines[3] != "context2" {
		t.Fatalf("unexpected new lines: %#v", chunk.NewLines)
	}

	chunk, consumed, err = parseUpdateFileChunk([]string{"@@", "+line", "*** End of File"}, 123, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if consumed != 3 || !chunk.IsEndOfFile || len(chunk.NewLines) != 1 || chunk.NewLines[0] != "line" || len(chunk.OldLines) != 0 || chunk.ChangeContext != nil {
		t.Fatalf("unexpected eof chunk: %+v consumed=%d", chunk, consumed)
	}
}
