package applypatch

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runMainForTest(t *testing.T, args []string, stdin string) (int, string, string) {
	t.Helper()
	oldArgs, oldStdin, oldStdout, oldStderr := os.Args, os.Stdin, os.Stdout, os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stdin = oldStdin
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	stdinFile, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stdinFile.WriteString(stdin); err != nil {
		t.Fatal(err)
	}
	if _, err := stdinFile.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Args = args
	os.Stdin = stdinFile
	os.Stdout = stdoutW
	os.Stderr = stderrW

	code := RunMain()

	_ = stdoutW.Close()
	_ = stderrW.Close()
	stdout, err := io.ReadAll(stdoutR)
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := io.ReadAll(stderrR)
	if err != nil {
		t.Fatal(err)
	}
	return code, string(stdout), string(stderr)
}

func TestRunMainArgSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "arg-success.txt")
	patch := "*** Begin Patch\n*** Add File: " + path + "\n+hello\n*** End Patch"
	code, stdout, stderr := runMainForTest(t, []string{"apply_patch", patch}, "")
	if code != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%q", code, stderr)
	}
	if stdout != "Success. Updated the following files:\nA "+path+"\n" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestRunMainStdinSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdin-success.txt")
	patch := "*** Begin Patch\n*** Add File: " + path + "\n+hello\n*** End Patch"
	code, stdout, stderr := runMainForTest(t, []string{"apply_patch"}, patch)
	if code != 0 {
		t.Fatalf("unexpected exit code: %d stderr=%q", code, stderr)
	}
	if stdout != "Success. Updated the following files:\nA "+path+"\n" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestRunMainRejectsExtraArgs(t *testing.T) {
	code, stdout, stderr := runMainForTest(t, []string{"apply_patch", "one", "two"}, "")
	if code != 2 {
		t.Fatalf("unexpected exit code: %d", code)
	}
	if stdout != "" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
	if stderr != "Error: apply_patch accepts exactly one argument.\n" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestRunMainUsageOnEmptyStdin(t *testing.T) {
	code, stdout, stderr := runMainForTest(t, []string{"apply_patch"}, "")
	if code != 2 {
		t.Fatalf("unexpected exit code: %d", code)
	}
	if stdout != "" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
	if stderr != "Usage: apply_patch 'PATCH'\n       echo 'PATCH' | apply_patch\n" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestRunMainPatchFailure(t *testing.T) {
	patch := "*** Begin Patch\n*** Delete File: missing.txt\n*** End Patch"
	code, stdout, stderr := runMainForTest(t, []string{"apply_patch", patch}, "")
	if code != 1 {
		t.Fatalf("unexpected exit code: %d", code)
	}
	if stdout != "" {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
	if stderr != "Failed to delete file missing.txt\n" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestRunMainReadStdinError(t *testing.T) {
	oldArgs, oldStdin, oldStdout, oldStderr := os.Args, os.Stdin, os.Stdout, os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stdin = oldStdin
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	stdinFile, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatal(err)
	}
	if err := stdinFile.Close(); err != nil {
		t.Fatal(err)
	}

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Args = []string{"apply_patch"}
	os.Stdin = stdinFile
	os.Stdout = stdoutW
	os.Stderr = stderrW

	code := RunMain()

	_ = stdoutW.Close()
	_ = stderrW.Close()
	stdout, err := io.ReadAll(stdoutR)
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := io.ReadAll(stderrR)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 {
		t.Fatalf("unexpected exit code: %d", code)
	}
	if string(stdout) != "" {
		t.Fatalf("unexpected stdout: %q", string(stdout))
	}
	if string(stderr) == "" || !strings.HasPrefix(string(stderr), "Error: Failed to read PATCH from stdin.\n") {
		t.Fatalf("unexpected stderr: %q", string(stderr))
	}
}

func TestMainSubprocess(t *testing.T) {
	if os.Getenv("APPLYPATCH_MAIN_HELPER") == "1" {
		path := os.Getenv("APPLYPATCH_MAIN_PATH")
		os.Args = []string{"apply_patch", "*** Begin Patch\n*** Add File: " + path + "\n+hello\n*** End Patch"}
		Main()
		return
	}

	path := filepath.Join(t.TempDir(), "main-success.txt")
	cmd := exec.Command(os.Args[0], "-test.run=TestMainSubprocess$")
	cmd.Env = append(os.Environ(), "APPLYPATCH_MAIN_HELPER=1", "APPLYPATCH_MAIN_PATH="+path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper failed: %v\n%s", err, string(out))
	}
	if string(out) != "Success. Updated the following files:\nA "+path+"\n" {
		t.Fatalf("unexpected helper output: %q", string(out))
	}
}
