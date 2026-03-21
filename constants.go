package applypatch

import _ "embed"

const CODEX_CORE_APPLY_PATCH_ARG1 = "--codex-run-as-apply-patch"

//go:embed codex-upstream/codex-rs/apply-patch/apply_patch_tool_instructions.md
var APPLY_PATCH_TOOL_INSTRUCTIONS string
