package axme

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// ── Heartbeat ───────────────────────────────────────────────────────────────

func TestMeshHeartbeat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/mesh/heartbeat" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("unexpected x-api-key: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "status": "healthy"})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Mesh().Heartbeat(context.Background(), nil, RequestOptions{})
	if err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}
	if resp["ok"] != true {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestMeshHeartbeatWithMetrics(t *testing.T) {
	var receivedMetrics map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/mesh/heartbeat" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if m, ok := body["metrics"]; ok {
			receivedMetrics = m.(map[string]any)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	metrics := map[string]any{
		"intents_total":     3,
		"intents_succeeded": 2,
		"intents_failed":    1,
		"avg_latency_ms":    42.5,
		"cost_usd":          0.05,
	}
	_, err = client.Mesh().Heartbeat(context.Background(), metrics, RequestOptions{})
	if err != nil {
		t.Fatalf("heartbeat with metrics failed: %v", err)
	}

	if receivedMetrics == nil {
		t.Fatal("expected metrics in body")
	}
	if receivedMetrics["intents_total"] != float64(3) {
		t.Fatalf("unexpected intents_total: %v", receivedMetrics["intents_total"])
	}
}

func TestMeshHeartbeatWithTraceID(t *testing.T) {
	var gotTraceID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = r.Header.Get("X-Trace-Id")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Mesh().Heartbeat(context.Background(), nil, RequestOptions{TraceID: "trace-123"})
	if err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}
	if gotTraceID != "trace-123" {
		t.Fatalf("expected trace-123, got: %s", gotTraceID)
	}
}

// ── Start/Stop Heartbeat Goroutine ──────────────────────────────────────────

func TestMeshStartStopHeartbeat(t *testing.T) {
	var heartbeatCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/mesh/heartbeat" {
			heartbeatCount.Add(1)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	mesh := client.Mesh()
	mesh.StartHeartbeat(HeartbeatOptions{Interval: 50 * time.Millisecond})

	// Wait enough time for at least 2 heartbeats.
	time.Sleep(150 * time.Millisecond)

	mesh.StopHeartbeat()

	count := heartbeatCount.Load()
	if count < 2 {
		t.Fatalf("expected at least 2 heartbeats, got %d", count)
	}
}

func TestMeshStartHeartbeatIdempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	mesh := client.Mesh()
	mesh.StartHeartbeat(HeartbeatOptions{Interval: 50 * time.Millisecond})
	mesh.StartHeartbeat(HeartbeatOptions{Interval: 50 * time.Millisecond}) // Should be no-op

	time.Sleep(80 * time.Millisecond)
	mesh.StopHeartbeat()
}

