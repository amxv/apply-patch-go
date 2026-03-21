package applypatch

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func ApplyPatch(patch string, stdout io.Writer, stderr io.Writer) error {
	args, err := ParsePatch(patch)
	if err != nil {
		perr := err.(*ParseError)
		if perr.Kind == ParseErrorInvalidPatch {
			_, _ = io.WriteString(stderr, "Invalid patch: "+perr.Message+"\n")
		} else {
			_, _ = io.WriteString(stderr, "Invalid patch hunk on line "+strconv.Itoa(perr.LineNumber)+": "+perr.Message+"\n")
		}
		return &ApplyPatchError{ParseError: perr}
	}
	return ApplyHunks(args.Hunks, stdout, stderr)
}

func ApplyHunks(hunks []Hunk, stdout io.Writer, stderr io.Writer) error {
	affected, err := applyHunksToFiles(hunks)
	if err != nil {
		_, _ = io.WriteString(stderr, err.Error()+"\n")
		return err
	}
	return PrintSummary(affected, stdout)
}

func applyHunksToFiles(hunks []Hunk) (*AffectedPaths, error) {
	if len(hunks) == 0 {
		return nil, &ApplyPatchError{IOError: &IoError{Context: "No files were modified.", Source: nil}}
	}
	affected := &AffectedPaths{}
	for _, hunk := range hunks {
		switch hunk.Kind {
		case HunkAddFile:
			if err := applyAddFile(hunk); err != nil {
				return nil, err
			}
			affected.Added = append(affected.Added, hunk.Path)
		case HunkDeleteFile:
			if err := applyDeleteFile(hunk); err != nil {
				return nil, err
			}
			affected.Deleted = append(affected.Deleted, hunk.Path)
		case HunkUpdateFile:
			if err := applyUpdateFile(hunk); err != nil {
				return nil, err
			}
			if hunk.MovePath != nil {
				affected.Modified = append(affected.Modified, *hunk.MovePath)
			} else {
				affected.Modified = append(affected.Modified, hunk.Path)
			}
		}
	}
	return affected, nil
}

func applyAddFile(hunk Hunk) error {
	if err := ensureParentDir(hunk.Path); err != nil {
		return err
	}
	if err := os.WriteFile(hunk.Path, []byte(hunk.Contents), 0o644); err != nil {
		return &ApplyPatchError{IOError: &IoError{Context: "Failed to write file " + hunk.Path, Source: err}}
	}
	return nil
}

func applyDeleteFile(hunk Hunk) error {
	info, err := os.Stat(hunk.Path)
	if err != nil {
		return &ApplyPatchError{Message: "Failed to delete file " + hunk.Path}
	}
	if info.IsDir() {
		return &ApplyPatchError{Message: "Failed to delete file " + hunk.Path}
	}
	if err := os.Remove(hunk.Path); err != nil {
		return &ApplyPatchError{Message: "Failed to delete file " + hunk.Path}
	}
	return nil
}

func applyUpdateFile(hunk Hunk) error {
	content, err := os.ReadFile(hunk.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ApplyPatchError{Message: "Failed to read file to update " + hunk.Path + ": No such file or directory (os error 2)"}
		}
		return &ApplyPatchError{IOError: &IoError{Context: "Failed to read file to update " + hunk.Path, Source: err}}
	}
	newContent, err := deriveNewContent(string(content), hunk.Path, hunk.Chunks)
	if err != nil {
		return err
	}
	path := hunk.Path
	if hunk.MovePath != nil {
		path = *hunk.MovePath
		if err := ensureParentDir(path); err != nil {
			return err
		}
	}
	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return &ApplyPatchError{IOError: &IoError{Context: "Failed to write file " + path, Source: err}}
	}
	if hunk.MovePath != nil {
		if err := os.Remove(hunk.Path); err != nil {
			return &ApplyPatchError{IOError: &IoError{Context: "Failed to remove original " + hunk.Path, Source: err}}
		}
	}
	return nil
}

