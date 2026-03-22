package parse

import "strings"

const (
	BeginPatchMarker         = "*** Begin Patch"
	EndPatchMarker           = "*** End Patch"
	AddFileMarker            = "*** Add File: "
	DeleteFileMarker         = "*** Delete File: "
	UpdateFileMarker         = "*** Update File: "
	MoveToMarker             = "*** Move to: "
	EOFMarker                = "*** End of File"
	ChangeContextMarker      = "@@ "
	EmptyChangeContextMarker = "@@"
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
	return e.Message
}

func ParsePatch(patch string) (*ApplyPatchArgs, error) {
	return parsePatchText(patch, true)
}

func ParsePatchText(patch string, lenient bool) (*ApplyPatchArgs, error) {
	return parsePatchText(patch, lenient)
}

func parsePatchText(patch string, lenient bool) (*ApplyPatchArgs, error) {
	trimmed := strings.TrimSpace(patch)
	lines := SplitPatchLines(trimmed)
	usable, err := CheckPatchBoundariesStrict(lines)
	if err != nil {
		if !lenient {
			return nil, err
		}
		usable, err = CheckPatchBoundariesLenient(lines, err)
		if err != nil {
			return nil, err
		}
	}

	hunks := make([]Hunk, 0)
	if len(usable) >= 2 {
		remaining := usable[1 : len(usable)-1]
		lineNumber := 2
		for len(remaining) > 0 {
			hunk, consumed, err := ParseOneHunk(remaining, lineNumber)
			if err != nil {
				return nil, err
			}
			hunks = append(hunks, hunk)
			lineNumber += consumed
			remaining = remaining[consumed:]
		}
	}

	joined := strings.Join(usable, "\n")
	return &ApplyPatchArgs{Patch: joined, Hunks: hunks}, nil
}

