# apply-patch-go

A Go port of the upstream Rust `apply_patch` crate from OpenAI Codex. It keeps the familiar patch format and CLI behavior while exposing a small Go API for embedding the same functionality in Go programs.

Original upstream crate: <https://github.com/openai/codex/tree/main/codex-rs/apply-patch>

## Install

### CLI

```bash
go install github.com/amxv/apply-patch-go/cmd/apply_patch@latest
```

Use it with an argument or stdin:

```bash
apply_patch '*** Begin Patch
*** Add File: hello.txt
+hello
*** End Patch'

echo '*** Begin Patch
*** Delete File: old.txt
*** End Patch' | apply_patch
```

### Library

```bash
go get github.com/amxv/apply-patch-go
```

```go
package main

import (
	"bytes"
	"log"

	applypatch "github.com/amxv/apply-patch-go"
)

func main() {
	patch := "*** Begin Patch\n*** Add File: hello.txt\n+hello\n*** End Patch"
	var stdout, stderr bytes.Buffer
	if err := applypatch.ApplyPatch(patch, &stdout, &stderr); err != nil {
		log.Fatal(err)
	}
}
```

## Layout

- `cmd/apply_patch`: CLI entrypoint
- `internal/parse`: parser internals
- `internal/textmatch`: internal line-matching helpers
- root package: stable user-facing API

## License

MIT. See `LICENSE`.
