package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"

	"github.com/AxmeAI/axme-sdk-go/axme"
)

func newUUID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16],
	), nil
}

func main() {
	client, err := axme.NewClient(axme.ClientConfig{
		APIKey:  os.Getenv("AXME_API_KEY"),
		BaseURL: os.Getenv("AXME_BASE_URL"),
	})
	if err != nil {
		log.Fatal(err)
	}

	correlationID, err := newUUID()
	if err != nil {
		log.Fatal(err)
	}

	created, err := client.CreateIntent(context.Background(), map[string]any{
		"intent_type":    "intent.demo.v1",
		"correlation_id": correlationID,
		"to_agent":       "agent://acme-corp/production/target",
		"payload":        map[string]any{"task": "hello-from-go"},
	}, axme.RequestOptions{})
	if err != nil {
		log.Fatal(err)
	}

	intentID, _ := created["intent_id"].(string)
	current, err := client.GetIntent(context.Background(), intentID, axme.RequestOptions{})
	if err != nil {
		log.Fatal(err)
	}

	intent, ok := current["intent"].(map[string]any)
	if !ok {
		fmt.Println("UNKNOWN")
		return
	}
	if status, ok := intent["status"].(string); ok && status != "" {
		fmt.Println(status)
		return
	}
	if lifecycleStatus, ok := intent["lifecycle_status"].(string); ok && lifecycleStatus != "" {
		fmt.Println(lifecycleStatus)
		return
	}
	fmt.Println("UNKNOWN")
}
