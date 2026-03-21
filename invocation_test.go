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
