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

The repository now runs the upstream filesystem scenario corpus directly from:

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

The test suite also exercises the real built binary from `cmd/apply_patch`, not just in-process library helpers.

That coverage now includes:

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

The Go suite now includes exact unified diff shape checks for key upstream cases, including:

- first-line replacement
- last-line replacement
- insert-at-EOF
- interleaved changes
- custom context radius behavior

### 7. Public API / helper parity work

The Go package now exposes more of the upstream crate’s public helper surface, including:

- `CODEX_CORE_APPLY_PATCH_ARG1`
- `APPLY_PATCH_TOOL_INSTRUCTIONS`
- `ApplyHunks`
- `UnifiedDiffFromChunksWithContext`
- `ApplyPatchAction.Changes()`
- `NewAddForTest`

## Current status

The port is much closer to the upstream crate than it was at the start.

Large parts of the upstream behavior matrix are already covered and passing.

However, the port is **not finished yet**.

The remaining work is now down to smaller but important correctness gaps in core engine behavior and edge-case semantics.

## Current concrete blocker

The current known blocker is a real engine bug found by a newer direct library-style test based on the upstream Rust crate.

That failing case exercises a patch containing:

- a pure-addition update chunk
- followed by a later removal chunk

That case currently exposes a replacement-ordering bug in the Go patch engine in `apply.go`.

So the repository is in the following state:

- broad parity work completed
- large portions of upstream behavior already green
- still not complete until the remaining core engine edge cases are fixed and revalidated

## Working approach going forward

The porting strategy remains:

1. compare directly against the upstream Rust crate
2. port missing tests or behavior slices from upstream
3. fix the Go implementation until those cases pass
4. keep the repo green with focused verification after each change
5. continue until the Go port behaves like the Rust crate across parser, engine, CLI, verified invocation, and helper APIs

## Bottom line

This is an exact-behavior porting effort, not a loose reimplementation.

The Go port has already made substantial progress toward one-to-one parity with the Codex upstream `apply-patch` crate, and the remaining work is now focused on closing the last correctness gaps rather than building missing major subsystems.