func TestMeshStopHeartbeatWhenNotRunning(t *testing.T) {
	client, err := NewClient(ClientConfig{BaseURL: "http://localhost", APIKey: "test-key"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	// Should not panic.
	client.Mesh().StopHeartbeat()
}

func TestMeshHeartbeatFlushesMetrics(t *testing.T) {
	var lastMetrics map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/mesh/heartbeat" {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				if m, ok := body["metrics"]; ok && m != nil {
					lastMetrics = m.(map[string]any)
				}
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	mesh := client.Mesh()

	// Buffer some metrics before starting heartbeat.
	latency := 100.0
	cost := 0.05
	mesh.ReportMetric(true, &latency, &cost)

	mesh.StartHeartbeat(HeartbeatOptions{Interval: 50 * time.Millisecond})
	time.Sleep(120 * time.Millisecond)
	mesh.StopHeartbeat()

	if lastMetrics == nil {
		t.Fatal("expected metrics to be flushed with heartbeat")
	}
	if lastMetrics["intents_total"] != float64(1) {
		t.Fatalf("unexpected intents_total: %v", lastMetrics["intents_total"])
	}
	if lastMetrics["intents_succeeded"] != float64(1) {
		t.Fatalf("unexpected intents_succeeded: %v", lastMetrics["intents_succeeded"])
	}
}

// ── Metrics Buffering ───────────────────────────────────────────────────────

func TestMeshReportMetricSuccess(t *testing.T) {
	client, err := NewClient(ClientConfig{BaseURL: "http://localhost", APIKey: "test-key"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	mesh := client.Mesh()
	latency := 100.0
	cost := 0.05
	mesh.ReportMetric(true, &latency, &cost)

	metrics := mesh.flushMetrics()
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
	if metrics["intents_total"] != 1 {
		t.Fatalf("unexpected intents_total: %v", metrics["intents_total"])
	}
	if metrics["intents_succeeded"] != 1 {
		t.Fatalf("unexpected intents_succeeded: %v", metrics["intents_succeeded"])
	}
	if metrics["intents_failed"] != 0 {
		t.Fatalf("unexpected intents_failed: %v", metrics["intents_failed"])
	}
	if metrics["avg_latency_ms"] != 100.0 {
		t.Fatalf("unexpected avg_latency_ms: %v", metrics["avg_latency_ms"])
	}
	if metrics["cost_usd"] != 0.05 {
		t.Fatalf("unexpected cost_usd: %v", metrics["cost_usd"])
	}
}

func TestMeshReportMetricFailure(t *testing.T) {
	client, err := NewClient(ClientConfig{BaseURL: "http://localhost", APIKey: "test-key"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	mesh := client.Mesh()
	mesh.ReportMetric(false, nil, nil)

	metrics := mesh.flushMetrics()
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
	if metrics["intents_total"] != 1 {
		t.Fatalf("unexpected intents_total: %v", metrics["intents_total"])
	}
	if metrics["intents_succeeded"] != 0 {
		t.Fatalf("unexpected intents_succeeded: %v", metrics["intents_succeeded"])
	}
	if metrics["intents_failed"] != 1 {
		t.Fatalf("unexpected intents_failed: %v", metrics["intents_failed"])
	}
	// No latency or cost keys when not provided.
	if _, ok := metrics["avg_latency_ms"]; ok {
		t.Fatal("expected no avg_latency_ms key")
	}
	if _, ok := metrics["cost_usd"]; ok {
		t.Fatal("expected no cost_usd key")
	}
}

func TestMeshReportMetricRunningAverage(t *testing.T) {
	client, err := NewClient(ClientConfig{BaseURL: "http://localhost", APIKey: "test-key"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	mesh := client.Mesh()
	lat1 := 100.0
	lat2 := 200.0
	lat3 := 300.0
	mesh.ReportMetric(true, &lat1, nil)
	mesh.ReportMetric(true, &lat2, nil)
	mesh.ReportMetric(false, &lat3, nil)

	metrics := mesh.flushMetrics()
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
	if metrics["intents_total"] != 3 {
		t.Fatalf("unexpected intents_total: %v", metrics["intents_total"])
	}
	if metrics["intents_succeeded"] != 2 {
		t.Fatalf("unexpected intents_succeeded: %v", metrics["intents_succeeded"])
	}
	if metrics["intents_failed"] != 1 {
		t.Fatalf("unexpected intents_failed: %v", metrics["intents_failed"])
	}

	avgLatency := metrics["avg_latency_ms"].(float64)
	expected := 200.0
	if avgLatency < expected-0.01 || avgLatency > expected+0.01 {
		t.Fatalf("expected avg_latency_ms ~200.0, got: %v", avgLatency)
	}
}

func TestMeshReportMetricCostAccumulates(t *testing.T) {
	client, err := NewClient(ClientConfig{BaseURL: "http://localhost", APIKey: "test-key"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	mesh := client.Mesh()
	cost1 := 0.10
	cost2 := 0.25
	mesh.ReportMetric(true, nil, &cost1)
	mesh.ReportMetric(true, nil, &cost2)

	metrics := mesh.flushMetrics()
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
	costUSD := metrics["cost_usd"].(float64)
	expected := 0.35
	if costUSD < expected-0.001 || costUSD > expected+0.001 {
		t.Fatalf("expected cost_usd ~0.35, got: %v", costUSD)
	}
}

func TestMeshFlushMetricsResetsBuffer(t *testing.T) {
	client, err := NewClient(ClientConfig{BaseURL: "http://localhost", APIKey: "test-key"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	mesh := client.Mesh()
	lat := 50.0
	mesh.ReportMetric(true, &lat, nil)

	metrics1 := mesh.flushMetrics()
	if metrics1 == nil {
		t.Fatal("expected non-nil metrics on first flush")
	}

	// Second flush should return nil (buffer reset).
	metrics2 := mesh.flushMetrics()
	if metrics2 != nil {
		t.Fatalf("expected nil after flush, got: %v", metrics2)
	}
}

func TestMeshFlushMetricsEmptyBuffer(t *testing.T) {
	client, err := NewClient(ClientConfig{BaseURL: "http://localhost", APIKey: "test-key"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	metrics := client.Mesh().flushMetrics()
	if metrics != nil {
		t.Fatalf("expected nil for empty buffer, got: %v", metrics)
	}
}

// ── ListAgents ──────────────────────────────────────────────────────────────

func TestMeshListAgents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/mesh/agents" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Fatalf("unexpected limit: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":     true,
			"agents": []any{map[string]any{"address_id": "agent-1", "health": "healthy"}},
		})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Mesh().ListAgents(context.Background(), 50, "", RequestOptions{})
	if err != nil {
		t.Fatalf("list agents failed: %v", err)
	}
	if resp["ok"] != true {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestMeshListAgentsWithHealthFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("health"); got != "unhealthy" {
			t.Fatalf("unexpected health filter: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "agents": []any{}})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Mesh().ListAgents(context.Background(), 100, "unhealthy", RequestOptions{})
	if err != nil {
		t.Fatalf("list agents failed: %v", err)
	}
	if resp["ok"] != true {
		t.Fatalf("unexpected response: %v", resp)
	}
}

// ── GetAgent ────────────────────────────────────────────────────────────────

func TestMeshGetAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/mesh/agents/agent-123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":         true,
			"address_id": "agent-123",
			"health":     "healthy",
		})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Mesh().GetAgent(context.Background(), "agent-123", RequestOptions{})
	if err != nil {
		t.Fatalf("get agent failed: %v", err)
	}
	if resp["address_id"] != "agent-123" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

// ── Kill ────────────────────────────────────────────────────────────────────

func TestMeshKill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/mesh/agents/agent-123/kill" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "status": "killed"})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Mesh().Kill(context.Background(), "agent-123", RequestOptions{})
	if err != nil {
		t.Fatalf("kill failed: %v", err)
	}
	if resp["status"] != "killed" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

// ── Resume ──────────────────────────────────────────────────────────────────

func TestMeshResume(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/mesh/agents/agent-123/resume" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "status": "active"})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Mesh().Resume(context.Background(), "agent-123", RequestOptions{})
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if resp["status"] != "active" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

// ── ListEvents ──────────────────────────────────────────────────────────────

func TestMeshListEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/mesh/events" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "25" {
			t.Fatalf("unexpected limit: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":     true,
			"events": []any{map[string]any{"event_type": "agent.killed", "address_id": "agent-1"}},
		})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Mesh().ListEvents(context.Background(), 25, "", RequestOptions{})
	if err != nil {
		t.Fatalf("list events failed: %v", err)
	}
	if resp["ok"] != true {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestMeshListEventsWithTypeFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("event_type"); got != "agent.killed" {
			t.Fatalf("unexpected event_type filter: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "events": []any{}})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Mesh().ListEvents(context.Background(), 50, "agent.killed", RequestOptions{})
	if err != nil {
		t.Fatalf("list events failed: %v", err)
	}
	if resp["ok"] != true {
		t.Fatalf("unexpected response: %v", resp)
	}
}

// ── Lazy init ───────────────────────────────────────────────────────────────

func TestMeshLazyInit(t *testing.T) {
	client, err := NewClient(ClientConfig{BaseURL: "http://localhost", APIKey: "test-key"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	mesh1 := client.Mesh()
	mesh2 := client.Mesh()

	if mesh1 != mesh2 {
		t.Fatal("expected Mesh() to return the same instance")
	}
}

// ── Error handling ──────────────────────────────────────────────────────────

func TestMeshHeartbeatHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal"}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Mesh().Heartbeat(context.Background(), nil, RequestOptions{})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got: %T", err)
	}
	if httpErr.StatusCode != 500 {
		t.Fatalf("unexpected status: %d", httpErr.StatusCode)
	}
}

func TestMeshKillHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"agent not found"}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Mesh().Kill(context.Background(), "nonexistent", RequestOptions{})
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got: %T", err)
	}
	if httpErr.StatusCode != 404 {
		t.Fatalf("unexpected status: %d", httpErr.StatusCode)
	}
}

// ── Background heartbeat tolerates errors ───────────────────────────────────

func TestMeshStartHeartbeatToleratesErrors(t *testing.T) {
	var callCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	mesh := client.Mesh()
	mesh.StartHeartbeat(HeartbeatOptions{Interval: 50 * time.Millisecond})
	time.Sleep(150 * time.Millisecond)
	mesh.StopHeartbeat()

	// The goroutine should keep running despite errors.
	count := callCount.Load()
	if count < 2 {
		t.Fatalf("expected at least 2 attempts despite errors, got %d", count)
	}
}

// ── Heartbeat without metrics ───────────────────────────────────────────────

func TestMeshStartHeartbeatWithoutMetrics(t *testing.T) {
	var receivedBody bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/mesh/heartbeat" {
			if r.ContentLength > 0 {
				receivedBody = true
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	mesh := client.Mesh()
	// Buffer a metric.
	lat := 50.0
	mesh.ReportMetric(true, &lat, nil)

	includeMetrics := false
	mesh.StartHeartbeat(HeartbeatOptions{
		Interval:       50 * time.Millisecond,
		IncludeMetrics: &includeMetrics,
	})
	time.Sleep(120 * time.Millisecond)
	mesh.StopHeartbeat()

	if receivedBody {
		t.Fatal("expected no body when IncludeMetrics=false")
	}
}

// ── Authorization header propagation ────────────────────────────────────────

func TestMeshUsesActorToken(t *testing.T) {
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "agents": []any{}})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		ActorToken: "my-actor-token",
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Mesh().ListAgents(context.Background(), 10, "", RequestOptions{})
	if err != nil {
		t.Fatalf("list agents failed: %v", err)
	}
	if gotAuth != "Bearer my-actor-token" {
		t.Fatalf("expected actor token in Authorization header, got: %s", gotAuth)
	}
}
