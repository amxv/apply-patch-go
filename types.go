package applypatch

import (
	"path/filepath"
	"strconv"
)

type HunkKind string

const (
	HunkAddFile    HunkKind = "add_file"
	HunkDeleteFile HunkKind = "delete_file"
	HunkUpdateFile HunkKind = "update_file"
)

type ApplyPatchArgs struct {
	Patch   string
	Hunks   []Hunk
	Workdir *string
}

type Hunk struct {
	Kind     HunkKind
	Path     string
	Contents string
	MovePath *string
	Chunks   []UpdateFileChunk
}

func (h Hunk) ResolvePath(cwd string) string {
	return filepath.Join(cwd, h.Path)
}

type UpdateFileChunk struct {
	ChangeContext *string
	OldLines      []string
	NewLines      []string
	IsEndOfFile   bool
}

type ParseErrorKind string

const (
	ParseErrorInvalidPatch ParseErrorKind = "invalid_patch"
	ParseErrorInvalidHunk  ParseErrorKind = "invalid_hunk"
)

type ParseError struct {
	Kind       ParseErrorKind
	Message    string
	LineNumber int
}

func (e *ParseError) Error() string {
	if e == nil {
		return ""
	}
	if e.Kind == ParseErrorInvalidHunk {
		return "invalid hunk at line " + strconv.Itoa(e.LineNumber) + ", " + e.Message
	}
	return "invalid patch: " + e.Message
}

func invalidPatchError(message string) *ParseError {
	return &ParseError{Kind: ParseErrorInvalidPatch, Message: message}
}

func invalidHunkError(message string, lineNumber int) *ParseError {
	return &ParseError{Kind: ParseErrorInvalidHunk, Message: message, LineNumber: lineNumber}
}

type ApplyPatchError struct {
	ParseError *ParseError
	IOError    *IoError
	Message    string
	ImplicitInvocation bool
}

func (e *ApplyPatchError) Error() string {
	if e == nil {
		return ""
	}
	if e.ParseError != nil {
		return e.ParseError.Error()
	}
	if e.IOError != nil {
		return e.IOError.Error()
	}
	if e.ImplicitInvocation {
		return "patch detected without explicit call to apply_patch. Rerun as [\"apply_patch\", \"<patch>\"]"
	}
	return e.Message
}

type IoError struct {
	Context string
	Source  error
}

func (e *IoError) Error() string {
	if e == nil {
		return ""
	}
	if e.Source == nil {
		return e.Context
	}
	return e.Context + ": " + e.Source.Error()
}

type ApplyPatchFileChangeKind string

const (
	ApplyPatchFileChangeAdd    ApplyPatchFileChangeKind = "add"
	ApplyPatchFileChangeDelete ApplyPatchFileChangeKind = "delete"
	ApplyPatchFileChangeUpdate ApplyPatchFileChangeKind = "update"
)

type ApplyPatchFileChange struct {
	Kind        ApplyPatchFileChangeKind
	Content     string
	UnifiedDiff string
	MovePath    *string
	NewContent  string
}

type ApplyPatchAction struct {
	Changes map[string]ApplyPatchFileChange
	Patch   string
	Cwd     string
}

func (a *ApplyPatchAction) IsEmpty() bool {
	return a == nil || len(a.Changes) == 0
}

type AffectedPaths struct {
	Added    []string
	Modified []string
	Deleted  []string
}

type ApplyPatchFileUpdate struct {
	UnifiedDiff string
	Content     string
}
