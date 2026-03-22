package applypatch

import "testing"

func TestMaybeParseApplyPatchLiteral(t *testing.T) {
	argv := []string{"apply_patch", "*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch"}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchBody || got.Args == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	if len(got.Args.Hunks) != 1 || got.Args.Hunks[0].Kind != HunkAddFile {
		t.Fatalf("unexpected hunks: %+v", got.Args.Hunks)
	}
}

func TestMaybeParseApplyPatchShellHeredoc(t *testing.T) {
	script := "apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchBody || got.Args == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.Args.Workdir != nil {
		t.Fatalf("unexpected workdir: %v", *got.Args.Workdir)
	}
}

func TestMaybeParseApplyPatchShellHeredocWithCd(t *testing.T) {
	script := "cd alt && apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchBody || got.Args == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.Args.Workdir == nil || *got.Args.Workdir != "alt" {
		t.Fatalf("unexpected workdir: %+v", got.Args.Workdir)
	}
}

func TestMaybeParseApplyPatchAlias(t *testing.T) {
	argv := []string{"applypatch", "*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch"}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchBody || got.Args == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestMaybeParseApplyPatchLiteralPatchError(t *testing.T) {
	argv := []string{"apply_patch", "*** Begin Patch\n*** Frobnicate File: foo\n*** End Patch"}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchPatchError || got.PatchError == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestMaybeParseApplyPatchPowerShellNoProfile(t *testing.T) {
	script := "apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"powershell.exe", "-NoProfile", "-Command", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchBody || got.Args == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestMaybeParseApplyPatchQuotedCdPath(t *testing.T) {
	script := "cd 'foo bar' && apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchBody || got.Args == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.Args.Workdir == nil || *got.Args.Workdir != "foo bar" {
		t.Fatalf("unexpected workdir: %+v", got.Args.Workdir)
	}
}

func TestMaybeParseApplyPatchRejectsHeredocWithExtraArg(t *testing.T) {
	script := "apply_patch foo <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchNotApplyPatch {
		t.Fatalf("expected not-apply-patch, got %+v", got)
	}
}

func TestMaybeParseApplyPatchRejectsTrailingCommandsAfterHeredoc(t *testing.T) {
	script := "cd bar && apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH && echo done"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchNotApplyPatch {
		t.Fatalf("expected not-apply-patch, got %+v", got)
	}
}

func TestMaybeParseApplyPatchRejectsCdWithTwoArgs(t *testing.T) {
	script := "cd foo bar && apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchNotApplyPatch {
		t.Fatalf("expected not-apply-patch, got %+v", got)
	}
}

func TestMaybeParseApplyPatchRejectsCdWithSemicolon(t *testing.T) {
	script := "cd foo; apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchNotApplyPatch {
		t.Fatalf("expected not-apply-patch, got %+v", got)
	}
}

func TestMaybeParseApplyPatchRejectsCdWithOr(t *testing.T) {
	script := "cd foo || apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchNotApplyPatch {
		t.Fatalf("expected not-apply-patch, got %+v", got)
	}
}

func TestMaybeParseApplyPatchRejectsCdWithPipe(t *testing.T) {
	script := "cd foo | apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchNotApplyPatch {
		t.Fatalf("expected not-apply-patch, got %+v", got)
	}
}

func TestMaybeParseApplyPatchRejectsDoubleCd(t *testing.T) {
	script := "cd foo && cd bar && apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchNotApplyPatch {
		t.Fatalf("expected not-apply-patch, got %+v", got)
	}
}

func TestMaybeParseApplyPatchPowerShellHeredoc(t *testing.T) {
	script := "apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"powershell", "-Command", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchBody || got.Args == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestMaybeParseApplyPatchPwshHeredoc(t *testing.T) {
	script := "apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"pwsh", "-Command", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchBody || got.Args == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestMaybeParseApplyPatchCmdHeredocWithCd(t *testing.T) {
	script := "cd foo && apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"cmd.exe", "/c", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchBody || got.Args == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.Args.Workdir == nil || *got.Args.Workdir != "foo" {
		t.Fatalf("unexpected workdir: %+v", got.Args.Workdir)
	}
}

