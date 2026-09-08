# AGENTS.md

This directory contains the RELAI Go SDK used by Go simulators and learning
environments.

## Validation

Work from `sdk/go` and prefer the smallest relevant command while iterating:

```bash
go test ./...
```

Before handoff, run:

```bash
test "$(gofmt -l .)" = ""
go vet ./...
go test ./...
```

Keep the module path as `github.com/relai-ai/relai-cli/sdk/go`. User code
imports runtime symbols from that module and aliases the package as `relai`.
