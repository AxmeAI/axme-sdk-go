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
