package applypatch

import "strings"

type MaybeApplyPatchKind string

const (
	MaybeApplyPatchBody          MaybeApplyPatchKind = "body"
	MaybeApplyPatchPatchError    MaybeApplyPatchKind = "patch_parse_error"
	MaybeApplyPatchNotApplyPatch MaybeApplyPatchKind = "not_apply_patch"
)

type MaybeApplyPatch struct {
	Kind       MaybeApplyPatchKind
	Args       *ApplyPatchArgs
	PatchError *ParseError
}

func MaybeParseApplyPatch(argv []string) MaybeApplyPatch {
	if len(argv) == 2 && isApplyPatchCommand(argv[0]) {
		args, err := ParsePatch(argv[1])
		if err != nil {
			return MaybeApplyPatch{Kind: MaybeApplyPatchPatchError, PatchError: err.(*ParseError)}
		}
		return MaybeApplyPatch{Kind: MaybeApplyPatchBody, Args: args}
	}
	if script, ok := parseShellScript(argv); ok {
		body, workdir, ok := extractApplyPatchFromScript(script)
		if ok {
			args, err := ParsePatch(body)
			if err != nil {
				return MaybeApplyPatch{Kind: MaybeApplyPatchPatchError, PatchError: err.(*ParseError)}
			}
			if workdir != "" {
				wd := workdir
				args.Workdir = &wd
			}
			return MaybeApplyPatch{Kind: MaybeApplyPatchBody, Args: args}
		}
	}
	return MaybeApplyPatch{Kind: MaybeApplyPatchNotApplyPatch}
}

func isApplyPatchCommand(cmd string) bool {
	base := shellBase(cmd)
	return base == "apply_patch" || base == "applypatch"
}

func parseShellScript(argv []string) (string, bool) {
	if len(argv) == 3 && isShellCommand(argv[0], argv[1]) {
		return argv[2], true
	}
	if len(argv) == 4 {
		base := shellBase(argv[0])
		if (base == "pwsh" || base == "powershell") && strings.EqualFold(argv[1], "-noprofile") && strings.EqualFold(argv[2], "-command") {
			return argv[3], true
		}
	}
	return "", false
}

func isShellCommand(shell string, flag string) bool {
	base := shellBase(shell)
	switch base {
	case "bash", "zsh", "sh":
		return flag == "-lc" || flag == "-c"
	case "pwsh", "powershell":
		return strings.EqualFold(flag, "-command")
	case "cmd":
		return strings.EqualFold(flag, "/c")
	default:
		return false
	}
}

func shellBase(shell string) string {
	shell = strings.ReplaceAll(shell, "\\", "/")
	if idx := strings.LastIndex(shell, "/"); idx >= 0 {
		shell = shell[idx+1:]
	}
	shell = strings.ToLower(shell)
	return strings.TrimSuffix(shell, ".exe")
}

func extractApplyPatchFromScript(script string) (body string, workdir string, ok bool) {
	script = strings.TrimSpace(script)
	if strings.HasPrefix(script, "cd ") {
		idx := strings.Index(script, "&&")
		if idx < 0 {
			return "", "", false
		}
		workdir = trimShellWord(strings.TrimSpace(script[3:idx]))
		script = strings.TrimSpace(script[idx+2:])
	}
	idx := strings.Index(script, "<<")
	if idx < 0 {
		return "", workdir, false
	}
	cmdPart := strings.TrimSpace(script[:idx])
	if !(cmdPart == "apply_patch" || cmdPart == "applypatch") {
		return "", workdir, false
	}
	rest := strings.TrimSpace(script[idx+2:])
	lineEnd := strings.Index(rest, "\n")
	if lineEnd < 0 {
		return "", "", false
	}
	marker := trimShellWord(strings.TrimSpace(rest[:lineEnd]))
	payload := rest[lineEnd+1:]
	endMarker := "\n" + marker
	endIdx := strings.LastIndex(payload, endMarker)
	if endIdx < 0 {
		return "", workdir, false
	}
	return strings.TrimRight(payload[:endIdx], "\n"), workdir, true
}

func trimShellWord(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
