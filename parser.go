package applypatch

import "strings"

const (
	beginPatchMarker         = "*** Begin Patch"
	endPatchMarker           = "*** End Patch"
	addFileMarker            = "*** Add File: "
	deleteFileMarker         = "*** Delete File: "
	updateFileMarker         = "*** Update File: "
	moveToMarker             = "*** Move to: "
	eofMarker                = "*** End of File"
	changeContextMarker      = "@@ "
	emptyChangeContextMarker = "@@"
)

func ParsePatch(patch string) (*ApplyPatchArgs, error) {
	return parsePatchText(patch, true)
}

func parsePatchText(patch string, lenient bool) (*ApplyPatchArgs, error) {
	trimmed := strings.TrimSpace(patch)
	lines := splitPatchLines(trimmed)
	usable, err := checkPatchBoundariesStrict(lines)
	if err != nil {
		if !lenient {
			return nil, err
		}
		usable, err = checkPatchBoundariesLenient(lines, err)
		if err != nil {
			return nil, err
		}
	}

	hunks := make([]Hunk, 0)
	if len(usable) >= 2 {
		remaining := usable[1 : len(usable)-1]
		lineNumber := 2
		for len(remaining) > 0 {
			hunk, consumed, err := parseOneHunk(remaining, lineNumber)
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

func splitPatchLines(s string) []string {
	if s == "" {
		return []string{}
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}

func checkPatchBoundariesStrict(lines []string) ([]string, error) {
	var first, last *string
	switch len(lines) {
	case 0:
		first, last = nil, nil
	case 1:
		first, last = &lines[0], &lines[0]
	default:
		first, last = &lines[0], &lines[len(lines)-1]
	}
	if first != nil && last != nil && strings.TrimSpace(*first) == beginPatchMarker && strings.TrimSpace(*last) == endPatchMarker {
		return lines, nil
	}
	if first != nil && strings.TrimSpace(*first) != beginPatchMarker {
		return nil, invalidPatchError("The first line of the patch must be '*** Begin Patch'")
	}
	return nil, invalidPatchError("The last line of the patch must be '*** End Patch'")
}

func checkPatchBoundariesLenient(lines []string, original error) ([]string, error) {
	if len(lines) < 4 {
		return nil, original
	}
	first := lines[0]
	last := lines[len(lines)-1]
	if (first == "<<EOF" || first == "<<'EOF'" || first == "<<\"EOF\"") && strings.HasSuffix(last, "EOF") {
		inner := lines[1 : len(lines)-1]
		return checkPatchBoundariesStrict(inner)
	}
	return nil, original
}

func parseOneHunk(lines []string, lineNumber int) (Hunk, int, error) {
	firstLine := strings.TrimSpace(lines[0])
	if path, ok := strings.CutPrefix(firstLine, addFileMarker); ok {
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
	if path, ok := strings.CutPrefix(firstLine, deleteFileMarker); ok {
		return Hunk{Kind: HunkDeleteFile, Path: path}, 1, nil
	}
	if path, ok := strings.CutPrefix(firstLine, updateFileMarker); ok {
		remaining := lines[1:]
		parsed := 1
		var movePath *string
		if len(remaining) > 0 {
			if m, ok := strings.CutPrefix(remaining[0], moveToMarker); ok {
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
			chunk, consumed, err := parseUpdateFileChunk(remaining, lineNumber+parsed, len(chunks) == 0)
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

func parseUpdateFileChunk(lines []string, lineNumber int, allowMissingContext bool) (UpdateFileChunk, int, error) {
	if len(lines) == 0 {
		return UpdateFileChunk{}, 0, invalidHunkError("Update hunk does not contain any lines", lineNumber)
	}
	startIndex := 0
	var changeContext *string
	if lines[0] == emptyChangeContextMarker {
		startIndex = 1
	} else if context, ok := strings.CutPrefix(lines[0], changeContextMarker); ok {
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
		case line == eofMarker:
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
