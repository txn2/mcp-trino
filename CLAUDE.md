# CLAUDE.md

This file provides guidance to Claude Code when working with this project.

## Project Overview

**mcp-trino** is a generic, open-source MCP (Model Context Protocol) server for Trino. It enables AI assistants to query and explore data warehouses via the MCP protocol.

**Key Design Goals:**
- Composable: Can be used standalone OR imported as a library
- Generic: No domain-specific logic; suitable for any Trino deployment
- Secure: Configurable limits and timeouts to prevent abuse

## CRITICAL - Factual Integrity (No Confabulation)

AI-generated prose (PR descriptions, commit messages, reviews, explanations) is held to the same verification standard as code. Unverified claims are as unacceptable as untested code.

1. **Never assert facts you haven't verified.** Before stating that a file contains X, a config is missing Y, or a system behaves in way Z — READ the file, CHECK the config, VERIFY the behavior. If you haven't looked, say "I haven't verified this" or say nothing.

2. **Every claim must be evidence-linked.** PR descriptions, commit messages, and work summaries may only include claims that are either: (a) directly visible in the diff, or (b) verified by reading a specific file (cite file:line). No exceptions.

3. **Never pad or embellish.** If you made two fixes, describe two fixes. Do not invent a third to make the work look more complete. Do not present hypotheses as confirmed diagnoses.

4. **Uncertainty must be explicit.** Use "I believe," "possibly," or "I haven't verified" when uncertain. Never upgrade a guess to a fact.

5. **When reviewing, verify claims against evidence.** Treat PR descriptions and commit messages as claims to be fact-checked, not trusted context.

6. **Omission over fabrication.** A gap stated honestly is better than a fabricated answer stated confidently. When in doubt, leave it out.

## Order of Work (AI contributions)

Every change follows this order. `make verify` is the LAST step before handing work to the human — not the first, and not a substitute for review.

1. **Implement**, with acceptance criteria stated before the code is written.
2. **Adversarial review**: spawn a `general-purpose` sub-agent to review the working tree adversarially, using the prompt in `~/.claude/hooks/review-prompt-template.md`. Do not write the review yourself. Address every finding by fixing it, disputing it in writing, or deferring it with a TODO.
3. **`make verify`** on the exact tree that will be handed over. It writes `.claude/.last-verify-passed`, the sentinel the pre-commit gate checks; a commit whose diff does not match that sentinel is denied.
4. **Hand to the human for review.** Report what passed and what did not, verbatim. A human commits; Claude does not.

Announcing work as ready without steps 2 and 3 is the failure mode this section exists to prevent.

## Code Standards

1. **Idiomatic Go**: All code must follow idiomatic Go patterns and conventions. Use `gofmt`, follow Effective Go guidelines, and adhere to Go Code Review Comments.

2. **Test Coverage**: Project must maintain >=82% total unit test coverage (`COVERAGE_MIN` in the Makefile, matched by `codecov.yml`), and changed lines must reach >=80% (`PATCH_COVERAGE_MIN`). Build mocks where necessary. Use table-driven tests where appropriate.
   - **CRITICAL — verify coverage before declaring done**: run `make test`, then for EVERY new function run `go tool cover -func=coverage.out | grep <function_name>`. If any new function is below 80% (or 0.0%), add tests before saying the work is complete. This is blocking.
   - `scripts/patch-coverage.sh` covers untracked files explicitly, because `git diff` cannot see them and this project runs `make verify` before anything is committed — without that step a brand-new file would be absent from the report while the gate printed PASS.

