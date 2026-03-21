# axme-sdk-go

**Official Go SDK for the AXME platform.** Send and manage intents, poll lifecycle events and history, handle approvals and inbox operations, and access the full enterprise admin surface — idiomatic Go, context-aware, no dependencies beyond the standard library.

> **Alpha** · API surface is stabilizing. Not recommended for production workloads yet.  
> **Alpha** — install CLI, log in, run your first example in under 5 minutes. [Quick Start](https://cloud.axme.ai/alpha/cli) · [hello@axme.ai](mailto:hello@axme.ai)

---

## What Is AXME?

AXME is a coordination infrastructure for durable execution of long-running intents across distributed systems.

It provides a model for executing **intents** — requests that may take minutes, hours, or longer to complete — across services, agents, and human participants.

## AXP — the Intent Protocol

At the core of AXME is **AXP (Intent Protocol)** — an open protocol that defines contracts and lifecycle rules for intent processing.

AXP can be implemented independently.  
The open part of the platform includes:

- the protocol specification and schemas
- SDKs and CLI for integration
- conformance tests
- implementation and integration documentation

## AXME Cloud

**AXME Cloud** is the managed service that runs AXP in production together with **The Registry** (identity and routing).

It removes operational complexity by providing:

- reliable intent delivery and retries  
- lifecycle management for long-running operations  
- handling of timeouts, waits, reminders, and escalation  
- observability of intent status and execution history  

State and events can be accessed through:

- API and SDKs  
- event streams and webhooks  
- the cloud console

---

## What You Can Do With This SDK

- **Send intents** — create typed, durable actions with delivery guarantees
- **Poll lifecycle events** — retrieve real-time state events and intent history via `ListIntentEvents`
- **Approve or reject** — handle human-in-the-loop steps from Go services
- **Control workflows** — pause, resume, cancel, update retry policies and reminders
- **Administer** — manage organizations, workspaces, service accounts, and grants

---

## Install

```bash
go get github.com/AxmeAI/axme-sdk-go@latest
```

Go modules are published by git tag and module path (no separate central package name). The import package remains `axme`.

---

## Quickstart

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/AxmeAI/axme-sdk-go/axme"
)

