# apply-patch Go port status

## High-level goal

This repository is porting the `apply-patch` crate from the Codex upstream repository to Go.

The goal is not to build a merely similar tool. The goal is an exact one-to-one port with the same implementation behavior, the same observable semantics, and the same user-facing results as the upstream Rust crate.

That means matching the upstream crate at multiple layers:

- patch parsing behavior
- patch application behavior
- CLI behavior
- tool stdout/stderr behavior
- verified invocation parsing
- shell/heredoc extraction behavior
- unified diff generation
- exported helper/public API surface

## Upstream source of truth

The upstream Rust crate lives under:

- `codex-upstream/codex-rs/apply-patch`

The port is being measured directly against that source, not against a hand-written approximation.

## Broad progress so far

Work completed so far includes:

### 1. Core Go implementation

The Go port now contains working implementations for:

- patch parsing
- patch application
- file add / delete / update / move
- verified apply-patch invocation parsing
- shell/heredoc parsing for `bash`, `sh`, `zsh`, `powershell`, `pwsh`, and `cmd`
- unified diff helpers
- CLI entrypoint in `cmd/apply_patch`

### 2. Upstream scenario fixture runner

The repository runs the upstream filesystem scenario corpus directly from:

- `codex-upstream/codex-rs/apply-patch/tests/fixtures/scenarios`

That gives a direct behavior check against the same scenario fixtures used by the upstream Rust crate.

### 3. Tool-style behavior coverage

The Go suite now mirrors a broad slice of the upstream `tool.rs` behavior, including:

- empty patch rejection
- missing-context failure
- missing-file delete failure
- missing update file failure
- invalid hunk header rejection
- empty update hunk rejection
- delete-directory failure
- multiple-operation summaries
- multiple-chunk updates
- move behavior
- overwrite behavior
- trailing-newline behavior
- partial-success semantics

### 4. Real CLI binary coverage

The test suite exercises the real built binary from `cmd/apply_patch`, not just in-process library helpers.

That coverage includes:

- argv invocation
- stdin invocation
- usage / extra-arg behavior
- success summaries
- failure exit codes
- many upstream `tool.rs`-style CLI cases

### 5. Invocation / verified parser parity work

The Go port now includes broad coverage for upstream invocation behavior, including:

- literal `apply_patch` parsing
- `applypatch` alias handling
- heredoc parsing
- non-login shell forms
- PowerShell / pwsh / cmd forms
- quoted `cd` workdir handling
- negative shell-shape cases that must be ignored
- implicit invocation correctness errors
- shell parse error propagation

### 6. Unified diff parity work

The Go suite includes exact unified diff shape checks for key upstream cases, including:

- first-line replacement
- last-line replacement
- insert-at-EOF
- interleaved changes
- custom context radius behavior
- diff execution failure paths
- temp-directory creation failure paths

### 7. Public API / helper parity work

The Go package now exposes more of the upstream crate’s public helper surface, including:

- `CODEX_CORE_APPLY_PATCH_ARG1`
- `APPLY_PATCH_TOOL_INSTRUCTIONS`
- `ApplyHunks`
- `UnifiedDiffFromChunksWithContext`
- `ApplyPatchAction.Changes()`
- `NewAddForTest`

## Major blocker that has already been fixed

A previous direct library-style upstream test exposed a real engine bug in the Go patch engine for a patch made of:

- a pure-addition update chunk
- followed by a later removal/replacement chunk

That replacement-ordering bug in `apply.go` has now been fixed.

This is important because the repo is no longer in the earlier “broadly green but still known-core-engine-buggy” state. That blocker was real, was reproduced, and is now fixed.

## Verified clean checkpoints on the VPS

Progress has been preserved incrementally in local commits on the VPS. Important verified checkpoints include:

- `28401b6` — direct parity tests and broad coverage expansion
- `31a2ec1` — targeted helper branch coverage tests
- `f8ca1cb` — move-parent and verified add-file branch coverage
- `00955ff` — parser and verified fallback branch coverage

## Latest fully verified status

The repository has now moved beyond commit `00955ff`.

On the current working tree on the VPS, the following have been re-verified successfully:

- `go test ./...`
- `go test ./... -coverprofile=cover.out`

The current verified totals are now:

- root package coverage: **100.0%**
- `cmd/apply_patch` coverage: **100.0%**
- total statement coverage in `cover.out`: **100.0%**

This means the previous remaining uncovered function has now been fully covered as well:

- `UnifiedDiffFromChunksWithContext` in `unified_diff.go`: **100.0%**

Function-by-function coverage output for the current working tree now shows every covered function at **100.0%**, including:

- `apply.go`
- `types.go`
- `invocation.go`
- `parser.go`
- `seek_sequence.go`
- `verified.go`
- `unified_diff.go`
- CLI wrapper code

## Direct upstream Rust binary parity

The repository now also includes a differential oracle test:

- `TestRustUpstreamBinaryParity` in `rust_parity_test.go`

That test builds:

- the Go `cmd/apply_patch` binary
- the real upstream Rust `apply_patch` binary from `codex-upstream/codex-rs/apply-patch`

It then compares the two binaries directly on the VPS for:

- exit code parity
- combined stdout/stderr parity
- final filesystem-state parity

The current verified scope of that direct comparison is:

- **20 curated CLI/tool parity cases**
- **all 23 upstream scenario fixtures**

That means the repository is no longer relying only on ported assertions and coverage metrics. It now also has a direct binary-to-binary parity check against the upstream Rust implementation.

## What changed beyond `00955ff`

The final delta beyond `00955ff` is small and focused.

### 1. Final unified-diff test seam

`unified_diff.go` now exposes three internal-only function variables:

- `mkdirTemp`
- `writeFile`
- `execCommand`

These are test seams only. They preserve runtime behavior while allowing deterministic coverage of temp-directory, temp-file, and diff-execution failure paths.

### 2. Final unified-diff failure-path coverage

`unified_diff_test.go` now includes the final two deterministic temp-file write failure tests:

- `TestUnifiedDiffFailsWritingTempOldFile`
- `TestUnifiedDiffFailsWritingTempNewFile`

Those tests are green and are what closed the remaining uncovered branches in `UnifiedDiffFromChunksWithContext`.

### 3. Small additional branch coverage improvements

Two other small targeted tests were also added:

- `TestParseOneHunkUpdateChunkErrorPropagates` in `parser_internal_test.go`
- `TestMaybeParseApplyPatchVerifiedUpdateCorrectnessErrorOnDiffFailure` in `verified_test.go`

## Practical meaning of the current status

The port is no longer merely “close” to full parity coverage.

The current working tree is now:

- full-suite green
- **100.0%** statement-covered
- fully covered in `unified_diff.go`
- still backed by the upstream scenario corpus
- directly oracle-tested against the upstream Rust binary
- still aligned to the exact-behavior porting goal rather than a loose approximation

The major known engine blocker from earlier work remains fixed, and the final coverage blocker has now also been eliminated.

## Exact next steps

The core completion bar for this port is now met on the current VPS working tree:

- full Go suite green
- **100.0%** statement coverage
- direct binary parity against the upstream Rust implementation across curated CLI cases and the full upstream scenario corpus

Any further work from here is optional hardening rather than a known missing parity requirement.

## Bottom line

This is an exact-behavior porting effort, not a loose reimplementation.

The Go port has now reached a verified state where the live VPS working tree is green under `go test ./...`, measures **100.0%** statement coverage, and passes a direct binary-to-binary parity test against the upstream Rust implementation across curated CLI cases and all upstream scenario fixtures.
