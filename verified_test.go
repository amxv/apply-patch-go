package applypatch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaybeParseApplyPatchVerifiedDeleteReadsContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "del.txt")
	if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := []string{"apply_patch", "*** Begin Patch\n*** Delete File: del.txt\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, dir)
	if got.Kind != MaybeApplyPatchVerifiedBody || got.Action == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	change, ok := got.Action.Changes()[path]
	if !ok {
		t.Fatalf("missing change for %s", path)
	}
	if change.Kind != ApplyPatchFileChangeDelete || change.Content != "x\n" {
		t.Fatalf("unexpected change: %+v", change)
	}
}

func TestMaybeParseApplyPatchVerifiedUpdateComputesNewContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "u.txt")
	if err := os.WriteFile(path, []byte("foo\nbar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := []string{"apply_patch", "*** Begin Patch\n*** Update File: u.txt\n@@\n foo\n-bar\n+baz\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, dir)
	if got.Kind != MaybeApplyPatchVerifiedBody || got.Action == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	change, ok := got.Action.Changes()[path]
	if !ok {
		t.Fatalf("missing change for %s", path)
	}
	if change.Kind != ApplyPatchFileChangeUpdate || change.NewContent != "foo\nbaz\n" {
		t.Fatalf("unexpected change: %+v", change)
	}
}

func TestMaybeParseApplyPatchVerifiedAddFileBody(t *testing.T) {
	dir := t.TempDir()
	argv := []string{"apply_patch", "*** Begin Patch\n*** Add File: new.txt\n+hello\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, dir)
	if got.Kind != MaybeApplyPatchVerifiedBody || got.Action == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	path := filepath.Join(dir, "new.txt")
	change, ok := got.Action.Changes()[path]
	if !ok {
		t.Fatalf("missing change for %s", path)
	}
	if change.Kind != ApplyPatchFileChangeAdd || change.Content != "hello\n" {
		t.Fatalf("unexpected change: %+v", change)
	}
}

func TestMaybeParseApplyPatchVerifiedUpdateCorrectnessErrorOnDiffFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "u.txt")
	if err := os.WriteFile(path, []byte("foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := []string{"apply_patch", "*** Begin Patch\n*** Update File: u.txt\n@@\n-missing\n+bar\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, dir)
	if got.Kind != MaybeApplyPatchVerifiedCorrectness || got.CorrectnessError == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	if !strings.Contains(got.CorrectnessError.Error(), "Failed to find expected lines in "+path) {
		t.Fatalf("unexpected correctness error: %v", got.CorrectnessError)
	}
}

