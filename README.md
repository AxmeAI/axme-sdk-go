# axme-sdk-go

**Go SDK for AXME** - send intents, poll for deliveries, resume workflows. Context-aware, idiomatic Go, no dependencies beyond the standard library.

[![Alpha](https://img.shields.io/badge/status-alpha-orange)](https://cloud.axme.ai/alpha/cli) [![Go Reference](https://pkg.go.dev/badge/github.com/AxmeAI/axme-sdk-go/axme.svg)](https://pkg.go.dev/github.com/AxmeAI/axme-sdk-go/axme) [![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

**[Quick Start](https://cloud.axme.ai/alpha/cli)** · **[Docs](https://github.com/AxmeAI/axme-docs)** · **[Examples](https://github.com/AxmeAI/axme-examples)**

---

## Install

```bash
go get github.com/AxmeAI/axme-sdk-go@latest
```

Requires Go 1.22+.

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/AxmeAI/axme-sdk-go/axme"
)

func main() {
    client, err := axme.NewClient(axme.ClientConfig{APIKey: "axme_sa_..."})
    if err != nil { log.Fatal(err) }

    ctx := context.Background()

    // Send an intent - survives crashes, retries, timeouts
    intent, err := client.CreateIntent(ctx, map[string]any{
        "intent_type": "order.fulfillment.v1",
        "to_agent":    "agent://myorg/production/fulfillment-service",
        "payload":     map[string]any{"order_id": "ord_123"},
    }, axme.RequestOptions{IdempotencyKey: "fulfill-ord-123-001"})
    if err != nil { log.Fatal(err) }

    fmt.Println(intent["intent_id"], intent["status"])
}
```

---

## Human Approvals

```go
result, err := client.CreateIntent(ctx, map[string]any{
    "intent_type": "intent.budget.approval.v1",
    "to_agent":    "agent://myorg/prod/agent_core",
    "payload":     map[string]any{"amount": 32000},
    "human_task": map[string]any{
        "task_type":        "approval",
        "notify_email":     "approver@example.com",
        "allowed_outcomes": []string{"approved", "rejected"},
    },
}, axme.RequestOptions{})
```

8 task types: `approval`, `confirmation`, `review`, `assignment`, `form`, `clarification`, `manual_action`, `override`. Full reference: [axme-docs](https://github.com/AxmeAI/axme-docs).

---

## Observe Lifecycle Events

```go
events, err := client.ListIntentEvents(ctx, intentID, nil, axme.RequestOptions{})
```

---

## Agent Mesh - Monitor and Govern

Agent Mesh gives every agent real-time health monitoring, policy enforcement, and a kill switch - all from a single dashboard.

```go
client.Mesh.StartHeartbeat(ctx)
client.Mesh.ReportMetric(ctx, axme.Metric{Success: true, LatencyMs: 230, CostUSD: 0.02})
```

Set action policies (allowlist/denylist intent types) and cost policies (intents/day, $/day limits) per agent via dashboard or API. Mesh module coming soon to this SDK - [Python SDK](https://github.com/AxmeAI/axme-sdk-python) available now. [Full overview](https://github.com/AxmeAI/axme#agent-mesh---see-and-control-your-agents).

Open the live dashboard at [mesh.axme.ai](https://mesh.axme.ai) or run `axme mesh dashboard` from the CLI.

---

## Examples

```bash
AXME_API_KEY="axme_sa_..." go run ./examples/basic_submit.go
```

More: [axme-examples](https://github.com/AxmeAI/axme-examples)

---

## Development

```bash
go test ./...
```

---

## Related

| | |
|---|---|
| [axme-docs](https://github.com/AxmeAI/axme-docs) | API reference and integration guides |
| [axme-examples](https://github.com/AxmeAI/axme-examples) | Runnable examples |
| [axp-spec](https://github.com/AxmeAI/axp-spec) | Protocol specification |
| [axme-cli](https://github.com/AxmeAI/axme-cli) | CLI tool |
| [axme-conformance](https://github.com/AxmeAI/axme-conformance) | Conformance suite |

---

[hello@axme.ai](mailto:hello@axme.ai) · [Security](SECURITY.md) · [License](LICENSE)
