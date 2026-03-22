package applypatch

import iparse "github.com/amxv/apply-patch-go/internal/parse"

const (
	beginPatchMarker         = iparse.BeginPatchMarker
	endPatchMarker           = iparse.EndPatchMarker
	addFileMarker            = iparse.AddFileMarker
	deleteFileMarker         = iparse.DeleteFileMarker
	updateFileMarker         = iparse.UpdateFileMarker
	moveToMarker             = iparse.MoveToMarker
	eofMarker                = iparse.EOFMarker
	changeContextMarker      = iparse.ChangeContextMarker
	emptyChangeContextMarker = iparse.EmptyChangeContextMarker
)

func ParsePatch(patch string) (*ApplyPatchArgs, error) {
	args, err := iparse.ParsePatch(patch)
	if err != nil {
		return nil, convertParseError(err.(*iparse.ParseError))
	}
	return convertArgsFromInternal(args), nil
}

func parsePatchText(patch string, lenient bool) (*ApplyPatchArgs, error) {
	args, err := iparse.ParsePatchText(patch, lenient)
	if err != nil {
		return nil, convertParseError(err.(*iparse.ParseError))
	}
	return convertArgsFromInternal(args), nil
}

func splitPatchLines(s string) []string {
	return iparse.SplitPatchLines(s)
}

func checkPatchBoundariesStrict(lines []string) ([]string, error) {
	usable, err := iparse.CheckPatchBoundariesStrict(lines)
	if err != nil {
		return nil, convertParseError(err.(*iparse.ParseError))
	}
	return usable, nil
}

func checkPatchBoundariesLenient(lines []string, original error) ([]string, error) {
	var source error = original
	if perr, ok := original.(*ParseError); ok {
		source = &iparse.ParseError{Kind: iparse.ParseErrorKind(perr.Kind), Message: perr.Message, LineNumber: perr.LineNumber}
	}
	usable, err := iparse.CheckPatchBoundariesLenient(lines, source)
	if err != nil {
		return nil, convertParseError(err.(*iparse.ParseError))
	}
	return usable, nil
}

func parseOneHunk(lines []string, lineNumber int) (Hunk, int, error) {
	hunk, consumed, err := iparse.ParseOneHunk(lines, lineNumber)
	if err != nil {
		return Hunk{}, 0, convertParseError(err.(*iparse.ParseError))
	}
	return convertHunkFromInternal(hunk), consumed, nil
}

func parseUpdateFileChunk(lines []string, lineNumber int, allowMissingContext bool) (UpdateFileChunk, int, error) {
	chunk, consumed, err := iparse.ParseUpdateFileChunk(lines, lineNumber, allowMissingContext)
	if err != nil {
		return UpdateFileChunk{}, 0, convertParseError(err.(*iparse.ParseError))
	}
	return convertChunkFromInternal(chunk), consumed, nil
}

func convertArgsFromInternal(args *iparse.ApplyPatchArgs) *ApplyPatchArgs {
	if args == nil {
		return nil
	}
	hunks := make([]Hunk, len(args.Hunks))
	for i, hunk := range args.Hunks {
		hunks[i] = convertHunkFromInternal(hunk)
	}
	return &ApplyPatchArgs{Patch: args.Patch, Hunks: hunks, Workdir: args.Workdir}
}

func convertHunkFromInternal(hunk iparse.Hunk) Hunk {
	chunks := make([]UpdateFileChunk, len(hunk.Chunks))
	for i, chunk := range hunk.Chunks {
		chunks[i] = convertChunkFromInternal(chunk)
	}
	return Hunk{Kind: HunkKind(hunk.Kind), Path: hunk.Path, Contents: hunk.Contents, MovePath: hunk.MovePath, Chunks: chunks}
}

func convertChunkFromInternal(chunk iparse.UpdateFileChunk) UpdateFileChunk {
	return UpdateFileChunk{
		ChangeContext: chunk.ChangeContext,
		OldLines:      append([]string(nil), chunk.OldLines...),
		NewLines:      append([]string(nil), chunk.NewLines...),
		IsEndOfFile:   chunk.IsEndOfFile,
	}
}

func convertParseError(err *iparse.ParseError) *ParseError {
	if err == nil {
		return nil
	}
	return &ParseError{Kind: ParseErrorKind(err.Kind), Message: err.Message, LineNumber: err.LineNumber}
}