func TestMaybeParseApplyPatchVerifiedImplicitInvocation(t *testing.T) {
	dir := t.TempDir()
	argv := []string{"*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, dir)
	if got.Kind != MaybeApplyPatchVerifiedCorrectness || got.CorrectnessError == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	err, ok := got.CorrectnessError.(*ApplyPatchError)
	if !ok || !err.ImplicitInvocation {
		t.Fatalf("unexpected correctness error: %#v", got.CorrectnessError)
	}
}

func TestMaybeParseApplyPatchVerifiedResolvesMovePath(t *testing.T) {
	dir := t.TempDir()
	workdir := filepath.Join(dir, "alt")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(workdir, "old.txt")
	if err := os.WriteFile(src, []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	script := "cd alt && apply_patch <<'PATCH'\n*** Begin Patch\n*** Update File: old.txt\n*** Move to: renamed.txt\n@@\n-before\n+after\n*** End Patch\nPATCH"
	got := MaybeParseApplyPatchVerified([]string{"bash", "-lc", script}, dir)
	if got.Kind != MaybeApplyPatchVerifiedBody || got.Action == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	change, ok := got.Action.Changes()[src]
	if !ok {
		t.Fatalf("missing change for %s", src)
	}
	expected := filepath.Join(workdir, "renamed.txt")
	if change.MovePath == nil || *change.MovePath != expected {
		t.Fatalf("unexpected move path: %+v", change.MovePath)
	}
}

func TestMaybeParseApplyPatchVerifiedUpdateIncludesUnifiedDiff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "u2.txt")
	if err := os.WriteFile(path, []byte("foo\nbar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := []string{"apply_patch", "*** Begin Patch\n*** Update File: u2.txt\n@@\n foo\n-bar\n+baz\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, dir)
	if got.Kind != MaybeApplyPatchVerifiedBody || got.Action == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	change, ok := got.Action.Changes()[path]
	if !ok {
		t.Fatalf("missing change for %s", path)
	}
	if change.UnifiedDiff == "" {
		t.Fatalf("expected unified diff, got empty change: %+v", change)
	}
	if !strings.Contains(change.UnifiedDiff, "@@") || !strings.Contains(change.UnifiedDiff, "-bar") || !strings.Contains(change.UnifiedDiff, "+baz") {
		t.Fatalf("unexpected unified diff: %q", change.UnifiedDiff)
	}
}

func TestMaybeParseApplyPatchVerifiedResolvesRelativePathsInCwd(t *testing.T) {
	dir := t.TempDir()
	rel := "source.txt"
	path := filepath.Join(dir, rel)
	if err := os.WriteFile(path, []byte("session directory content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := []string{"apply_patch", "*** Begin Patch\n*** Update File: source.txt\n@@\n-session directory content\n+updated session directory content\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, dir)
	if got.Kind != MaybeApplyPatchVerifiedBody || got.Action == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	change, ok := got.Action.Changes()[path]
	if !ok {
		t.Fatalf("missing change for %s", path)
	}
	if got.Action.Cwd != dir {
		t.Fatalf("unexpected cwd: %q", got.Action.Cwd)
	}
	if change.Kind != ApplyPatchFileChangeUpdate || change.NewContent != "updated session directory content\n" {
		t.Fatalf("unexpected change: %+v", change)
	}
}

func TestMaybeParseApplyPatchVerifiedImplicitPatchBashScriptIsError(t *testing.T) {
	dir := t.TempDir()
	argv := []string{"bash", "-lc", "*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, dir)
	if got.Kind != MaybeApplyPatchVerifiedCorrectness || got.CorrectnessError == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	err, ok := got.CorrectnessError.(*ApplyPatchError)
	if !ok || !err.ImplicitInvocation {
		t.Fatalf("unexpected correctness error: %#v", got.CorrectnessError)
	}
}

func TestMaybeParseApplyPatchVerifiedPropagatesShellParseError(t *testing.T) {
	dir := t.TempDir()
	argv := []string{"bash", "-lc", "apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, dir)
	if got.Kind != MaybeApplyPatchVerifiedShellParseError || got.ShellParseError == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	if *got.ShellParseError != ExtractHeredocFailedToFindHeredocBody {
		t.Fatalf("unexpected shell parse error: %+v", got.ShellParseError)
	}
}

func TestMaybeParseApplyPatchVerifiedNotApplyPatch(t *testing.T) {
	got := MaybeParseApplyPatchVerified([]string{"echo", "hello"}, t.TempDir())
	if got.Kind != MaybeApplyPatchVerifiedNotApplyPatch {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestMaybeParseApplyPatchVerifiedDeleteReadError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.txt")
	argv := []string{"apply_patch", "*** Begin Patch\n*** Delete File: " + path + "\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, "/")
	if got.Kind != MaybeApplyPatchVerifiedCorrectness || got.CorrectnessError == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	if !strings.Contains(got.CorrectnessError.Error(), "Failed to read "+path) {
		t.Fatalf("unexpected correctness error: %v", got.CorrectnessError)
	}
}

func TestMaybeParseApplyPatchVerifiedAbsoluteWorkdir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(path, []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := []string{"bash", "-lc", "cd '" + dir + "' && apply_patch <<'PATCH'\n*** Begin Patch\n*** Update File: source.txt\n@@\n-before\n+after\n*** End Patch\nPATCH"}
	got := MaybeParseApplyPatchVerified(argv, t.TempDir())
	if got.Kind != MaybeApplyPatchVerifiedBody || got.Action == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.Action.Cwd != dir {
		t.Fatalf("unexpected cwd: %q", got.Action.Cwd)
	}
}

func TestApplyPatchActionChangesAccessorAndNewAddForTest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	action := NewAddForTest(path, "hello")
	if action == nil {
		t.Fatal("expected action")
	}
	if action.Cwd != dir {
		t.Fatalf("unexpected cwd: %q", action.Cwd)
	}
	if action.IsEmpty() {
		t.Fatal("expected non-empty action")
	}
	change, ok := action.Changes()[path]
	if !ok {
		t.Fatalf("missing change for %s", path)
	}
	if change.Kind != ApplyPatchFileChangeAdd || change.Content != "hello" {
		t.Fatalf("unexpected change: %+v", change)
	}
}

func TestNewAddForTestPanicsOnRelativePath(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = NewAddForTest("relative.txt", "hello")
}

func TestMaybeParseApplyPatchVerifiedInvalidPatchIsNotApplyPatch(t *testing.T) {
	got := MaybeParseApplyPatchVerified([]string{"*** Begin Patch\n*** Nope File: foo\n*** End Patch"}, t.TempDir())
	if got.Kind != MaybeApplyPatchVerifiedNotApplyPatch {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestMaybeParseApplyPatchVerifiedShellInvalidPatchIsNotApplyPatch(t *testing.T) {
	argv := []string{"bash", "-lc", "apply_patch <<'PATCH'\n*** Begin Patch\n*** Nope File: foo\n*** End Patch\nPATCH"}
	got := MaybeParseApplyPatchVerified(argv, t.TempDir())
	if got.Kind != MaybeApplyPatchVerifiedNotApplyPatch {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestMaybeParseApplyPatchVerifiedLiteralPatchErrorFallsBack(t *testing.T) {
	argv := []string{"apply_patch", "*** Begin Patch\n*** Nope File: foo\n*** End Patch"}
	got := MaybeParseApplyPatchVerified(argv, t.TempDir())
	if got.Kind != MaybeApplyPatchVerifiedNotApplyPatch {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestMaybeParseApplyPatchVerifiedShellLiteralPatchErrorFallsBack(t *testing.T) {
	argv := []string{"bash", "-lc", "apply_patch <<'PATCH'\n*** Begin Patch\n*** Nope File: foo\n*** End Patch\nPATCH"}
	got := MaybeParseApplyPatchVerified(argv, t.TempDir())
	if got.Kind != MaybeApplyPatchVerifiedNotApplyPatch {
		t.Fatalf("unexpected result: %+v", got)
	}
}
