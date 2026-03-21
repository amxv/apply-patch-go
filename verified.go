package applypatch

import (
	"os"
	"path/filepath"
)

type MaybeApplyPatchVerifiedKind string

const (
	MaybeApplyPatchVerifiedBody          MaybeApplyPatchVerifiedKind = "body"
	MaybeApplyPatchVerifiedCorrectness   MaybeApplyPatchVerifiedKind = "correctness_error"
	MaybeApplyPatchVerifiedNotApplyPatch MaybeApplyPatchVerifiedKind = "not_apply_patch"
)

type MaybeApplyPatchVerified struct {
	Kind             MaybeApplyPatchVerifiedKind
	Action           *ApplyPatchAction
	CorrectnessError error
}

func MaybeParseApplyPatchVerified(argv []string, cwd string) MaybeApplyPatchVerified {
	if len(argv) == 1 {
		if _, err := ParsePatch(argv[0]); err == nil {
			return MaybeApplyPatchVerified{Kind: MaybeApplyPatchVerifiedCorrectness, CorrectnessError: &ApplyPatchError{ImplicitInvocation: true}}
		}
	}
	parsed := MaybeParseApplyPatch(argv)
	if parsed.Kind != MaybeApplyPatchBody || parsed.Args == nil {
		return MaybeApplyPatchVerified{Kind: MaybeApplyPatchVerifiedNotApplyPatch}
	}
	effectiveCwd := cwd
	if parsed.Args.Workdir != nil {
		if filepath.IsAbs(*parsed.Args.Workdir) {
			effectiveCwd = *parsed.Args.Workdir
		} else {
			effectiveCwd = filepath.Join(cwd, *parsed.Args.Workdir)
		}
	}
	changes := map[string]ApplyPatchFileChange{}
	for _, hunk := range parsed.Args.Hunks {
		path := hunk.ResolvePath(effectiveCwd)
		switch hunk.Kind {
		case HunkAddFile:
			changes[path] = ApplyPatchFileChange{Kind: ApplyPatchFileChangeAdd, Content: hunk.Contents}
		case HunkDeleteFile:
			content, err := os.ReadFile(path)
			if err != nil {
				return MaybeApplyPatchVerified{Kind: MaybeApplyPatchVerifiedCorrectness, CorrectnessError: &ApplyPatchError{IOError: &IoError{Context: "Failed to read " + path, Source: err}}}
			}
			changes[path] = ApplyPatchFileChange{Kind: ApplyPatchFileChangeDelete, Content: string(content)}
		case HunkUpdateFile:
			diffUpdate, err := UnifiedDiffFromChunks(path, hunk.Chunks)
			if err != nil {
				return MaybeApplyPatchVerified{Kind: MaybeApplyPatchVerifiedCorrectness, CorrectnessError: err}
			}
			var movePath *string
			if hunk.MovePath != nil {
				resolved := filepath.Join(effectiveCwd, *hunk.MovePath)
				movePath = &resolved
			}
			changes[path] = ApplyPatchFileChange{Kind: ApplyPatchFileChangeUpdate, MovePath: movePath, UnifiedDiff: diffUpdate.UnifiedDiff, NewContent: diffUpdate.Content}
		}
	}
	return MaybeApplyPatchVerified{Kind: MaybeApplyPatchVerifiedBody, Action: &ApplyPatchAction{Changes: changes, Patch: parsed.Args.Patch, Cwd: effectiveCwd}}
}