func SplitPatchLines(s string) []string {
	if s == "" {
		return []string{}
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}

func CheckPatchBoundariesStrict(lines []string) ([]string, error) {
	var first, last *string
	switch len(lines) {
	case 0:
		first, last = nil, nil
	case 1:
		first, last = &lines[0], &lines[0]
	default:
		first, last = &lines[0], &lines[len(lines)-1]
	}
	if first != nil && last != nil && strings.TrimSpace(*first) == BeginPatchMarker && strings.TrimSpace(*last) == EndPatchMarker {
		return lines, nil
	}
	if first != nil && strings.TrimSpace(*first) != BeginPatchMarker {
		return nil, invalidPatchError("The first line of the patch must be '*** Begin Patch'")
	}
	return nil, invalidPatchError("The last line of the patch must be '*** End Patch'")
}

func CheckPatchBoundariesLenient(lines []string, original error) ([]string, error) {
	if len(lines) < 4 {
		return nil, original
	}
	first := lines[0]
	last := lines[len(lines)-1]
	if (first == "<<EOF" || first == "<<'EOF'" || first == "<<\"EOF\"") && strings.HasSuffix(last, "EOF") {
		inner := lines[1 : len(lines)-1]
		return CheckPatchBoundariesStrict(inner)
	}
	return nil, original
}

func ParseOneHunk(lines []string, lineNumber int) (Hunk, int, error) {
	firstLine := strings.TrimSpace(lines[0])
	if path, ok := strings.CutPrefix(firstLine, AddFileMarker); ok {
		contents := strings.Builder{}
		parsed := 1
		for _, addLine := range lines[1:] {
			if lineToAdd, ok := strings.CutPrefix(addLine, "+"); ok {
				contents.WriteString(lineToAdd)
				contents.WriteByte('\n')
				parsed++
			} else {
				break
			}
		}
		return Hunk{Kind: HunkAddFile, Path: path, Contents: contents.String()}, parsed, nil
	}
	if path, ok := strings.CutPrefix(firstLine, DeleteFileMarker); ok {
		return Hunk{Kind: HunkDeleteFile, Path: path}, 1, nil
	}
	if path, ok := strings.CutPrefix(firstLine, UpdateFileMarker); ok {
		remaining := lines[1:]
		parsed := 1
		var movePath *string
		if len(remaining) > 0 {
			if m, ok := strings.CutPrefix(remaining[0], MoveToMarker); ok {
				cp := m
				movePath = &cp
				remaining = remaining[1:]
				parsed++
			}
		}
		chunks := make([]UpdateFileChunk, 0)
		for len(remaining) > 0 {
			if strings.TrimSpace(remaining[0]) == "" {
				parsed++
				remaining = remaining[1:]
				continue
			}
			if strings.HasPrefix(remaining[0], "***") {
				break
			}
			chunk, consumed, err := ParseUpdateFileChunk(remaining, lineNumber+parsed, len(chunks) == 0)
			if err != nil {
				return Hunk{}, 0, err
			}
			chunks = append(chunks, chunk)
			parsed += consumed
			remaining = remaining[consumed:]
		}
		if len(chunks) == 0 {
			return Hunk{}, 0, invalidHunkError("Update file hunk for path '"+path+"' is empty", lineNumber)
		}
		return Hunk{Kind: HunkUpdateFile, Path: path, MovePath: movePath, Chunks: chunks}, parsed, nil
	}
	return Hunk{}, 0, invalidHunkError("'"+firstLine+"' is not a valid hunk header. Valid hunk headers: '*** Add File: {path}', '*** Delete File: {path}', '*** Update File: {path}'", lineNumber)
}

func ParseUpdateFileChunk(lines []string, lineNumber int, allowMissingContext bool) (UpdateFileChunk, int, error) {
	if len(lines) == 0 {
		return UpdateFileChunk{}, 0, invalidHunkError("Update hunk does not contain any lines", lineNumber)
	}
	startIndex := 0
	var changeContext *string
	if lines[0] == EmptyChangeContextMarker {
		startIndex = 1
	} else if context, ok := strings.CutPrefix(lines[0], ChangeContextMarker); ok {
		startIndex = 1
		cc := context
		changeContext = &cc
	} else if !allowMissingContext {
		return UpdateFileChunk{}, 0, invalidHunkError("Expected update hunk to start with a @@ context marker, got: '"+lines[0]+"'", lineNumber)
	}
	if startIndex >= len(lines) {
		return UpdateFileChunk{}, 0, invalidHunkError("Update hunk does not contain any lines", lineNumber+1)
	}
	chunk := UpdateFileChunk{ChangeContext: changeContext}
	parsed := 0
	for _, line := range lines[startIndex:] {
		switch {
		case line == EOFMarker:
			if parsed == 0 {
				return UpdateFileChunk{}, 0, invalidHunkError("Update hunk does not contain any lines", lineNumber+1)
			}
			chunk.IsEndOfFile = true
			parsed++
			return chunk, parsed + startIndex, nil
		case line == "":
			chunk.OldLines = append(chunk.OldLines, "")
			chunk.NewLines = append(chunk.NewLines, "")
			parsed++
		case strings.HasPrefix(line, " "):
			v := line[1:]
			chunk.OldLines = append(chunk.OldLines, v)
			chunk.NewLines = append(chunk.NewLines, v)
			parsed++
		case strings.HasPrefix(line, "+"):
			chunk.NewLines = append(chunk.NewLines, line[1:])
			parsed++
		case strings.HasPrefix(line, "-"):
			chunk.OldLines = append(chunk.OldLines, line[1:])
			parsed++
		default:
			if parsed == 0 {
				return UpdateFileChunk{}, 0, invalidHunkError("Unexpected line found in update hunk: '"+line+"'. Every line should start with ' ' (context line), '+' (added line), or '-' (removed line)", lineNumber+1)
			}
			return chunk, parsed + startIndex, nil
		}
	}
	return chunk, parsed + startIndex, nil
}

func invalidPatchError(message string) *ParseError {
	return &ParseError{Kind: ParseErrorInvalidPatch, Message: message}
}

func invalidHunkError(message string, lineNumber int) *ParseError {
	return &ParseError{Kind: ParseErrorInvalidHunk, Message: message, LineNumber: lineNumber}
}
