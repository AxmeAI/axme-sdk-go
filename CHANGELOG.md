# Changelog

## 0.1.2 (2026-03-18)

### Features
- `Listen()` — channel-based SSE stream for agent intent inbox
- `ListIntentEvents()` — poll intent lifecycle events with cursor
- `ApplyScenario()` — compile and submit scenario bundle
- `ValidateScenario()` — dry-run scenario validation
- `SendIntent()` — convenience wrapper with auto-generated correlation_id
- `Observe()` — channel-based intent lifecycle event polling
- `WaitFor()` — block until intent reaches terminal state
- `Health()` — gateway health check
- `McpInitialize()` — MCP protocol handshake
- `McpListTools()` — list available MCP tools
- `McpCallTool()` — invoke MCP tool

## 0.1.1 (2026-03-08)

- Initial alpha release with AXME API coverage (70+ methods)
- Intent lifecycle, inbox, webhooks, admin APIs
- Zero external dependencies (standard library only)
