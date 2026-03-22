package applypatch

import (
	"io"
	"os"
)

func Main() {
	os.Exit(RunMain())
}

func RunMain() int {
	args := os.Args
	if len(args) > 2 {
		_, _ = io.WriteString(os.Stderr, "Error: apply_patch accepts exactly one argument.\n")
		return 2
	}
	var patchArg string
	if len(args) == 2 {
		patchArg = args[1]
	} else {
		buf, err := io.ReadAll(os.Stdin)
		if err != nil {
			_, _ = io.WriteString(os.Stderr, "Error: Failed to read PATCH from stdin.\n"+err.Error()+"\n")
			return 1
		}
		patchArg = string(buf)
		if patchArg == "" {
			_, _ = io.WriteString(os.Stderr, "Usage: apply_patch 'PATCH'\n       echo 'PATCH' | apply_patch\n")
			return 2
		}
	}
	if err := ApplyPatch(patchArg, os.Stdout, os.Stderr); err != nil {
		return 1
	}
	return 0
}
