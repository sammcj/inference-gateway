# Repository Guidelines

## Project Structure & Module Organization

This repository contains the Go-based Inference Gateway. The main entrypoint is
`cmd/gateway/main.go`; `cmd/generate` drives code generation. HTTP routing and
middleware live in `api/`, configuration in `config/`, provider abstractions in
`providers/`, MCP support in `mcp/`, telemetry in `otel/`, and shared internal
generators/helpers in `internal/`. Tests are split between package-level
`*_test.go` files and broader tests under `tests/`. Deployment assets are in
`charts/inference-gateway/`, `Dockerfile*`, and `examples/` for Docker Compose
and Kubernetes scenarios. Generated outputs include provider types, config docs,
Helm defaults, and example `.env` files; regenerate them instead of hand-editing
when the source schema changes.

## Build, Test, and Development Commands

- `task --list`: show available Taskfile targets.
- `task run`: run the gateway from `cmd/gateway/main.go`.
- `task build`: compile `bin/inference-gateway`.
- `task test`: run `go test -v ./...`.
- `task benchmark`: run benchmarks for `./tests/...`.
- `task generate`: regenerate code from `openapi.yaml` and MCP schemas.
- `task format`: run Prettier and `go fmt ./...`.
- `task lint`: run `golangci-lint` and markdownlint.
- `task openapi:lint`: lint `openapi.yaml` with Spectral.

Use `flox activate` for the pinned development toolchain, then install hooks with
`task pre-commit:install`.

## Coding Style & Naming Conventions

Go code must be formatted by `gofmt`; `golangci-lint` also enables gofmt,
goimports, and `gocritic`. Markdown and YAML use two-space indentation,
LF endings, final newlines, and single quotes where applicable, as configured in
`.editorconfig` and `.prettierrc`. Provider identifiers should be lowercase Go
identifier-safe names, for example `openai`, `deepseek`, or `newai`.

## Testing Guidelines

Add focused Go tests close to the package under change, or use `tests/` for
cross-package behavior and gateway flows. Name test files `*_test.go`, test
functions `TestXxx`, and benchmarks `BenchmarkXxx`. Run `task test` before
opening a PR; run `task benchmark` when changing performance-sensitive routing,
provider transformation, or MCP behavior.

## Commit & Pull Request Guidelines

Commits follow Conventional Commits, as required by semantic-release. Recent
examples include `chore(deps): ...`, `ci(deps): ...`, and `chore: ...`. Keep
subjects imperative and scoped when useful.

PRs should include a clear description, linked issues when relevant, test output
or a note explaining skipped tests, and screenshots only for UI or dashboard
changes. If `openapi.yaml`, MCP schemas, provider configs, or Helm defaults
change, include the generated files from `task generate`.

## Security & Configuration Tips

Do not commit real credentials. Start from the `.env.example` files under
`examples/docker-compose/*/` and document any new environment variable in the
schema-driven configuration flow.
