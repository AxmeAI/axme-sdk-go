package axme

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRegisterNick(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/users/register-nick" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "token" {
			t.Fatalf("unexpected x-api-key header: %s", got)
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

func TestClientSendsXAxmeClientHeader(t *testing.T) {
	expected := "axme-sdk-go/" + SDKVersion
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Axme-Client"); got != expected {
			t.Fatalf("unexpected X-Axme-Client header: got=%q want=%q", got, expected)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "available": true})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		BaseURL:    server.URL,
		APIKey:     "token",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := client.CheckNick(context.Background(), "@partner.user", RequestOptions{}); err != nil {
		t.Fatalf("check nick failed: %v", err)
	}
}

func TestClientSendsConfiguredActorToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "platform-token" {
			t.Fatalf("unexpected x-api-key header: %s", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer actor-token" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "available": true})
	}))
	defer server.Close()

	client, err := NewClient(
		ClientConfig{
			BaseURL:    server.URL,
			APIKey:     "platform-token",
			ActorToken: "actor-token",
			HTTPClient: server.Client(),
		},
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if _, err := client.CheckNick(context.Background(), "@partner.user", RequestOptions{}); err != nil {
		t.Fatalf("check nick failed: %v", err)
	}
}

func TestNewClientRejectsConflictingActorTokenAliases(t *testing.T) {
	_, err := NewClient(
		ClientConfig{
			BaseURL:     "https://api.axme.test",
			APIKey:      "platform-token",
			ActorToken:  "actor-a",
			BearerToken: "actor-b",
		},
	)
	if err == nil {
		t.Fatalf("expected constructor error for conflicting actor token aliases")
	}
}

