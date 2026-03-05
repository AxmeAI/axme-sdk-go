# axme-sdk-go

**Official Go SDK for the AXME platform.** Send and manage intents, observe lifecycle events, handle approvals and inbox operations, and access the full enterprise admin surface — idiomatic Go, context-aware, no dependencies beyond the standard library.

> **Alpha** · API surface is stabilizing. Not recommended for production workloads yet.  
> Bug reports, feedback, and alpha access → [hello@axme.ai](mailto:hello@axme.ai)

---

## What You Can Do With This SDK

- **Send intents** — create typed, durable actions with delivery guarantees
- **Observe lifecycle** — stream real-time state events
- **Approve or reject** — handle human-in-the-loop steps from Go services
- **Control workflows** — pause, resume, cancel, update retry policies and reminders
- **Administer** — manage organizations, workspaces, service accounts, and grants

---

## Install

```bash
go get github.com/AxmeAI/axme-sdk-go
```

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
        BaseURL: "https://gateway.axme.ai",
        APIKey:  "YOUR_API_KEY",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Check connectivity
    health, err := client.Health(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(health)

    // Send an intent
    intent, err := client.CreateIntent(ctx, map[string]any{
        "intent_type":  "order.fulfillment.v1",
        "payload":      map[string]any{"order_id": "ord_123", "priority": "high"},
        "owner_agent":  "agent://fulfillment-service",
    }, axme.RequestOptions{IdempotencyKey: "fulfill-ord-123-001"})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(intent["intent_id"], intent["status"])
}
```

---

## API Method Families

The SDK covers the full public API surface:

![API Method Family Map](docs/diagrams/01-api-method-family-map.svg)

*D1 families (intents, inbox, approvals) are the core integration path. D2 adds schemas, webhooks, and media. D3 covers enterprise admin. The Go SDK implements all three tiers.*

---

## Error Model and Retriability

The Go SDK maps platform error codes to typed errors. Use the error model to decide whether to retry:

![Error Model and Retriability](docs/diagrams/02-error-model-retriability.svg)

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

## Approvals

```go
inbox, err := client.ListInbox(ctx, map[string]any{
    "owner_agent": "agent://manager",
    "status":      "PENDING",
}, axme.RequestOptions{})

for _, item := range inbox["items"].([]any) {
    entry := item.(map[string]any)
    _, err = client.ResolveApproval(ctx, entry["intent_id"].(string), map[string]any{
        "decision": "approved",
        "note":     "LGTM",
    }, axme.RequestOptions{})
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
key, _ := client.CreateServiceAccountKey(ctx, sa["id"].(string), axme.RequestOptions{})

// List all service accounts
list, _ := client.ListServiceAccounts(ctx, "org_abc", axme.RequestOptions{})

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

## Repository Structure

```
axme-sdk-go/
├── axme/
│   ├── client.go              # AxmeClient — all API methods
│   └── config.go              # ClientConfig and RequestOptions
└── docs/
    └── diagrams/              # Diagram copies for README embedding
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
- Alpha program access and integration questions: [hello@axme.ai](mailto:hello@axme.ai)
- Security disclosures: see [SECURITY.md](SECURITY.md)
- Contribution guidelines: [CONTRIBUTING.md](CONTRIBUTING.md)