func TestMaybeParseApplyPatchShellParseErrorWhenHeredocBodyMissing(t *testing.T) {
	script := "apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchShellParseError || got.ShellParseError == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
	if *got.ShellParseError != ExtractHeredocFailedToFindHeredocBody {
		t.Fatalf("unexpected shell parse error: %+v", got.ShellParseError)
	}
}

func TestMaybeParseApplyPatchShellScriptInvalidPatch(t *testing.T) {
	script := "apply_patch <<'PATCH'\n*** Begin Patch\n*** Frobnicate File: foo\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchPatchError || got.PatchError == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestMaybeParseApplyPatchNotApplyPatchFallback(t *testing.T) {
	got := MaybeParseApplyPatch([]string{"echo", "hello"})
	if got.Kind != MaybeApplyPatchNotApplyPatch {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestMaybeParseApplyPatchRejectsEchoThenApplyPatch(t *testing.T) {
	script := "echo foo && apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchNotApplyPatch {
		t.Fatalf("expected not-apply-patch, got %+v", got)
	}
}

func TestMaybeParseApplyPatchRejectsEchoThenCdThenApplyPatch(t *testing.T) {
	script := "echo foo; cd bar && apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchNotApplyPatch {
		t.Fatalf("expected not-apply-patch, got %+v", got)
	}
}

func TestMaybeParseApplyPatchHeredocNonLoginShell(t *testing.T) {
	script := "apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-c", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchBody || got.Args == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestMaybeParseApplyPatchHeredocAlias(t *testing.T) {
	script := "applypatch <<'PATCH'\n*** Begin Patch\n*** Add File: foo\n+hi\n*** End Patch\nPATCH"
	argv := []string{"bash", "-lc", script}
	got := MaybeParseApplyPatch(argv)
	if got.Kind != MaybeApplyPatchBody || got.Args == nil {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestIsShellCommandAndShellBaseHelpers(t *testing.T) {
	if !isShellCommand("/bin/sh", "-c") {
		t.Fatal("expected sh -c to be recognized")
	}
	if isShellCommand("python", "-c") {
		t.Fatal("unexpected python recognition")
	}
	if got := shellBase(`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`); got != "powershell" {
		t.Fatalf("unexpected shell base: %q", got)
	}
}

func TestExtractApplyPatchFromScriptDirectBranches(t *testing.T) {
	body, workdir, shellErr := extractApplyPatchFromScript("apply_patch")
	if body != "" || workdir != "" || shellErr == nil || *shellErr != ExtractHeredocFailedToFindHeredocBody {
		t.Fatalf("unexpected result: body=%q workdir=%q err=%v", body, workdir, shellErr)
	}
	body, workdir, shellErr = extractApplyPatchFromScript("echo hello")
	if body != "" || workdir != "" || shellErr == nil || *shellErr != ExtractHeredocCommandDidNotStartWithApplyPatch {
		t.Fatalf("unexpected result: body=%q workdir=%q err=%v", body, workdir, shellErr)
	}
	body, workdir, shellErr = extractApplyPatchFromScript("apply_patch <<EOF")
	if body != "" || shellErr == nil || *shellErr != ExtractHeredocFailedToFindHeredocBody {
		t.Fatalf("unexpected result: body=%q workdir=%q err=%v", body, workdir, shellErr)
	}
	body, workdir, shellErr = extractApplyPatchFromScript("apply_patch <<EOF\nbody\nEOF\ntrailing")
	if body != "" || shellErr == nil || *shellErr != ExtractHeredocCommandDidNotStartWithApplyPatch {
		t.Fatalf("unexpected result: body=%q workdir=%q err=%v", body, workdir, shellErr)
	}
}
