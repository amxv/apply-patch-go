package applypatch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type binaryRunResult struct {
	ExitCode int
	Output   []byte
}

type binaryParityCase struct {
	Name  string
	Args  []string
	Stdin string
	Setup func(t *testing.T, dir string)
}

func buildUpstreamRustApplyPatchBinary(t *testing.T) string {
	t.Helper()
	bin := os.Getenv("APPLY_PATCH_RUST_BIN")
	if bin == "" {
		t.Skip("set APPLY_PATCH_RUST_BIN to run Rust parity against an external upstream binary")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("APPLY_PATCH_RUST_BIN=%s is not usable: %v", bin, err)
	}
	return bin
}

func runApplyPatchBinary(t *testing.T, bin, dir string, stdin string, args ...string) binaryRunResult {
	t.Helper()
	cmd := exec.Command(bin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	if err == nil {
		return binaryRunResult{ExitCode: 0, Output: out}
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return binaryRunResult{ExitCode: exitErr.ExitCode(), Output: out}
	}
	t.Fatalf("run %s failed unexpectedly: %v", bin, err)
	return binaryRunResult{}
}

func runBinaryParityCase(t *testing.T, goBin, rustBin string, tc binaryParityCase) {
	t.Helper()
	goDir := t.TempDir()
	rustDir := t.TempDir()
	if tc.Setup != nil {
		tc.Setup(t, goDir)
		tc.Setup(t, rustDir)
	}
	goRes := runApplyPatchBinary(t, goBin, goDir, tc.Stdin, tc.Args...)
	rustRes := runApplyPatchBinary(t, rustBin, rustDir, tc.Stdin, tc.Args...)
	if goRes.ExitCode != rustRes.ExitCode {
		t.Fatalf("exit code mismatch: go=%d rust=%d\ngo output=%q\nrust output=%q", goRes.ExitCode, rustRes.ExitCode, string(goRes.Output), string(rustRes.Output))
	}
	if string(goRes.Output) != string(rustRes.Output) {
		t.Fatalf("output mismatch:\ngo=%q\nrust=%q", string(goRes.Output), string(rustRes.Output))
	}
	goSnap, err := snapshotDir(goDir)
	if err != nil {
		t.Fatalf("snapshot go dir: %v", err)
	}
	rustSnap, err := snapshotDir(rustDir)
	if err != nil {
		t.Fatalf("snapshot rust dir: %v", err)
	}
	if diff := compareSnapshots(rustSnap, goSnap); diff != "" {
		t.Fatalf("filesystem mismatch:\n%s", diff)
	}
}

func TestRustUpstreamBinaryParity(t *testing.T) {
	goBin := buildApplyPatchBinary(t)
	rustBin := buildUpstreamRustApplyPatchBinary(t)

	cases := []binaryParityCase{
		{
			Name: "cli_add_via_argv",
			Args: []string{"*** Begin Patch\n*** Add File: cli_test.txt\n+hello\n*** End Patch"},
		},
		{
			Name: "cli_update_via_argv",
			Args: []string{"*** Begin Patch\n*** Update File: cli_test.txt\n@@\n-hello\n+world\n*** End Patch"},
			Setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "cli_test.txt"), []byte("hello\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name:  "cli_add_via_stdin",
			Stdin: "*** Begin Patch\n*** Add File: cli_test_stdin.txt\n+hello\n*** End Patch",
		},
		{
			Name:  "cli_update_via_stdin",
			Stdin: "*** Begin Patch\n*** Update File: cli_test_stdin.txt\n@@\n-hello\n+world\n*** End Patch",
			Setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "cli_test_stdin.txt"), []byte("hello\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{Name: "usage_on_empty_stdin"},
		{Name: "rejects_extra_args", Args: []string{"one", "two"}},
		{Name: "rejects_empty_patch", Args: []string{"*** Begin Patch\n*** End Patch"}},
		{
			Name: "reports_missing_context",
			Args: []string{"*** Begin Patch\n*** Update File: modify.txt\n@@\n-missing\n+changed\n*** End Patch"},
			Setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "modify.txt"), []byte("line1\nline2\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{Name: "rejects_missing_file_delete", Args: []string{"*** Begin Patch\n*** Delete File: missing.txt\n*** End Patch"}},
		{Name: "rejects_empty_update_hunk", Args: []string{"*** Begin Patch\n*** Update File: foo.txt\n*** End Patch"}},
		{Name: "requires_existing_file_for_update", Args: []string{"*** Begin Patch\n*** Update File: missing.txt\n@@\n-old\n+new\n*** End Patch"}},
		{
			Name: "move_overwrites_existing_destination",
			Args: []string{"*** Begin Patch\n*** Update File: old/name.txt\n*** Move to: renamed/dir/name.txt\n@@\n-from\n+new\n*** End Patch"},
			Setup: func(t *testing.T, dir string) {
				t.Helper()
				oldPath := filepath.Join(dir, "old", "name.txt")
				dstPath := filepath.Join(dir, "renamed", "dir", "name.txt")
				if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(oldPath, []byte("from\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(dstPath, []byte("existing\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name: "add_overwrites_existing_file",
			Args: []string{"*** Begin Patch\n*** Add File: duplicate.txt\n+new content\n*** End Patch"},
			Setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "duplicate.txt"), []byte("old content\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name: "delete_directory_fails",
			Args: []string{"*** Begin Patch\n*** Delete File: dir\n*** End Patch"},
			Setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(dir, "dir"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{Name: "rejects_invalid_hunk_header", Args: []string{"*** Begin Patch\n*** Frobnicate File: foo\n*** End Patch"}},
		{
			Name: "update_appends_trailing_newline",
			Args: []string{"*** Begin Patch\n*** Update File: no_newline.txt\n@@\n-no newline at end\n+first line\n+second line\n*** End Patch"},
			Setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "no_newline.txt"), []byte("no newline at end"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name: "failure_after_partial_success_leaves_changes",
			Args: []string{"*** Begin Patch\n*** Add File: created.txt\n+hello\n*** Update File: missing.txt\n@@\n-old\n+new\n*** End Patch"},
		},
		{
			Name: "multiple_operations",
			Args: []string{"*** Begin Patch\n*** Add File: nested/new.txt\n+created\n*** Delete File: delete.txt\n*** Update File: modify.txt\n@@\n-line2\n+changed\n*** End Patch"},
			Setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "modify.txt"), []byte("line1\nline2\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "delete.txt"), []byte("obsolete\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name: "multiple_chunks",
			Args: []string{"*** Begin Patch\n*** Update File: multi.txt\n@@\n-line2\n+changed2\n@@\n-line4\n+changed4\n*** End Patch"},
			Setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "multi.txt"), []byte("line1\nline2\nline3\nline4\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			Name: "moves_file_to_new_directory",
			Args: []string{"*** Begin Patch\n*** Update File: old/name.txt\n*** Move to: renamed/dir/name.txt\n@@\n-old content\n+new content\n*** End Patch"},
			Setup: func(t *testing.T, dir string) {
				t.Helper()
				oldPath := filepath.Join(dir, "old", "name.txt")
				if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(oldPath, []byte("old content\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			runBinaryParityCase(t, goBin, rustBin, tc)
		})
	}

	scenariosDir := filepath.Join("codex-upstream", "codex-rs", "apply-patch", "tests", "fixtures", "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		t.Fatalf("read scenarios: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run("scenario_"+name, func(t *testing.T) {
			scenarioDir := filepath.Join(scenariosDir, name)
			goDir := t.TempDir()
			rustDir := t.TempDir()

			inputDir := filepath.Join(scenarioDir, "input")
			if info, err := os.Stat(inputDir); err == nil && info.IsDir() {
				if err := copyDirRecursive(inputDir, goDir); err != nil {
					t.Fatalf("copy go input: %v", err)
				}
				if err := copyDirRecursive(inputDir, rustDir); err != nil {
					t.Fatalf("copy rust input: %v", err)
				}
			}

			patchBytes, err := os.ReadFile(filepath.Join(scenarioDir, "patch.txt"))
			if err != nil {
				t.Fatalf("read patch: %v", err)
			}
			patch := string(patchBytes)

			goRes := runApplyPatchBinary(t, goBin, goDir, "", patch)
			rustRes := runApplyPatchBinary(t, rustBin, rustDir, "", patch)
			if goRes.ExitCode != rustRes.ExitCode {
				t.Fatalf("exit code mismatch: go=%d rust=%d\ngo output=%q\nrust output=%q", goRes.ExitCode, rustRes.ExitCode, string(goRes.Output), string(rustRes.Output))
			}
			if string(goRes.Output) != string(rustRes.Output) {
				t.Fatalf("output mismatch:\ngo=%q\nrust=%q", string(goRes.Output), string(rustRes.Output))
			}

			goSnap, err := snapshotDir(goDir)
			if err != nil {
				t.Fatalf("snapshot go: %v", err)
			}
			rustSnap, err := snapshotDir(rustDir)
			if err != nil {
				t.Fatalf("snapshot rust: %v", err)
			}
			if diff := compareSnapshots(rustSnap, goSnap); diff != "" {
				t.Fatalf("scenario filesystem mismatch:\n%s", diff)
			}
		})
	}
}
