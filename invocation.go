package applypatch

import "strings"

type MaybeApplyPatchKind string

type ExtractHeredocError string

const (
	MaybeApplyPatchBody            MaybeApplyPatchKind = "body"
	MaybeApplyPatchShellParseError MaybeApplyPatchKind = "shell_parse_error"
	MaybeApplyPatchPatchError      MaybeApplyPatchKind = "patch_parse_error"
	MaybeApplyPatchNotApplyPatch   MaybeApplyPatchKind = "not_apply_patch"
)

const (
	ExtractHeredocCommandDidNotStartWithApplyPatch ExtractHeredocError = "command_did_not_start_with_apply_patch"
	ExtractHeredocFailedToFindHeredocBody          ExtractHeredocError = "failed_to_find_heredoc_body"
)

type MaybeApplyPatch struct {
	Kind            MaybeApplyPatchKind
	Args            *ApplyPatchArgs
	PatchError      *ParseError
	ShellParseError *ExtractHeredocError
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
		body, workdir, shellErr := extractApplyPatchFromScript(script)
		if shellErr == nil {
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
		if *shellErr == ExtractHeredocCommandDidNotStartWithApplyPatch {
			return MaybeApplyPatch{Kind: MaybeApplyPatchNotApplyPatch}
		}
		return MaybeApplyPatch{Kind: MaybeApplyPatchShellParseError, ShellParseError: shellErr}
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

func extractApplyPatchFromScript(script string) (body string, workdir string, shellErr *ExtractHeredocError) {
	script = strings.TrimSpace(script)
	if strings.HasPrefix(script, "cd ") {
		idx := strings.Index(script, "&&")
		if idx < 0 {
			err := ExtractHeredocCommandDidNotStartWithApplyPatch
			return "", "", &err
		}
		rawCd := strings.TrimSpace(script[3:idx])
		if !isSingleShellWord(rawCd) {
			err := ExtractHeredocCommandDidNotStartWithApplyPatch
			return "", "", &err
		}
		workdir = trimShellWord(rawCd)
		script = strings.TrimSpace(script[idx+2:])
	}
	idx := strings.Index(script, "<<")
	if idx < 0 {
		if isApplyPatchCommand(strings.TrimSpace(script)) {
			err := ExtractHeredocFailedToFindHeredocBody
			return "", workdir, &err
		}
		err := ExtractHeredocCommandDidNotStartWithApplyPatch
		return "", workdir, &err
	}
	cmdPart := strings.TrimSpace(script[:idx])
	if !(cmdPart == "apply_patch" || cmdPart == "applypatch") {
		err := ExtractHeredocCommandDidNotStartWithApplyPatch
		return "", workdir, &err
	}
	rest := strings.TrimSpace(script[idx+2:])
	lineEnd := strings.Index(rest, "\n")
	if lineEnd < 0 {
		err := ExtractHeredocFailedToFindHeredocBody
		return "", workdir, &err
	}
	marker := trimShellWord(strings.TrimSpace(rest[:lineEnd]))
	payload := rest[lineEnd+1:]
	trimmedPayload := strings.TrimRight(payload, "\n")
	endMarker := "\n" + marker
	if !strings.HasSuffix(trimmedPayload, endMarker) {
		if strings.Contains(trimmedPayload, endMarker) {
			err := ExtractHeredocCommandDidNotStartWithApplyPatch
			return "", workdir, &err
		}
		err := ExtractHeredocFailedToFindHeredocBody
		return "", workdir, &err
	}
	body = strings.TrimSuffix(trimmedPayload, endMarker)
	return body, workdir, nil
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

func isSingleShellWord(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return true
		}
	}
	return !strings.ContainsAny(s, " 	")
}
