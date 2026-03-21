package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCmdMainSubprocess(t *testing.T) {
	if os.Getenv("CMD_APPLYPATCH_MAIN_HELPER") == "1" {
		path := os.Getenv("CMD_APPLYPATCH_MAIN_PATH")
		os.Args = []string{"apply_patch", "*** Begin Patch\n*** Add File: " + path + "\n+hello\n*** End Patch"}
		main()
		return
	}

	path := filepath.Join(t.TempDir(), "cmd-main-success.txt")
	cmd := exec.Command(os.Args[0], "-test.run=TestCmdMainSubprocess$")
	cmd.Env = append(os.Environ(), "CMD_APPLYPATCH_MAIN_HELPER=1", "CMD_APPLYPATCH_MAIN_PATH="+path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper failed: %v\n%s", err, string(out))
	}
	if string(out) != "Success. Updated the following files:\nA "+path+"\n" {
		t.Fatalf("unexpected helper output: %q", string(out))
	}
}