3. **Testing Definition**: When asked to "test" or "testing" the code, this means running `make verify`, which executes the CI-equivalent suite:
   - **Tools-check (parity gate)** — verifies local `golangci-lint` and `gosec` equal `GOLANGCI_LINT_VERSION` and `GOSEC_VERSION` in the Makefile, which mirror `.github/workflows/ci.yml`. A local scanner that drifts from CI's is the most insidious parity gap: a newer local gosec can drop a rule CI still enforces, so the diff ships green locally and red on the PR. Override with `TOOLS_CHECK_STRICT=0` only with a stated reason.
   - `go mod tidy` + `go mod verify`, then `gofmt`/`goimports` — the two steps that rewrite the tree, run first and in order
   - Unit tests with race detection, `-shuffle=on -count=1`, writing the coverage profile
   - **Total coverage** must be >=82% (`COVERAGE_MIN`, mirrors `codecov.yml`'s project target) — hard gate
   - **Patch coverage** — changed lines vs `main` must be >=80% (`PATCH_COVERAGE_MIN`, mirrors codecov's patch target), via `scripts/patch-coverage.sh`
   - Linting (`golangci-lint run` + `go vet`)
   - Security (`gosec -quiet ./...` + `govulncheck`, whose report is judged against `.govulncheck-allow.txt` by `scripts/govulncheck-gate.py`: an advisory with no patched release may be accepted with a written reason, and the gate fails when an accepted advisory gains a fix or stops being reported)
   - Dead code analysis (advisory — exported library API is an expected false positive here)
   - `go build ./...` + `go mod verify`, and a GoReleaser dry-run
   - All checks must pass locally before considering code "tested"

   Semgrep and CodeQL are deliberately **not** in `make verify`. The rule for that target is CI-equivalence, and `.github/workflows/ci.yml` runs neither; `codeql.yml` runs CodeQL on the pull request where it blocks the merge. Both are available on demand (`make semgrep`, `make codeql`), and CodeQL plus mutation testing run in `make verify-release` before a tag.

   Suppressing a gosec finding requires `#nosec <RuleID> -- justification`. A `//nolint:gosec` comment silences golangci-lint's copy of gosec but NOT the standalone `gosec ./...` that `make security` and CI run, so it reads as a suppression while the finding still fails the build.

4. **Human Review Required**: A human must review and approve every line of code before it is committed. Therefore, commits are always performed by a human, not by Claude.

5. **Go Report Card**: The project MUST always maintain 100% across all categories on [Go Report Card](https://goreportcard.com/): `gofmt`, `go vet`, `golint`, `ineffassign`, `license`, and `misspell`.

6. **Cyclomatic complexity**: enforced by golangci-lint's `gocyclo` linter at `min-complexity: 20` (`.golangci.yml`), which excludes `_test.go`. That config is the enforced rule; a bare `gocyclo -over 15 .` run reports test helpers the linter deliberately ignores and is not a gate.

7. **Diagrams**: Use Mermaid for all diagrams. Never use ASCII art.

8. **Pinned dependencies**: GitHub Actions are pinned by commit SHA with a version comment; Go modules are pinned via `go.sum` and the toolchain by the `toolchain` directive in `go.mod`; `golangci-lint`, `gosec`, and `govulncheck` are pinned in the Makefile, mirrored in `.github/workflows/ci.yml`, and compared against the installed binaries by `make tools-check`. Do not add a pin that nothing compares — a version constant no gate reads gives the appearance of a guarantee without one.

## AI Verification Requirements

When AI (Claude Code or similar) contributes code, these additional checks apply:

1. **No tautological tests**: a test that sets `x.Field = "value"` then asserts `x.Field == "value"` tests the Go compiler, not the application. Delete such tests on sight. Ask of every new test: would it still pass if the production code returned a hardcoded value?

2. **Integration tests for cross-component behavior**: unit tests are not sufficient for anything spanning middleware, interceptors, transformers, or the MCP wire format. Wire up a real `mcp.Server` with `mcp.NewInMemoryTransports()`, call the tool through a client session, and assert on what the client actually receives. A unit test that hand-constructs the correct input does not prove the pieces connect.

3. **Advertised schemas are a wire contract**: `pkg/tools` is imported as a library. Input and output schemas, tool names, and structured-output field names are shipped API. Changing one is a breaking change for every consumer and for hosts that compose this toolkit.

4. **Dead-code audit**: run `make dead-code` before submitting. It is advisory, not a gate. `deadcode` walks from `main`, so for a library it reports the exported API — and every unexported function reachable only through that API — as unreachable. `deepCopySchema` in `pkg/tools/output_schemas.go` is reported and is live, called by the exported `DefaultOutputSchema`. Only delete or relocate a function that is unexported **and** unreachable from any exported entry point; check the callers before acting on a line of this output.

5. **Mutation survival review**: after adding tests, `make mutate` reports surviving mutants. Address survivors in security-relevant paths; informational ones may be deferred.

6. **No vaporware**: every package under `pkg/` must be imported by at least one non-test file, and every interface with a noop implementation must also have a real one. Do not create packages, options, or interfaces "for future use" — code that isn't wired into anything is dead code regardless of whether it has its own unit tests.

7. **Dependency-first verification**: before building on an external capability (an SDK behavior, a Trino feature), verify the dependency actually supports it. If it doesn't, surface the gap immediately instead of building scaffolding around it.

## Project Structure

```
mcp-trino/
├── cmd/mcp-trino/main.go      # Standalone server entrypoint
├── pkg/                        # PUBLIC API (importable by other projects)
│   ├── client/                 # Trino client wrapper
│   │   ├── client.go           # Connection and query execution
│   │   └── config.go           # Configuration from env/struct
│   ├── tools/                  # MCP tool definitions
│   │   ├── toolkit.go          # NewToolkit(), Register(), RegisterAll()
│   │   ├── names.go            # ToolName constants and AllTools()
│   │   ├── options.go          # Toolkit and per-registration options
│   │   ├── outputs.go          # Structured output types
│   │   ├── output_schemas.go   # Advertised JSON Schema per tool
│   │   ├── query.go            # trino_query
│   │   ├── execute.go          # trino_execute
│   │   ├── explain.go          # trino_explain
│   │   ├── browse.go           # trino_browse
│   │   ├── schema.go           # trino_describe_table
│   │   └── connections.go      # trino_list_connections
│   ├── semantic/               # Semantic metadata providers (optional)
│   ├── multiserver/            # Multi-connection manager
│   └── extensions/             # Built-in middleware/interceptors
├── internal/server/            # Default server setup (private)
├── scripts/                    # Verification gates used by `make verify`
├── go.mod
├── LICENSE                     # Apache 2.0
└── README.md
```

## Key Dependencies

- `github.com/modelcontextprotocol/go-sdk` - Official MCP SDK
- `github.com/trinodb/trino-go-client` - Trino Go driver

Versions live in `go.mod` and are not restated here; a version copied into prose
goes stale the first time Dependabot opens a PR.

## Building and Running

```bash
# Build
go build -o mcp-trino ./cmd/mcp-trino

# Run (requires Trino connection)
export TRINO_HOST=trino.example.com
export TRINO_USER=user
export TRINO_PASSWORD=pass
export TRINO_CATALOG=hive
export TRINO_SCHEMA=default
./mcp-trino
```

## Testing with Docker

```bash
# Start local Trino
docker run -d -p 8080:8080 --name trino trinodb/trino

# Configure for local testing
export TRINO_HOST=localhost
export TRINO_PORT=8080
export TRINO_USER=admin
export TRINO_SSL=false
export TRINO_CATALOG=memory
export TRINO_SCHEMA=default

./mcp-trino
```

## Composition Pattern

This package is designed to be imported by other MCP servers:

```go
import (
    "github.com/txn2/mcp-trino/pkg/client"
    "github.com/txn2/mcp-trino/pkg/tools"
)

// Create client
trinoClient, _ := client.New(client.FromEnv())

// Create toolkit and register on your server
toolkit := tools.NewToolkit(trinoClient, tools.DefaultConfig())
toolkit.RegisterAll(yourMCPServer)
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `trino_query` | Execute read-only SQL (SELECT/SHOW/DESCRIBE), returns JSON/CSV/markdown |
| `trino_execute` | Execute any SQL including write ops (INSERT/UPDATE/DELETE/CREATE/DROP) |
| `trino_explain` | Get execution plan |
| `trino_browse` | Browse catalog hierarchy (catalogs → schemas → tables) |
| `trino_describe_table` | Get column definitions |
| `trino_list_connections` | List configured Trino connections |

## Configuration Reference

Environment variables:
- `TRINO_HOST` - Server hostname
- `TRINO_PORT` - Server port
- `TRINO_USER` - Username (required)
- `TRINO_PASSWORD` - Password
- `TRINO_CATALOG` - Default catalog
- `TRINO_SCHEMA` - Default schema
- `TRINO_SSL` - Enable SSL (true/false)
- `TRINO_SSL_VERIFY` - Verify SSL certs
- `TRINO_TIMEOUT` - Query timeout in seconds
- `TRINO_SOURCE` - Client source identifier
