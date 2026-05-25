# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`AGENTS.md` is the canonical short-form contributor guide (commands, style, PR rules) and should be read first. This file adds architectural context that requires reading several files to piece together.

## Commands

All everyday tasks go through `Taskfile.yml`. The most useful targets:

- `task run` — run the gateway from `cmd/gateway/main.go`
- `task build` — produce `bin/inference-gateway`
- `task test` — `go test -v ./...`
- `task benchmark` — benchmarks under `./tests/...`
- `task generate` — regenerate everything from `openapi.yaml` + `mcp/mcp-schema.yaml` (see "Code generation" below)
- `task format` — `prettier --write .` then `go fmt ./...`
- `task lint` — `golangci-lint run` and `markdownlint`
- `task openapi:lint` — Spectral lint of `openapi.yaml`
- `task pre-commit:install` — symlinks `scripts/pre-commit-check.sh` into `.git/hooks/pre-commit`; the hook runs generate + format + lint + build + test, and CI mirrors it

Running a single test: `go test -v -run TestName ./path/to/pkg`. Toolchain is pinned in `.flox/env/manifest.toml` (Go 1.26.2, golangci-lint, mockgen, Spectral, kubectl, helm, etc.); `flox activate` brings it all in.

## Architecture

### Request pipeline

`cmd/gateway/main.go` wires everything in this order: load `config.Config` from env → init logger → init OpenTelemetry (separate metrics server on `:9464` when `TELEMETRY_ENABLE=true`) → build provider registry and HTTP client → optionally init MCP client/agent/middleware → register Gin handlers.

Routes (`api/routes.go`):

- `GET  /health`
- `GET  /v1/models`
- `GET  /v1/mcp/tools`
- `POST /v1/chat/completions` — the main inference endpoint
- `ANY  /proxy/:provider/*path` — raw passthrough that bypasses middleware

Middleware chain (registered in `main.go`, defined in `api/middlewares/`): logger → telemetry (if enabled) → OIDC auth (if enabled) → MCP (if enabled). The MCP middleware inspects responses for tool calls and re-invokes the upstream provider with the tool results; to prevent loops, its internal follow-up requests set `X-MCP-Bypass: true`. Clients can set the same header to opt out. `/proxy/...` skips middleware entirely.

### Provider abstraction

A "provider" is one upstream LLM API (OpenAI, Anthropic, Groq, Ollama, etc.). The runtime pieces live in `providers/`:

- `core/` — `Provider` interface and base implementation
- `client/` — shared HTTP client config (`client.go` is generated)
- `registry/` — `ProviderRegistry.BuildProvider(id, client)` constructs a provider on demand from `cfg.Providers` (`registry.go` is generated)
- `transformers/` — per-provider request/response transformers (most are generated; the ones in `.openapi-ignore` are hand-written)
- `routing/model_mapping.go` — maps a model string like `openai/gpt-4` to a provider; if the model has no prefix, the request must include `?provider=...`
- `constants/`, `types/` — generated identifiers and OpenAPI-derived types

Provider detection is therefore: explicit `?provider=` query param wins, otherwise the prefix in `model` is parsed.

### Code generation

`openapi.yaml` and `mcp/mcp-schema.yaml` are the source of truth. `task generate` runs `cmd/generate/main.go` (a thin CLI over `internal/codegen`, `internal/dockergen`, `internal/kubegen`, `internal/mdgen`) repeatedly with different `-type` flags, then `oapi-codegen`, then `bin/generator` (MCP JSON-RPC types), then `go generate ./...` (mockgen). It regenerates: provider client + registry + transformers + constants + types, `config/config.go`, `Configurations.md`, the Helm `secrets-defaults.yaml` / `configmap-defaults.yaml` / `values.yaml`, and every `examples/docker-compose/*/.env.example`.

Hand-edits to any of those will be clobbered on the next run. Custom provider implementations are protected by `.openapi-ignore` (currently `providers/anthropic.go`, `cohere.go`, `cloudflare.go`, `ollama.go`, `openai.go`, `deepseek.go`, `groq.go`) — note these paths refer to legacy locations; the actively edited transformers under `providers/transformers/` use the `// Code generated from OpenAPI schema. DO NOT EDIT.` header to mark generated files. When adding a new provider, edit `openapi.yaml`'s `Provider` enum + `x-provider-configs` and run `task generate` — see `CONTRIBUTING.md` for the full flow.

### Configuration

`config/config.go` is generated; `config/meta.go` is hand-written. Loading uses `sethvargo/go-envconfig` with `envconfig.OsLookuper()`. Every supported env var is enumerated in the auto-generated `Configurations.md` — link users there rather than describing variables inline.

### MCP

`mcp/client.go` connects to one or more MCP servers listed in `MCP_SERVERS` (comma-separated). `mcp/agent.go` orchestrates the tool-call loop. `mcp/generated_types.go` is regenerated from `mcp-schema.yaml`. The middleware caps follow-up iterations at 10 and supports background reconnection when `MCP_ENABLE_RECONNECT=true`.

## Conventions worth remembering

- Conventional Commits are enforced by semantic-release (`.releaserc.yaml`); CI relies on them to compute the next version.
- Mocks live under `tests/mocks/` and are produced by `//go:generate mockgen ...` directives (see top of `api/routes.go`). They regenerate with `task generate`.
- Provider IDs must be lowercase valid Go identifiers (`openai`, `deepseek`, `newai`).
- `golangci-lint` enables `gocritic`, `gofmt`, `goimports`; run `task lint` before pushing.
