# axme-sdk-go

Official Go SDK for Axme APIs and workflows.

## Status

Status: Beta (best-effort).
API may change.
For production use, prefer REST/OpenAPI or Python/TS SDKs.

## Quickstart

```go
package main

import (
	"context"
	"log"

	"github.com/AxmeAI/axme-sdk-go/axme"
)

func main() {
	client, err := axme.NewClient(axme.ClientConfig{
		BaseURL: "https://gateway.example.com",
		APIKey:  "YOUR_API_KEY",
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	registered, err := client.RegisterNick(
		ctx,
		map[string]any{"nick": "@partner.user", "display_name": "Partner User"},
		axme.RequestOptions{IdempotencyKey: "nick-register-001"},
	)
	if err != nil {
		log.Fatal(err)
	}

	check, err := client.CheckNick(ctx, "@partner.user", axme.RequestOptions{})
	if err != nil {
		log.Fatal(err)
	}
	_, _ = registered, check

	renamed, err := client.RenameNick(
		ctx,
		map[string]any{"owner_agent": registered["owner_agent"], "nick": "@partner.new"},
		axme.RequestOptions{IdempotencyKey: "nick-rename-001"},
	)
	if err != nil {
		log.Fatal(err)
	}

	profile, err := client.GetUserProfile(ctx, registered["owner_agent"].(string), axme.RequestOptions{})
	if err != nil {
		log.Fatal(err)
	}

	updated, err := client.UpdateUserProfile(
		ctx,
		map[string]any{"owner_agent": profile["owner_agent"], "display_name": "Partner User Updated"},
		axme.RequestOptions{IdempotencyKey: "profile-update-001"},
	)
	if err != nil {
		log.Fatal(err)
	}

	_, _, _ = renamed, profile, updated
}
```

## Development

```bash
go test ./...
```