func TestNewClientUsesDefaultBaseURLWhenMissing(t *testing.T) {
	client, err := NewClient(ClientConfig{APIKey: "token"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if client.baseURL != "https://api.cloud.axme.ai" {
		t.Fatalf("unexpected default base url: %s", client.baseURL)
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
			"ok":              true,
			"nick":            "@partner.user",
			"normalized_nick": "partner.user",
			"public_address":  "partner.user@ax",
			"available":       true,
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

func TestResolveIntentSupportsOwnerScopeAndControlHeaders(t *testing.T) {
	intentID := "22222222-2222-4222-8222-222222222222"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/intents/"+intentID+"/resolve" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("owner_agent"); got != "agent://owner" {
			t.Fatalf("unexpected owner_agent query: %s", got)
		}
		if got := r.Header.Get("x-owner-agent"); got != "agent://owner" {
			t.Fatalf("unexpected x-owner-agent header: %s", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer scoped-token" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["expected_policy_generation"] != float64(3) {
			t.Fatalf("unexpected expected_policy_generation: %v", body["expected_policy_generation"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":                true,
			"applied":           false,
			"reason":            "stale_policy_generation",
			"policy_generation": 4,
		})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	response, err := client.ResolveIntent(
		context.Background(),
		intentID,
		map[string]any{"status": "COMPLETED", "expected_policy_generation": 3},
		RequestOptions{
			OwnerAgent:    "agent://owner",
			XOwnerAgent:   "agent://owner",
			Authorization: "Bearer scoped-token",
			TraceID:       "trace-1",
		},
	)
	if err != nil {
		t.Fatalf("resolve intent failed: %v", err)
	}
	if response["applied"] != false {
		t.Fatalf("unexpected response: %v", response)
	}
}

func TestResumeIntentPostsPayload(t *testing.T) {
	intentID := "22222222-2222-4222-8222-222222222222"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/intents/"+intentID+"/resume" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("owner_agent"); got != "agent://owner" {
			t.Fatalf("unexpected owner_agent query: %s", got)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "resume-1" {
			t.Fatalf("unexpected idempotency header: %s", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["approve_current_step"] != true {
			t.Fatalf("unexpected approve_current_step: %v", body["approve_current_step"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "applied": true})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	response, err := client.ResumeIntent(
		context.Background(),
		intentID,
		map[string]any{"approve_current_step": true, "expected_policy_generation": 2},
		RequestOptions{OwnerAgent: "agent://owner", IdempotencyKey: "resume-1"},
	)
	if err != nil {
		t.Fatalf("resume intent failed: %v", err)
	}
	if response["applied"] != true {
		t.Fatalf("unexpected response: %v", response)
	}
}

func TestUpdateIntentControlsAndPolicy(t *testing.T) {
	intentID := "22222222-2222-4222-8222-222222222222"
	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		switch call {
		case 1:
			if r.URL.Path != "/v1/intents/"+intentID+"/controls" {
				t.Fatalf("unexpected controls path: %s", r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode controls body: %v", err)
			}
			controlsPatch := body["controls_patch"].(map[string]any)
			if controlsPatch["timeout_seconds"] != float64(120) {
				t.Fatalf("unexpected timeout_seconds: %v", controlsPatch["timeout_seconds"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "applied": true, "policy_generation": 5})
		case 2:
			if r.URL.Path != "/v1/intents/"+intentID+"/policy" {
				t.Fatalf("unexpected policy path: %s", r.URL.Path)
			}
			if got := r.URL.Query().Get("owner_agent"); got != "agent://creator" {
				t.Fatalf("unexpected owner_agent query: %s", got)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode policy body: %v", err)
			}
			envelopePatch := body["envelope_patch"].(map[string]any)
			if envelopePatch["max_retry_count"] != float64(10) {
				t.Fatalf("unexpected max_retry_count: %v", envelopePatch["max_retry_count"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "applied": true, "policy_generation": 6})
		default:
			t.Fatalf("unexpected call: %d", call)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	controlsResponse, err := client.UpdateIntentControls(
		context.Background(),
		intentID,
		map[string]any{
			"controls_patch": map[string]any{"timeout_seconds": 120},
		},
		RequestOptions{},
	)
	if err != nil {
		t.Fatalf("update intent controls failed: %v", err)
	}
	if controlsResponse["policy_generation"] != float64(5) {
		t.Fatalf("unexpected controls response: %v", controlsResponse)
	}

	policyResponse, err := client.UpdateIntentPolicy(
		context.Background(),
		intentID,
		map[string]any{
			"grants_patch": map[string]any{
				"delegate:agent://ops": map[string]any{
					"allow": []any{"resume", "update_controls"},
				},
			},
			"envelope_patch": map[string]any{"max_retry_count": 10},
		},
		RequestOptions{OwnerAgent: "agent://creator"},
	)
	if err != nil {
		t.Fatalf("update intent policy failed: %v", err)
	}
	if policyResponse["policy_generation"] != float64(6) {
		t.Fatalf("unexpected policy response: %v", policyResponse)
	}
}

func TestCreateIntentGetIntentAndListIntentEvents(t *testing.T) {
	intentID := "22222222-2222-4222-8222-222222222222"
	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		switch call {
		case 1:
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method for create: %s", r.Method)
			}
			if r.URL.Path != "/v1/intents" {
				t.Fatalf("unexpected create path: %s", r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "intent_id": intentID})
		case 2:
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method for get: %s", r.Method)
			}
			if r.URL.Path != "/v1/intents/"+intentID {
				t.Fatalf("unexpected get path: %s", r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "intent": map[string]any{"intent_id": intentID}})
		case 3:
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method for events: %s", r.Method)
			}
			if r.URL.Path != "/v1/intents/"+intentID+"/events" {
				t.Fatalf("unexpected events path: %s", r.URL.Path)
			}
			if got := r.URL.Query().Get("since"); got != "2" {
				t.Fatalf("unexpected since query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "events": []any{}})
		default:
			t.Fatalf("unexpected call: %d", call)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	created, err := client.CreateIntent(
		context.Background(),
		map[string]any{
			"intent_type": "notify.message.v1",
			"from_agent":  "agent://self",
			"to_agent":    "agent://target",
			"payload":     map[string]any{"text": "hello"},
		},
		RequestOptions{},
	)
	if err != nil {
		t.Fatalf("create intent failed: %v", err)
	}
	if created["intent_id"] != intentID {
		t.Fatalf("unexpected create response: %v", created)
	}

	intent, err := client.GetIntent(context.Background(), intentID, RequestOptions{})
	if err != nil {
		t.Fatalf("get intent failed: %v", err)
	}
	if intent["ok"] != true {
		t.Fatalf("unexpected get response: %v", intent)
	}

	since := 2
	events, err := client.ListIntentEvents(context.Background(), intentID, &since, RequestOptions{})
	if err != nil {
		t.Fatalf("list intent events failed: %v", err)
	}
	if events["ok"] != true {
		t.Fatalf("unexpected events response: %v", events)
	}
}

func TestAccessRequestEndpoints(t *testing.T) {
	accessRequestID := "ar_123"
	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		switch call {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/access-requests" {
				t.Fatalf("unexpected create request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.Header.Get("Idempotency-Key"); got != "ar-create-1" {
				t.Fatalf("unexpected idempotency header: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "access_request_id": accessRequestID})
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/access-requests" {
				t.Fatalf("unexpected list request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("org_id"); got != "org_1" {
				t.Fatalf("unexpected org_id query: %s", got)
			}
			if got := r.URL.Query().Get("workspace_id"); got != "ws_1" {
				t.Fatalf("unexpected workspace_id query: %s", got)
			}
			if got := r.URL.Query().Get("state"); got != "pending" {
				t.Fatalf("unexpected state query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "items": []any{}})
		case 3:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/access-requests/"+accessRequestID {
				t.Fatalf("unexpected get request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "access_request": map[string]any{"access_request_id": accessRequestID}})
		case 4:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/access-requests/"+accessRequestID+"/review" {
				t.Fatalf("unexpected review request: %s %s", r.Method, r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode review body: %v", err)
			}
			if body["decision"] != "approve" {
				t.Fatalf("unexpected decision: %v", body["decision"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "access_request_id": accessRequestID, "state": "approved"})
		default:
			t.Fatalf("unexpected call: %d", call)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	created, err := client.CreateAccessRequest(
		context.Background(),
		map[string]any{"owner_agent": "agent://user/1", "scope": "workspace:read"},
		RequestOptions{IdempotencyKey: "ar-create-1"},
	)
	if err != nil {
		t.Fatalf("create access request failed: %v", err)
	}
	if created["access_request_id"] != accessRequestID {
		t.Fatalf("unexpected create response: %v", created)
	}

	listed, err := client.ListAccessRequests(context.Background(), "org_1", "ws_1", "pending", RequestOptions{})
	if err != nil {
		t.Fatalf("list access requests failed: %v", err)
	}
	if listed["ok"] != true {
		t.Fatalf("unexpected list response: %v", listed)
	}

	got, err := client.GetAccessRequest(context.Background(), accessRequestID, RequestOptions{})
	if err != nil {
		t.Fatalf("get access request failed: %v", err)
	}
	if got["ok"] != true {
		t.Fatalf("unexpected get response: %v", got)
	}

	reviewed, err := client.ReviewAccessRequest(
		context.Background(),
		accessRequestID,
		map[string]any{"decision": "approve"},
		RequestOptions{IdempotencyKey: "ar-review-1"},
	)
	if err != nil {
		t.Fatalf("review access request failed: %v", err)
	}
	if reviewed["state"] != "approved" {
		t.Fatalf("unexpected review response: %v", reviewed)
	}
}

func TestAliasApprovalAndCapabilitiesEndpoints(t *testing.T) {
	aliasID := "al_123"
	approvalID := "ap_123"
	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		switch call {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/aliases" {
				t.Fatalf("unexpected bind request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "alias_id": aliasID})
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/aliases" {
				t.Fatalf("unexpected list aliases request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("org_id"); got != "org_1" {
				t.Fatalf("unexpected org_id query: %s", got)
			}
			if got := r.URL.Query().Get("workspace_id"); got != "ws_1" {
				t.Fatalf("unexpected workspace_id query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "items": []any{}})
		case 3:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/aliases/resolve" {
				t.Fatalf("unexpected resolve alias request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("alias"); got != "@support" {
				t.Fatalf("unexpected alias query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "target": "agent://support"})
		case 4:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/aliases/"+aliasID+"/revoke" {
				t.Fatalf("unexpected revoke alias request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "revoked": true})
		case 5:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/approvals/"+approvalID+"/decision" {
				t.Fatalf("unexpected approval decision request: %s %s", r.Method, r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode approval body: %v", err)
			}
			if body["decision"] != "approve" {
				t.Fatalf("unexpected approval decision: %v", body["decision"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "approval_id": approvalID, "decision": "approve"})
		case 6:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/capabilities" {
				t.Fatalf("unexpected capabilities request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "capabilities": []any{"intent.submit"}})
		default:
			t.Fatalf("unexpected call: %d", call)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	bound, err := client.BindAlias(
		context.Background(),
		map[string]any{"alias": "@support", "target_agent": "agent://support"},
		RequestOptions{IdempotencyKey: "alias-bind-1"},
	)
	if err != nil {
		t.Fatalf("bind alias failed: %v", err)
	}
	if bound["alias_id"] != aliasID {
		t.Fatalf("unexpected bind response: %v", bound)
	}

	aliases, err := client.ListAliases(context.Background(), "org_1", "ws_1", RequestOptions{})
	if err != nil {
		t.Fatalf("list aliases failed: %v", err)
	}
	if aliases["ok"] != true {
		t.Fatalf("unexpected aliases response: %v", aliases)
	}

	resolved, err := client.ResolveAlias(context.Background(), "org_1", "ws_1", "@support", RequestOptions{})
	if err != nil {
		t.Fatalf("resolve alias failed: %v", err)
	}
	if resolved["target"] != "agent://support" {
		t.Fatalf("unexpected resolve response: %v", resolved)
	}

	revoked, err := client.RevokeAlias(context.Background(), aliasID, RequestOptions{IdempotencyKey: "alias-revoke-1"})
	if err != nil {
		t.Fatalf("revoke alias failed: %v", err)
	}
	if revoked["revoked"] != true {
		t.Fatalf("unexpected revoke response: %v", revoked)
	}

	approval, err := client.DecideApproval(
		context.Background(),
		approvalID,
		map[string]any{"decision": "approve"},
		RequestOptions{IdempotencyKey: "approval-1"},
	)
	if err != nil {
		t.Fatalf("decide approval failed: %v", err)
	}
	if approval["decision"] != "approve" {
		t.Fatalf("unexpected approval response: %v", approval)
	}

	capabilities, err := client.GetCapabilities(context.Background(), RequestOptions{})
	if err != nil {
		t.Fatalf("get capabilities failed: %v", err)
	}
	if capabilities["ok"] != true {
		t.Fatalf("unexpected capabilities response: %v", capabilities)
	}
}

func TestInboxEndpoints(t *testing.T) {
	threadID := "thread_123"
	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		switch call {
		case 1:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/inbox" {
				t.Fatalf("unexpected list inbox request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("owner_agent"); got != "agent://owner" {
				t.Fatalf("unexpected owner_agent query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "threads": []any{}})
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/inbox/"+threadID {
				t.Fatalf("unexpected get inbox thread request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("owner_agent"); got != "agent://owner" {
				t.Fatalf("unexpected owner_agent query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "thread_id": threadID})
		case 3:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/inbox/changes" {
				t.Fatalf("unexpected list inbox changes request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("owner_agent"); got != "agent://owner" {
				t.Fatalf("unexpected owner_agent query: %s", got)
			}
			if got := r.URL.Query().Get("cursor"); got != "cur-1" {
				t.Fatalf("unexpected cursor query: %s", got)
			}
			if got := r.URL.Query().Get("limit"); got != "50" {
				t.Fatalf("unexpected limit query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "changes": []any{}, "next_cursor": "cur-2"})
		case 4:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/inbox/"+threadID+"/reply" {
				t.Fatalf("unexpected reply inbox thread request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.Header.Get("Idempotency-Key"); got != "reply-1" {
				t.Fatalf("unexpected idempotency header: %s", got)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode reply body: %v", err)
			}
			if body["message"] != "ack" {
				t.Fatalf("unexpected message: %v", body["message"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "thread_id": threadID})
		case 5:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/inbox/"+threadID+"/delegate" {
				t.Fatalf("unexpected delegate inbox thread request: %s %s", r.Method, r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode delegate body: %v", err)
			}
			if body["delegate_to"] != "agent://delegate" {
				t.Fatalf("unexpected delegate_to: %v", body["delegate_to"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "applied": true})
		case 6:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/inbox/"+threadID+"/approve" {
				t.Fatalf("unexpected approve inbox thread request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "decision": "approve"})
		case 7:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/inbox/"+threadID+"/reject" {
				t.Fatalf("unexpected reject inbox thread request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "decision": "reject"})
		case 8:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/inbox/"+threadID+"/messages/delete" {
				t.Fatalf("unexpected delete inbox messages request: %s %s", r.Method, r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode delete messages body: %v", err)
			}
			ids, ok := body["message_ids"].([]any)
			if !ok || len(ids) != 2 {
				t.Fatalf("unexpected message_ids: %v", body["message_ids"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "deleted_count": 2})
		default:
			t.Fatalf("unexpected call: %d", call)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	inbox, err := client.ListInbox(context.Background(), "agent://owner", RequestOptions{})
	if err != nil {
		t.Fatalf("list inbox failed: %v", err)
	}
	if inbox["ok"] != true {
		t.Fatalf("unexpected list inbox response: %v", inbox)
	}

	thread, err := client.GetInboxThread(context.Background(), threadID, "agent://owner", RequestOptions{})
	if err != nil {
		t.Fatalf("get inbox thread failed: %v", err)
	}
	if thread["thread_id"] != threadID {
		t.Fatalf("unexpected get inbox thread response: %v", thread)
	}

	limit := 50
	changes, err := client.ListInboxChanges(context.Background(), "agent://owner", "cur-1", &limit, RequestOptions{})
	if err != nil {
		t.Fatalf("list inbox changes failed: %v", err)
	}
	if changes["next_cursor"] != "cur-2" {
		t.Fatalf("unexpected list inbox changes response: %v", changes)
	}

	replied, err := client.ReplyInboxThread(
		context.Background(),
		threadID,
		"ack",
		"agent://owner",
		RequestOptions{IdempotencyKey: "reply-1"},
	)
	if err != nil {
		t.Fatalf("reply inbox thread failed: %v", err)
	}
	if replied["thread_id"] != threadID {
		t.Fatalf("unexpected reply inbox thread response: %v", replied)
	}

	delegated, err := client.DelegateInboxThread(
		context.Background(),
		threadID,
		map[string]any{"delegate_to": "agent://delegate", "note": "handoff"},
		"agent://owner",
		RequestOptions{IdempotencyKey: "delegate-1"},
	)
	if err != nil {
		t.Fatalf("delegate inbox thread failed: %v", err)
	}
	if delegated["applied"] != true {
		t.Fatalf("unexpected delegate inbox thread response: %v", delegated)
	}

	approved, err := client.ApproveInboxThread(
		context.Background(),
		threadID,
		map[string]any{"comment": "approved"},
		"agent://owner",
		RequestOptions{IdempotencyKey: "approve-1"},
	)
	if err != nil {
		t.Fatalf("approve inbox thread failed: %v", err)
	}
	if approved["decision"] != "approve" {
		t.Fatalf("unexpected approve inbox thread response: %v", approved)
	}

	rejected, err := client.RejectInboxThread(
		context.Background(),
		threadID,
		map[string]any{"comment": "reject"},
		"agent://owner",
		RequestOptions{IdempotencyKey: "reject-1"},
	)
	if err != nil {
		t.Fatalf("reject inbox thread failed: %v", err)
	}
	if rejected["decision"] != "reject" {
		t.Fatalf("unexpected reject inbox thread response: %v", rejected)
	}

	deleted, err := client.DeleteInboxMessages(
		context.Background(),
		threadID,
		map[string]any{"message_ids": []any{"msg_1", "msg_2"}},
		"agent://owner",
		RequestOptions{IdempotencyKey: "delete-msg-1"},
	)
	if err != nil {
		t.Fatalf("delete inbox messages failed: %v", err)
	}
	if deleted["deleted_count"] != float64(2) {
		t.Fatalf("unexpected delete inbox messages response: %v", deleted)
	}
}

func TestInviteMediaAndSchemaEndpoints(t *testing.T) {
	token := "tok_123"
	uploadID := "up_123"
	semanticType := "notify.message.v1"
	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		switch call {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/invites/create" {
				t.Fatalf("unexpected create invite request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "token": token})
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/invites/"+token {
				t.Fatalf("unexpected get invite request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "token": token})
		case 3:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/invites/"+token+"/accept" {
				t.Fatalf("unexpected accept invite request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "accepted": true})
		case 4:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/media/create-upload" {
				t.Fatalf("unexpected create media upload request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "upload_id": uploadID})
		case 5:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/media/"+uploadID {
				t.Fatalf("unexpected get media upload request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "upload_id": uploadID})
		case 6:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/media/finalize-upload" {
				t.Fatalf("unexpected finalize media upload request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "status": "finalized"})
		case 7:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/schemas" {
				t.Fatalf("unexpected upsert schema request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "semantic_type": semanticType})
		case 8:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/schemas/"+semanticType {
				t.Fatalf("unexpected get schema request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "semantic_type": semanticType})
		default:
			t.Fatalf("unexpected call: %d", call)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	createdInvite, err := client.CreateInvite(
		context.Background(),
		map[string]any{"owner_agent": "agent://owner", "email": "owner@example.com"},
		RequestOptions{IdempotencyKey: "invite-create-1"},
	)
	if err != nil {
		t.Fatalf("create invite failed: %v", err)
	}
	if createdInvite["token"] != token {
		t.Fatalf("unexpected create invite response: %v", createdInvite)
	}

	invite, err := client.GetInvite(context.Background(), token, RequestOptions{})
	if err != nil {
		t.Fatalf("get invite failed: %v", err)
	}
	if invite["token"] != token {
		t.Fatalf("unexpected get invite response: %v", invite)
	}

	accepted, err := client.AcceptInvite(
		context.Background(),
		token,
		map[string]any{"owner_agent": "agent://owner"},
		RequestOptions{IdempotencyKey: "invite-accept-1"},
	)
	if err != nil {
		t.Fatalf("accept invite failed: %v", err)
	}
	if accepted["accepted"] != true {
		t.Fatalf("unexpected accept invite response: %v", accepted)
	}

	createdUpload, err := client.CreateMediaUpload(
		context.Background(),
		map[string]any{"owner_agent": "agent://owner", "file_name": "report.pdf"},
		RequestOptions{IdempotencyKey: "media-create-1"},
	)
	if err != nil {
		t.Fatalf("create media upload failed: %v", err)
	}
	if createdUpload["upload_id"] != uploadID {
		t.Fatalf("unexpected create media upload response: %v", createdUpload)
	}

	upload, err := client.GetMediaUpload(context.Background(), uploadID, RequestOptions{})
	if err != nil {
		t.Fatalf("get media upload failed: %v", err)
	}
	if upload["upload_id"] != uploadID {
		t.Fatalf("unexpected get media upload response: %v", upload)
	}

	finalized, err := client.FinalizeMediaUpload(
		context.Background(),
		map[string]any{"upload_id": uploadID, "checksum": "abc123"},
		RequestOptions{IdempotencyKey: "media-finalize-1"},
	)
	if err != nil {
		t.Fatalf("finalize media upload failed: %v", err)
	}
	if finalized["status"] != "finalized" {
		t.Fatalf("unexpected finalize media upload response: %v", finalized)
	}

	upsertedSchema, err := client.UpsertSchema(
		context.Background(),
		map[string]any{"semantic_type": semanticType, "schema": map[string]any{"type": "object"}},
		RequestOptions{IdempotencyKey: "schema-upsert-1"},
	)
	if err != nil {
		t.Fatalf("upsert schema failed: %v", err)
	}
	if upsertedSchema["semantic_type"] != semanticType {
		t.Fatalf("unexpected upsert schema response: %v", upsertedSchema)
	}

	schema, err := client.GetSchema(context.Background(), semanticType, RequestOptions{})
	if err != nil {
		t.Fatalf("get schema failed: %v", err)
	}
	if schema["semantic_type"] != semanticType {
		t.Fatalf("unexpected get schema response: %v", schema)
	}
}

func TestWebhookEndpoints(t *testing.T) {
	subscriptionID := "wh_sub_123"
	eventID := "evt_123"
	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		switch call {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/webhooks/subscriptions" {
				t.Fatalf("unexpected upsert webhook subscription request: %s %s", r.Method, r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode webhook subscription body: %v", err)
			}
			eventTypes, ok := body["event_types"].([]any)
			if !ok || len(eventTypes) == 0 {
				t.Fatalf("unexpected event_types: %v", body["event_types"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "subscription_id": subscriptionID})
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/webhooks/subscriptions" {
				t.Fatalf("unexpected list webhook subscriptions request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("owner_agent"); got != "agent://owner" {
				t.Fatalf("unexpected owner_agent query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "items": []any{}})
		case 3:
			if r.Method != http.MethodDelete || r.URL.Path != "/v1/webhooks/subscriptions/"+subscriptionID {
				t.Fatalf("unexpected delete webhook subscription request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("owner_agent"); got != "agent://owner" {
				t.Fatalf("unexpected owner_agent query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "deleted": true})
		case 4:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/webhooks/events" {
				t.Fatalf("unexpected publish webhook event request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("owner_agent"); got != "agent://owner" {
				t.Fatalf("unexpected owner_agent query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "event_id": eventID})
		case 5:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/webhooks/events/"+eventID+"/replay" {
				t.Fatalf("unexpected replay webhook event request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("owner_agent"); got != "agent://owner" {
				t.Fatalf("unexpected owner_agent query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "replayed": true})
		default:
			t.Fatalf("unexpected call: %d", call)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	upserted, err := client.UpsertWebhookSubscription(
		context.Background(),
		map[string]any{
			"owner_agent":  "agent://owner",
			"callback_url": "https://example.com/hook",
			"event_types":  []any{"inbox.thread_created"},
		},
		RequestOptions{IdempotencyKey: "wh-upsert-1"},
	)
	if err != nil {
		t.Fatalf("upsert webhook subscription failed: %v", err)
	}
	if upserted["subscription_id"] != subscriptionID {
		t.Fatalf("unexpected upsert webhook subscription response: %v", upserted)
	}

	listed, err := client.ListWebhookSubscriptions(context.Background(), "agent://owner", RequestOptions{})
	if err != nil {
		t.Fatalf("list webhook subscriptions failed: %v", err)
	}
	if listed["ok"] != true {
		t.Fatalf("unexpected list webhook subscriptions response: %v", listed)
	}

	deleted, err := client.DeleteWebhookSubscription(context.Background(), subscriptionID, "agent://owner", RequestOptions{})
	if err != nil {
		t.Fatalf("delete webhook subscription failed: %v", err)
	}
	if deleted["deleted"] != true {
		t.Fatalf("unexpected delete webhook subscription response: %v", deleted)
	}

	published, err := client.PublishWebhookEvent(
		context.Background(),
		map[string]any{"event_type": "inbox.thread_created", "payload": map[string]any{"thread_id": "thr_1"}},
		"agent://owner",
		RequestOptions{IdempotencyKey: "wh-publish-1"},
	)
	if err != nil {
		t.Fatalf("publish webhook event failed: %v", err)
	}
	if published["event_id"] != eventID {
		t.Fatalf("unexpected publish webhook event response: %v", published)
	}

	replayed, err := client.ReplayWebhookEvent(
		context.Background(),
		eventID,
		"agent://owner",
		RequestOptions{IdempotencyKey: "wh-replay-1"},
	)
	if err != nil {
		t.Fatalf("replay webhook event failed: %v", err)
	}
	if replayed["replayed"] != true {
		t.Fatalf("unexpected replay webhook event response: %v", replayed)
	}
}

func TestOrganizationQuotaUsageAndPrincipalEndpoints(t *testing.T) {
	orgID := "org_1"
	workspaceID := "ws_1"
	memberID := "member_1"
	principalID := "pr_1"
	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		switch call {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/organizations" {
				t.Fatalf("unexpected create organization request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "org_id": orgID})
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/organizations/"+orgID {
				t.Fatalf("unexpected get organization request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "org_id": orgID})
		case 3:
			if r.Method != http.MethodPatch || r.URL.Path != "/v1/organizations/"+orgID {
				t.Fatalf("unexpected update organization request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "updated": true})
		case 4:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/organizations/"+orgID+"/workspaces" {
				t.Fatalf("unexpected create workspace request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "workspace_id": workspaceID})
		case 5:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/organizations/"+orgID+"/workspaces" {
				t.Fatalf("unexpected list workspaces request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "items": []any{}})
		case 6:
			if r.Method != http.MethodPatch || r.URL.Path != "/v1/organizations/"+orgID+"/workspaces/"+workspaceID {
				t.Fatalf("unexpected update workspace request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "updated": true})
		case 7:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/organizations/"+orgID+"/members" {
				t.Fatalf("unexpected list org members request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("workspace_id"); got != workspaceID {
				t.Fatalf("unexpected workspace_id query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "items": []any{}})
		case 8:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/organizations/"+orgID+"/members" {
				t.Fatalf("unexpected add org member request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "member_id": memberID})
		case 9:
			if r.Method != http.MethodPatch || r.URL.Path != "/v1/organizations/"+orgID+"/members/"+memberID {
				t.Fatalf("unexpected update org member request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "updated": true})
		case 10:
			if r.Method != http.MethodDelete || r.URL.Path != "/v1/organizations/"+orgID+"/members/"+memberID {
				t.Fatalf("unexpected remove org member request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "removed": true})
		case 11:
			if r.Method != http.MethodPatch || r.URL.Path != "/v1/quotas" {
				t.Fatalf("unexpected update quota request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "enforced": true})
		case 12:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/quotas" {
				t.Fatalf("unexpected get quota request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("org_id"); got != orgID {
				t.Fatalf("unexpected org_id query: %s", got)
			}
			if got := r.URL.Query().Get("workspace_id"); got != workspaceID {
				t.Fatalf("unexpected workspace_id query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "quota": map[string]any{}})
		case 13:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/usage/summary" {
				t.Fatalf("unexpected usage summary request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("window"); got != "30d" {
				t.Fatalf("unexpected window query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "window": "30d"})
		case 14:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/usage/timeseries" {
				t.Fatalf("unexpected usage timeseries request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("window_days"); got != "7" {
				t.Fatalf("unexpected window_days query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "points": []any{}})
		case 15:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/principals" {
				t.Fatalf("unexpected create principal request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "principal_id": principalID})
		case 16:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/principals/"+principalID {
				t.Fatalf("unexpected get principal request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "principal_id": principalID})
		default:
			t.Fatalf("unexpected call: %d", call)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	createdOrg, err := client.CreateOrganization(
		context.Background(),
		map[string]any{"org_id": orgID, "name": "Acme"},
		RequestOptions{IdempotencyKey: "org-create-1"},
	)
	if err != nil || createdOrg["org_id"] != orgID {
		t.Fatalf("create organization failed: %v, response=%v", err, createdOrg)
	}
	_, _ = client.GetOrganization(context.Background(), orgID, RequestOptions{})
	_, _ = client.UpdateOrganization(
		context.Background(),
		orgID,
		map[string]any{"display_name": "Acme Inc"},
		RequestOptions{IdempotencyKey: "org-update-1"},
	)
	_, _ = client.CreateWorkspace(
		context.Background(),
		orgID,
		map[string]any{"workspace_id": workspaceID, "name": "Primary"},
		RequestOptions{IdempotencyKey: "ws-create-1"},
	)
	_, _ = client.ListWorkspaces(context.Background(), orgID, RequestOptions{})
	_, _ = client.UpdateWorkspace(
		context.Background(),
		orgID,
		workspaceID,
		map[string]any{"name": "Primary Updated"},
		RequestOptions{IdempotencyKey: "ws-update-1"},
	)
	_, _ = client.ListOrganizationMembers(context.Background(), orgID, workspaceID, RequestOptions{})
	addedMember, err := client.AddOrganizationMember(
		context.Background(),
		orgID,
		map[string]any{"owner_agent": "agent://owner", "role": "workspace_admin"},
		RequestOptions{IdempotencyKey: "member-add-1"},
	)
	if err != nil || addedMember["member_id"] != memberID {
		t.Fatalf("add organization member failed: %v, response=%v", err, addedMember)
	}
	_, _ = client.UpdateOrganizationMember(
		context.Background(),
		orgID,
		memberID,
		map[string]any{"role": "workspace_viewer"},
		RequestOptions{IdempotencyKey: "member-update-1"},
	)
	_, _ = client.RemoveOrganizationMember(context.Background(), orgID, memberID, RequestOptions{})
	_, _ = client.UpdateQuota(
		context.Background(),
		map[string]any{"org_id": orgID, "workspace_id": workspaceID, "hard_enforce": true},
		RequestOptions{IdempotencyKey: "quota-update-1"},
	)
	_, _ = client.GetQuota(context.Background(), orgID, workspaceID, RequestOptions{})
	_, _ = client.GetUsageSummary(context.Background(), orgID, workspaceID, "30d", RequestOptions{})
	windowDays := 7
	_, _ = client.GetUsageTimeseries(context.Background(), orgID, workspaceID, &windowDays, RequestOptions{})
	createdPrincipal, err := client.CreatePrincipal(
		context.Background(),
		map[string]any{"owner_agent": "agent://owner", "kind": "service"},
		RequestOptions{IdempotencyKey: "principal-create-1"},
	)
	if err != nil || createdPrincipal["principal_id"] != principalID {
		t.Fatalf("create principal failed: %v, response=%v", err, createdPrincipal)
	}
	gotPrincipal, err := client.GetPrincipal(context.Background(), principalID, RequestOptions{})
	if err != nil || gotPrincipal["principal_id"] != principalID {
		t.Fatalf("get principal failed: %v, response=%v", err, gotPrincipal)
	}
}

func TestRoutingTransportDeliveryAndBillingEndpoints(t *testing.T) {
	routeID := "route_1"
	bindingID := "binding_1"
	deliveryID := "delivery_1"
	invoiceID := "inv_1"
	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		switch call {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/routing/endpoints" {
				t.Fatalf("unexpected register routing endpoint request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "route_id": routeID})
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/routing/endpoints" {
				t.Fatalf("unexpected list routing endpoints request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "items": []any{}})
		case 3:
			if r.Method != http.MethodPatch || r.URL.Path != "/v1/routing/endpoints/"+routeID {
				t.Fatalf("unexpected update routing endpoint request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "updated": true})
		case 4:
			if r.Method != http.MethodDelete || r.URL.Path != "/v1/routing/endpoints/"+routeID {
				t.Fatalf("unexpected remove routing endpoint request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "removed": true})
		case 5:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/routing/resolve" {
				t.Fatalf("unexpected resolve routing request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "target": "queue://delivery"})
		case 6:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/transports/bindings" {
				t.Fatalf("unexpected upsert transport binding request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "binding_id": bindingID})
		case 7:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/transports/bindings" {
				t.Fatalf("unexpected list transport bindings request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "items": []any{}})
		case 8:
			if r.Method != http.MethodDelete || r.URL.Path != "/v1/transports/bindings/"+bindingID {
				t.Fatalf("unexpected remove transport binding request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "removed": true})
		case 9:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/deliveries" {
				t.Fatalf("unexpected submit delivery request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "delivery_id": deliveryID})
		case 10:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/deliveries" {
				t.Fatalf("unexpected list deliveries request: %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("status"); got != "pending" {
				t.Fatalf("unexpected deliveries status query: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "items": []any{}})
		case 11:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/deliveries/"+deliveryID {
				t.Fatalf("unexpected get delivery request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "delivery_id": deliveryID})
		case 12:
			if r.Method != http.MethodPost || r.URL.Path != "/v1/deliveries/"+deliveryID+"/replay" {
				t.Fatalf("unexpected replay delivery request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "replayed": true})
		case 13:
			if r.Method != http.MethodPatch || r.URL.Path != "/v1/billing/plan" {
				t.Fatalf("unexpected update billing plan request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "updated": true})
		case 14:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/billing/plan" {
				t.Fatalf("unexpected get billing plan request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "plan_id": "starter"})
		case 15:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/billing/invoices" {
				t.Fatalf("unexpected list billing invoices request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "items": []any{}})
		case 16:
			if r.Method != http.MethodGet || r.URL.Path != "/v1/billing/invoices/"+invoiceID {
				t.Fatalf("unexpected get billing invoice request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "invoice_id": invoiceID})
		default:
			t.Fatalf("unexpected call: %d", call)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, _ = client.RegisterRoutingEndpoint(
		context.Background(),
		map[string]any{"org_id": "org_1", "workspace_id": "ws_1", "transport": "http"},
		RequestOptions{IdempotencyKey: "route-register-1"},
	)
	_, _ = client.ListRoutingEndpoints(context.Background(), "org_1", "ws_1", RequestOptions{})
	_, _ = client.UpdateRoutingEndpoint(
		context.Background(),
		routeID,
		map[string]any{"weight": 10},
		RequestOptions{IdempotencyKey: "route-update-1"},
	)
	_, _ = client.RemoveRoutingEndpoint(context.Background(), routeID, RequestOptions{})
	resolved, err := client.ResolveRouting(
		context.Background(),
		map[string]any{"org_id": "org_1", "workspace_id": "ws_1", "semantic_type": "notify.message.v1"},
		RequestOptions{IdempotencyKey: "route-resolve-1"},
	)
	if err != nil || resolved["target"] != "queue://delivery" {
		t.Fatalf("resolve routing failed: %v, response=%v", err, resolved)
	}

	_, _ = client.UpsertTransportBinding(
		context.Background(),
		map[string]any{"org_id": "org_1", "workspace_id": "ws_1", "transport": "http"},
		RequestOptions{IdempotencyKey: "binding-upsert-1"},
	)
	_, _ = client.ListTransportBindings(context.Background(), "org_1", "ws_1", RequestOptions{})
	_, _ = client.RemoveTransportBinding(context.Background(), bindingID, RequestOptions{})

	_, _ = client.SubmitDelivery(
		context.Background(),
		map[string]any{"org_id": "org_1", "workspace_id": "ws_1", "principal_id": "pr_1"},
		RequestOptions{IdempotencyKey: "delivery-submit-1"},
	)
	_, _ = client.ListDeliveries(context.Background(), "org_1", "ws_1", "pr_1", "pending", RequestOptions{})
	gotDelivery, err := client.GetDelivery(context.Background(), deliveryID, RequestOptions{})
	if err != nil || gotDelivery["delivery_id"] != deliveryID {
		t.Fatalf("get delivery failed: %v, response=%v", err, gotDelivery)
	}
	replayed, err := client.ReplayDelivery(context.Background(), deliveryID, RequestOptions{IdempotencyKey: "delivery-replay-1"})
	if err != nil || replayed["replayed"] != true {
		t.Fatalf("replay delivery failed: %v, response=%v", err, replayed)
	}

	_, _ = client.UpdateBillingPlan(
		context.Background(),
		map[string]any{"org_id": "org_1", "workspace_id": "ws_1", "plan": "enterprise"},
		RequestOptions{IdempotencyKey: "billing-plan-1"},
	)
	_, _ = client.GetBillingPlan(context.Background(), "org_1", "ws_1", RequestOptions{})
	_, _ = client.ListBillingInvoices(context.Background(), "org_1", "ws_1", "open", RequestOptions{})
	invoice, err := client.GetBillingInvoice(context.Background(), invoiceID, RequestOptions{})
	if err != nil || invoice["invoice_id"] != invoiceID {
		t.Fatalf("get billing invoice failed: %v, response=%v", err, invoice)
	}
}

func makeSseBody(events [][]string) string {
	var sb strings.Builder
	for _, parts := range events {
		eventType := parts[0]
		data := parts[1]
		sb.WriteString("event: ")
		sb.WriteString(eventType)
		sb.WriteString("\n")
		sb.WriteString("data: ")
		sb.WriteString(data)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func TestListenYieldsIntentsFromSse(t *testing.T) {
	intent1 := `{"intent_id":"aaa-1","seq":1,"event_type":"intent.submitted","status":"SUBMITTED"}`
	intent2 := `{"intent_id":"bbb-2","seq":2,"event_type":"intent.submitted","status":"SUBMITTED"}`

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v1/agents/acme/main/router/intents/stream" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "token" {
			t.Fatalf("missing x-api-key header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if calls == 1 {
			_, _ = w.Write([]byte(makeSseBody([][]string{
				{"intent.submitted", intent1},
				{"intent.submitted", intent2},
			})))
		}
	}))
	defer server.Close()

	client, _ := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	ctx, cancel := context.WithCancel(context.Background())

	intents, errCh := client.Listen(ctx, "agent://acme/main/router", ListenOptions{WaitSeconds: 1})

	var received []map[string]any
	received = append(received, <-intents)
	received = append(received, <-intents)
	cancel()

	// Drain channels
	for range intents {
	}
	if err := <-errCh; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(received) != 2 {
		t.Fatalf("expected 2 intents, got %d", len(received))
	}
	if received[0]["intent_id"] != "aaa-1" {
		t.Fatalf("unexpected intent_id: %v", received[0]["intent_id"])
	}
	if received[1]["intent_id"] != "bbb-2" {
		t.Fatalf("unexpected intent_id: %v", received[1]["intent_id"])
	}
}

func TestListenStripsAgentScheme(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
	}))
	defer server.Close()

	client, _ := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	intents, errCh := client.Listen(ctx, "agent://org/ws/svc", ListenOptions{WaitSeconds: 1})
	for range intents {
	}
	<-errCh

	if capturedPath != "/v1/agents/org/ws/svc/intents/stream" {
		t.Fatalf("unexpected path: %s", capturedPath)
	}
}

func TestListenAdvancesSinceCursor(t *testing.T) {
	intent := `{"intent_id":"x-5","seq":5,"event_type":"intent.submitted","status":"SUBMITTED"}`
	sinceValues := []string{}
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		sinceValues = append(sinceValues, r.URL.Query().Get("since"))
		w.Header().Set("Content-Type", "text/event-stream")
		if calls == 1 {
			_, _ = w.Write([]byte(makeSseBody([][]string{{"intent.submitted", intent}})))
		}
	}))
	defer server.Close()

	client, _ := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	ctx, cancel := context.WithCancel(context.Background())

	intents, errCh := client.Listen(ctx, "acme/main/worker", ListenOptions{WaitSeconds: 1})

	// Receive one, then cancel
	<-intents
	cancel()
	for range intents {
	}
	<-errCh

	if len(sinceValues) < 1 {
		t.Fatal("expected at least one stream call")
	}
	if sinceValues[0] != "0" {
		t.Fatalf("first call should have since=0, got %s", sinceValues[0])
	}
	if len(sinceValues) >= 2 && sinceValues[1] != "5" {
		t.Fatalf("second call should have since=5, got %s", sinceValues[1])
	}
}

func TestListenPropagatesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"forbidden"}`, http.StatusForbidden)
	}))
	defer server.Close()

	client, _ := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	ctx := context.Background()

	intents, errCh := client.Listen(ctx, "acme/main/blocked", ListenOptions{WaitSeconds: 1})
	for range intents {
	}
	err := <-errCh
	if err == nil {
		t.Fatal("expected an error from 403 response")
	}
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected *HTTPError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", httpErr.StatusCode)
	}
}

func TestListenIgnoresKeepaliveEvents(t *testing.T) {
	intent := `{"intent_id":"real-1","seq":1,"event_type":"intent.submitted","status":"SUBMITTED"}`
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/event-stream")
		if calls == 1 {
			body := "event: stream.timeout\ndata: {}\n\n" +
				makeSseBody([][]string{{"intent.submitted", intent}})
			_, _ = w.Write([]byte(body))
		}
	}))
	defer server.Close()

	client, _ := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	ctx, cancel := context.WithCancel(context.Background())

	intents, errCh := client.Listen(ctx, "acme/main/keep", ListenOptions{WaitSeconds: 1})

	received := <-intents
	cancel()
	for range intents {
	}
	<-errCh

	if received["intent_id"] != "real-1" {
		t.Fatalf("unexpected intent_id: %v", received["intent_id"])
	}
}

func TestListenStopsOnContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty SSE response — no events
		w.Header().Set("Content-Type", "text/event-stream")
	}))
	defer server.Close()

	client, _ := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "token", HTTPClient: server.Client()})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	intents, errCh := client.Listen(ctx, "acme/main/stopped", ListenOptions{WaitSeconds: 1})
	for range intents {
		t.Fatal("should not receive any intents after immediate cancel")
	}
	err := <-errCh
	if err != nil {
		t.Fatalf("expected nil error on cancel, got %v", err)
	}
}
