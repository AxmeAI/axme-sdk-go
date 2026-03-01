package axme

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterNick(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/users/register-nick" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "register-1" {
			t.Fatalf("unexpected idempotency header: %s", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["nick"] != "@partner.user" {
			t.Fatalf("unexpected nick: %v", body["nick"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":          true,
			"user_id":     "11111111-1111-4111-8111-111111111111",
			"owner_agent": "agent://user/1",
			"nick":        "@partner.user",
		})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	response, err := client.RegisterNick(
		context.Background(),
		map[string]any{"nick": "@partner.user", "display_name": "Partner User"},
		RequestOptions{IdempotencyKey: "register-1"},
	)
	if err != nil {
		t.Fatalf("register nick failed: %v", err)
	}
	if response["ok"] != true {
		t.Fatalf("unexpected response: %v", response)
	}
}

func TestCheckNick(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/users/check-nick" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("nick"); got != "@partner.user" {
			t.Fatalf("unexpected nick query: %s", got)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":             true,
			"nick":           "@partner.user",
			"normalized_nick": "partner.user",
			"public_address": "partner.user@ax",
			"available":      true,
		})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	response, err := client.CheckNick(context.Background(), "@partner.user", RequestOptions{})
	if err != nil {
		t.Fatalf("check nick failed: %v", err)
	}
	if response["available"] != true {
		t.Fatalf("unexpected response: %v", response)
	}
}

func TestRenameNick(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/users/rename-nick" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "rename-1" {
			t.Fatalf("unexpected idempotency header: %s", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["owner_agent"] != "agent://user/1" {
			t.Fatalf("unexpected owner_agent: %v", body["owner_agent"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":          true,
			"user_id":     "11111111-1111-4111-8111-111111111111",
			"owner_agent": "agent://user/1",
			"nick":        "@partner.new",
		})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	response, err := client.RenameNick(
		context.Background(),
		map[string]any{"owner_agent": "agent://user/1", "nick": "@partner.new"},
		RequestOptions{IdempotencyKey: "rename-1"},
	)
	if err != nil {
		t.Fatalf("rename nick failed: %v", err)
	}
	if response["nick"] != "@partner.new" {
		t.Fatalf("unexpected response: %v", response)
	}
}

func TestGetUserProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/users/profile" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("owner_agent"); got != "agent://user/1" {
			t.Fatalf("unexpected owner_agent query: %s", got)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":          true,
			"user_id":     "11111111-1111-4111-8111-111111111111",
			"owner_agent": "agent://user/1",
			"nick":        "@partner.user",
		})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	response, err := client.GetUserProfile(context.Background(), "agent://user/1", RequestOptions{})
	if err != nil {
		t.Fatalf("get profile failed: %v", err)
	}
	if response["owner_agent"] != "agent://user/1" {
		t.Fatalf("unexpected response: %v", response)
	}
}

func TestUpdateUserProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/users/profile/update" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "profile-1" {
			t.Fatalf("unexpected idempotency header: %s", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["display_name"] != "Partner User Updated" {
			t.Fatalf("unexpected display_name: %v", body["display_name"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":           true,
			"user_id":      "11111111-1111-4111-8111-111111111111",
			"owner_agent":  "agent://user/1",
			"display_name": "Partner User Updated",
		})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	response, err := client.UpdateUserProfile(
		context.Background(),
		map[string]any{"owner_agent": "agent://user/1", "display_name": "Partner User Updated"},
		RequestOptions{IdempotencyKey: "profile-1"},
	)
	if err != nil {
		t.Fatalf("update profile failed: %v", err)
	}
	if response["display_name"] != "Partner User Updated" {
		t.Fatalf("unexpected response: %v", response)
	}
}
