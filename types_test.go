package applypatch

import (
	"errors"
	"strings"
	"testing"
)

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

type failAfterNWriter struct {
	remaining int
}

func (w *failAfterNWriter) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, errors.New("write failed")
	}
	w.remaining--
	return len(p), nil
}

func TestParseErrorFormatting(t *testing.T) {
	var nilErr *ParseError
	if nilErr.Error() != "" {
		t.Fatalf("unexpected nil parse error string: %q", nilErr.Error())
	}
	if got := invalidPatchError("bad patch").Error(); got != "invalid patch: bad patch" {
		t.Fatalf("unexpected patch error string: %q", got)
	}
	if got := invalidHunkError("bad hunk", 7).Error(); got != "invalid hunk at line 7, bad hunk" {
		t.Fatalf("unexpected hunk error string: %q", got)
	}
}

func TestApplyPatchErrorFormatting(t *testing.T) {
	var nilErr *ApplyPatchError
	if nilErr.Error() != "" {
		t.Fatalf("unexpected nil apply error string: %q", nilErr.Error())
	}
	if got := (&ApplyPatchError{ParseError: invalidPatchError("bad patch")}).Error(); got != "invalid patch: bad patch" {
		t.Fatalf("unexpected parse-backed error string: %q", got)
	}
	if got := (&ApplyPatchError{IOError: &IoError{Context: "context", Source: errors.New("boom")}}).Error(); got != "context: boom" {
		t.Fatalf("unexpected io-backed error string: %q", got)
	}
	if got := (&ApplyPatchError{Message: "plain"}).Error(); got != "plain" {
		t.Fatalf("unexpected plain error string: %q", got)
	}
	if got := (&ApplyPatchError{ImplicitInvocation: true}).Error(); got != "patch detected without explicit call to apply_patch. Rerun as [\"apply_patch\", \"<patch>\"]" {
		t.Fatalf("unexpected implicit error string: %q", got)
	}
}

func TestIoErrorFormatting(t *testing.T) {
	var nilErr *IoError
	if nilErr.Error() != "" {
		t.Fatalf("unexpected nil io error string: %q", nilErr.Error())
	}
	if got := (&IoError{Context: "context only"}).Error(); got != "context only" {
		t.Fatalf("unexpected context-only io error string: %q", got)
	}
	if got := (&IoError{Context: "context", Source: errors.New("boom")}).Error(); got != "context: boom" {
		t.Fatalf("unexpected wrapped io error string: %q", got)
	}
}

func TestApplyPatchActionNilHelpers(t *testing.T) {
	var action *ApplyPatchAction
	if !action.IsEmpty() {
		t.Fatal("expected nil action to be empty")
	}
	if action.Changes() != nil {
		t.Fatalf("expected nil changes map, got %#v", action.Changes())
	}
}

func TestPrintSummaryPropagatesWriterError(t *testing.T) {
	err := PrintSummary(&AffectedPaths{Added: []string{"a.txt"}}, failWriter{})
	if err == nil || err.Error() != "write failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrintSummaryModifiedAndDeletedBranches(t *testing.T) {
	writer := &failAfterNWriter{remaining: 2}
	err := PrintSummary(&AffectedPaths{Modified: []string{"m.txt"}, Deleted: []string{"d.txt"}}, writer)
	if err == nil || err.Error() != "write failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrintSummaryAllEntries(t *testing.T) {
	var out strings.Builder
	err := PrintSummary(&AffectedPaths{Added: []string{"a.txt"}, Modified: []string{"m.txt"}, Deleted: []string{"d.txt"}}, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Success. Updated the following files:\nA a.txt\nM m.txt\nD d.txt\n"
	if out.String() != expected {
		t.Fatalf("unexpected summary: %q", out.String())
	}
}

func TestPrintSummaryModifiedBranchError(t *testing.T) {
	writer := &failAfterNWriter{remaining: 1}
	err := PrintSummary(&AffectedPaths{Modified: []string{"m.txt"}}, writer)
	if err == nil || err.Error() != "write failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}