func deriveNewContent(original string, path string, chunks []UpdateFileChunk) (string, error) {
	lines := strings.Split(original, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	repls, err := computeReplacements(lines, path, chunks)
	if err != nil {
		return "", err
	}
	updated := applyReplacements(lines, repls)
	if len(updated) == 0 || updated[len(updated)-1] != "" {
		updated = append(updated, "")
	}
	return strings.Join(updated, "\n"), nil
}

func ensureParentDir(path string) error {
	idx := strings.LastIndex(path, "/")
	if idx <= 0 {
		return nil
	}
	if err := os.MkdirAll(path[:idx], 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return &ApplyPatchError{IOError: &IoError{Context: fmt.Sprintf("Failed to create parent directories for %s", path), Source: err}}
	}
	return nil
}

type replacement struct {
	Start    int
	OldLen   int
	NewLines []string
}

func computeReplacements(originalLines []string, path string, chunks []UpdateFileChunk) ([]replacement, error) {
	repls := make([]replacement, 0, len(chunks))
	lineIndex := 0
	for _, chunk := range chunks {
		if chunk.ChangeContext != nil {
			if idx := seekSequence(originalLines, []string{*chunk.ChangeContext}, lineIndex, false); idx != nil {
				lineIndex = *idx + 1
			} else {
				return nil, &ApplyPatchError{Message: fmt.Sprintf("Failed to find context '%s' in %s", *chunk.ChangeContext, path)}
			}
		}
		if len(chunk.OldLines) == 0 {
			insertionIdx := len(originalLines)
			repls = append(repls, replacement{Start: insertionIdx, OldLen: 0, NewLines: append([]string(nil), chunk.NewLines...)})
			continue
		}
		pattern := append([]string(nil), chunk.OldLines...)
		newSlice := append([]string(nil), chunk.NewLines...)
		found := seekSequence(originalLines, pattern, lineIndex, chunk.IsEndOfFile)
		if found == nil && len(pattern) > 0 && pattern[len(pattern)-1] == "" {
			pattern = pattern[:len(pattern)-1]
			if len(newSlice) > 0 && newSlice[len(newSlice)-1] == "" {
				newSlice = newSlice[:len(newSlice)-1]
			}
			found = seekSequence(originalLines, pattern, lineIndex, chunk.IsEndOfFile)
		}
		if found == nil {
			return nil, &ApplyPatchError{Message: fmt.Sprintf("Failed to find expected lines in %s:\n%s", path, strings.Join(chunk.OldLines, "\n"))}
		}
		repls = append(repls, replacement{Start: *found, OldLen: len(pattern), NewLines: newSlice})
		lineIndex = *found + len(pattern)
	}
	return repls, nil
}

func applyReplacements(lines []string, repls []replacement) []string {
	updated := append([]string(nil), lines...)
	for i := len(repls) - 1; i >= 0; i-- {
		r := repls[i]
		prefix := append([]string(nil), updated[:r.Start]...)
		suffixStart := r.Start + r.OldLen
		if suffixStart > len(updated) {
			suffixStart = len(updated)
		}
		suffix := append([]string(nil), updated[suffixStart:]...)
		updated = append(prefix, append(append([]string(nil), r.NewLines...), suffix...)...)
	}
	return updated
}

func PrintSummary(affected *AffectedPaths, out io.Writer) error {
	_, err := io.WriteString(out, "Success. Updated the following files:\n")
	if err != nil {
		return err
	}
	for _, path := range affected.Added {
		if _, err := io.WriteString(out, "A "+path+"\n"); err != nil {
			return err
		}
	}
	for _, path := range affected.Modified {
		if _, err := io.WriteString(out, "M "+path+"\n"); err != nil {
			return err
		}
	}
	for _, path := range affected.Deleted {
		if _, err := io.WriteString(out, "D "+path+"\n"); err != nil {
			return err
		}
	}
	return nil
}