func main() {
    client, err := axme.NewClient(axme.ClientConfig{
        APIKey:  "AXME_API_KEY",  // sent as x-api-key
        ActorToken: "OPTIONAL_USER_OR_SESSION_TOKEN", // sent as Authorization: Bearer
        // Optional override (defaults to https://api.cloud.axme.ai):
        // BaseURL: "https://staging-api.cloud.axme.ai",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Check connectivity / discover available capabilities
    capabilities, err := client.GetCapabilities(ctx, axme.RequestOptions{})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(capabilities)

    // Send an intent to a registered agent address
    intent, err := client.CreateIntent(ctx, map[string]any{
        "intent_type": "order.fulfillment.v1",
        "to_agent":    "agent://acme-corp/production/fulfillment-service",
        "payload":     map[string]any{"order_id": "ord_123", "priority": "high"},
    }, axme.RequestOptions{IdempotencyKey: "fulfill-ord-123-001"})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(intent["intent_id"], intent["status"])

    // List registered agent addresses
    agents, err := client.ListAgents(ctx, "acme-corp-uuid", "prod-ws-uuid", nil, axme.RequestOptions{})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(agents["agents"])
}
```

---

## Minimal Language-Native Example

Short basic submit/get example:

- [`examples/basic_submit.go`](examples/basic_submit.go)

Run:

```bash
AXME_API_KEY="axme_sa_..." go run ./examples/basic_submit.go
```

Full runnable scenario set lives in:

- Cloud: <https://github.com/AxmeAI/axme-examples/tree/main/cloud>
- Protocol-only: <https://github.com/AxmeAI/axme-examples/tree/main/protocol>

---

## API Method Families

The SDK covers the full public API surface:

![API Method Family Map](https://raw.githubusercontent.com/AxmeAI/axme-docs/main/docs/diagrams/api/01-api-method-family-map.svg)

*D1 families (intents, inbox, approvals) are the core integration path. D2 adds schemas, webhooks, and media. D3 covers enterprise admin. The Go SDK implements all three tiers.*

---

## Error Model and Retriability

The Go SDK maps platform error codes to typed errors. Use the error model to decide whether to retry:

![Error Model and Retriability](https://raw.githubusercontent.com/AxmeAI/axme-docs/main/docs/diagrams/api/02-error-model-retriability.svg)

*`4xx` client errors are wrapped in `AxmeClientError` — do not retry. `5xx` errors are `AxmeServerError` — safe to retry with the original idempotency key. The `RetryAfter` field provides the wait hint.*

```go
intent, err := client.CreateIntent(ctx, payload, opts)
if err != nil {
    var apiErr *axme.AxmeAPIError
    if errors.As(err, &apiErr) && apiErr.Retriable {
        time.Sleep(apiErr.RetryAfter)
        // retry...
    }
}
```

---

## Human-in-the-Loop (8 Task Types)

AXME supports 8 human task types. Each pauses the workflow and notifies a human via email with a link to a web task page.

| Task type | Use case | Default outcomes |
|-----------|----------|-----------------|
| `approval` | Approve or reject a request | approved, rejected |
| `confirmation` | Confirm a real-world action completed | confirmed, denied |
| `review` | Review content with multiple outcomes | approved, changes_requested, rejected |
| `assignment` | Assign work to a person or team | assigned, declined |
| `form` | Collect structured data via form fields | submitted |
| `clarification` | Request clarification (comment required) | provided, declined |
| `manual_action` | Physical task completion (evidence required) | completed, failed |
| `override` | Override a policy gate (comment required) | override_approved, rejected |

```go
// Create an intent with a human task step
result, err := client.CreateIntent(ctx, axme.CreateIntentParams{
    IntentType: "intent.budget.approval.v1",
    ToAgent:    "agent://agent_core",
    Payload:    map[string]any{"amount": 32000, "department": "engineering"},
    HumanTask: &axme.HumanTask{
        Title:           "Approve Q3 budget",
        Description:     "Review and approve the Q3 infrastructure budget.",
        TaskType:        "approval",
        NotifyEmail:     "approver@example.com",
        AllowedOutcomes: []string{"approved", "rejected"},
    },
})
```

Task types with forms use `form_schema` to define required fields:

```go
HumanTask: &axme.HumanTask{
    Title:       "Assign incident commander",
    TaskType:    "assignment",
    NotifyEmail: "oncall@example.com",
    FormSchema: map[string]any{
        "type":     "object",
        "required": []string{"assignee"},
        "properties": map[string]any{
            "assignee": map[string]any{"type": "string", "title": "Commander name"},
            "priority": map[string]any{"type": "string", "enum": []string{"P1", "P2", "P3"}},
        },
    },
},
```

### Programmatic approvals (inbox API)

```go
inbox, err := client.ListInbox(ctx, "agent://manager", axme.RequestOptions{})
if err != nil {
    log.Fatal(err)
}

items, _ := inbox["items"].([]any)
for _, item := range items {
    entry, ok := item.(map[string]any)
    if !ok {
        continue
    }
    threadID, ok := entry["thread_id"].(string)
    if !ok || threadID == "" {
        continue
    }
    _, err = client.ApproveInboxThread(
        ctx,
        threadID,
        map[string]any{"note": "LGTM"},
        "agent://manager",
        axme.RequestOptions{},
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

---

## Enterprise Admin APIs

The Go SDK includes the full service-account lifecycle surface:

```go
// Create a service account
sa, _ := client.CreateServiceAccount(ctx, map[string]any{
    "name": "ci-runner",
    "org_id": "org_abc",
}, axme.RequestOptions{IdempotencyKey: "sa-ci-runner-001"})

// Issue a key
key, _ := client.CreateServiceAccountKey(ctx, sa["id"].(string), map[string]any{}, axme.RequestOptions{})

// List all service accounts
list, _ := client.ListServiceAccounts(ctx, "org_abc", "", axme.RequestOptions{})

// Revoke a key
client.RevokeServiceAccountKey(ctx, sa["id"].(string), key["key_id"].(string), axme.RequestOptions{})
```

Available methods:
- `CreateServiceAccount` / `ListServiceAccounts` / `GetServiceAccount`
- `CreateServiceAccountKey` / `RevokeServiceAccountKey`

---

## Nick and Identity Registry

```go
// Register a user identity
registered, _ := client.RegisterNick(ctx,
    map[string]any{"nick": "@partner.user", "display_name": "Partner User"},
    axme.RequestOptions{IdempotencyKey: "nick-register-001"},
)

// Check existence
check, _ := client.CheckNick(ctx, "@partner.user", axme.RequestOptions{})

// Rename
renamed, _ := client.RenameNick(ctx,
    map[string]any{"owner_agent": registered["owner_agent"], "nick": "@partner.new"},
    axme.RequestOptions{IdempotencyKey: "nick-rename-001"},
)
```

---

## MCP (Model Context Protocol)

The Go SDK includes a built-in MCP endpoint client for gateway-hosted MCP sessions.

```go
// Initialize an MCP session
init, err := client.McpInitialize(ctx, axme.RequestOptions{})
fmt.Println(init["serverInfo"])

// List available tools
tools, err := client.McpListTools(ctx, axme.RequestOptions{})
for _, tool := range tools["tools"].([]any) {
    t := tool.(map[string]any)
    fmt.Println(t["name"])
}

// Call a tool
result, err := client.McpCallTool(ctx, "create_intent", axme.McpCallToolOptions{
    Arguments: map[string]any{
        "intent_type":  "order.fulfillment.v1",
        "payload":      map[string]any{"order_id": "ord_123"},
        "owner_agent":  "agent://fulfillment-service",
    },
})
fmt.Println(result)
```

By default the SDK posts to `/mcp`. Override with `McpEndpointPath` in client options.

---

## Repository Structure

```
axme-sdk-go/
├── axme/
│   ├── client.go              # AxmeClient — all API methods
│   └── config.go              # ClientConfig and RequestOptions
├── examples/
│   └── basic_submit.go        # Minimal language-native quickstart
└── docs/
```

---

## Tests

```bash
go test ./...
```

---

## Related Repositories

| Repository | Role |
|---|---|
| [axme-docs](https://github.com/AxmeAI/axme-docs) | Full API reference and integration guides |
| [axme-spec](https://github.com/AxmeAI/axme-spec) | Schema contracts this SDK implements |
| [axme-conformance](https://github.com/AxmeAI/axme-conformance) | Conformance suite that validates this SDK |
| [axme-examples](https://github.com/AxmeAI/axme-examples) | Runnable examples using this SDK |
| [axme-cli](https://github.com/AxmeAI/axme-cli) | CLI tool built on top of this SDK |
| [axme-sdk-python](https://github.com/AxmeAI/axme-sdk-python) | Python equivalent |

---

## Contributing & Contact

- Bug reports and feature requests: open an issue in this repository
- Quick Start: https://cloud.axme.ai/alpha/cli · Contact: [hello@axme.ai](mailto:hello@axme.ai)
- Security disclosures: see [SECURITY.md](SECURITY.md)
- Contribution guidelines: [CONTRIBUTING.md](CONTRIBUTING.md)
