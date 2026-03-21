package applypatch

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func UnifiedDiffFromChunks(path string, chunks []UpdateFileChunk) (*ApplyPatchFileUpdate, error) {
	return UnifiedDiffFromChunksWithContext(path, chunks, 1)
}

func UnifiedDiffFromChunksWithContext(path string, chunks []UpdateFileChunk, context int) (*ApplyPatchFileUpdate, error) {
	original, err := os.ReadFile(path)
	if err != nil {
		return nil, &ApplyPatchError{IOError: &IoError{Context: "Failed to read file to update " + path, Source: err}}
	}
	newContent, err := deriveNewContent(string(original), path, chunks)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "apply-patch-go-diff-")
	if err != nil {
		return nil, &ApplyPatchError{IOError: &IoError{Context: "Failed to create temp directory", Source: err}}
	}
	defer os.RemoveAll(dir)
	oldPath := dir + "/old"
	newPath := dir + "/new"
	if err := os.WriteFile(oldPath, original, 0o600); err != nil {
		return nil, &ApplyPatchError{IOError: &IoError{Context: "Failed to write temp old file", Source: err}}
	}
	if err := os.WriteFile(newPath, []byte(newContent), 0o600); err != nil {
		return nil, &ApplyPatchError{IOError: &IoError{Context: "Failed to write temp new file", Source: err}}
	}
	cmd := exec.Command("diff", "-U"+strconv.Itoa(context), oldPath, newPath)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			// exit code 1 means files differ; that's the normal case here
			if len(out) == 0 {
				out = ee.Stderr
			}
			if ee.ExitCode() > 1 {
				return nil, &ApplyPatchError{Message: string(out)}
			}
		} else {
			return nil, &ApplyPatchError{IOError: &IoError{Context: "Failed to execute diff", Source: err}}
		}
	}
	return &ApplyPatchFileUpdate{UnifiedDiff: trimUnifiedHeaders(out), Content: newContent}, nil
}

func trimUnifiedHeaders(out []byte) string {
	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	for len(lines) > 0 && (strings.HasPrefix(lines[0], "--- ") || strings.HasPrefix(lines[0], "+++ ")) {
		lines = lines[1:]
	}
	return strings.TrimLeft(bytes.NewBufferString(strings.Join(lines, "\n")).String(), "\n")
}
