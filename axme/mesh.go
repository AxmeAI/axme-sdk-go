package axme

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// MeshClient provides Agent Mesh operations: heartbeat, health monitoring,
// metrics reporting, and agent lifecycle management (kill/resume).
type MeshClient struct {
	client *Client

	mu             sync.Mutex
	metricsBuffer  metricsBuffer
	heartbeatCtx   context.Context
	heartbeatStop  context.CancelFunc
	heartbeatRunning bool
}

// metricsBuffer accumulates metric observations between heartbeat flushes.
type metricsBuffer struct {
	intentsTotal     int
	intentsSucceeded int
	intentsFailed    int
	avgLatencyMs     float64
	costUSD          float64
	hasCost          bool
	hasLatency       bool
}

func newMeshClient(client *Client) *MeshClient {
	return &MeshClient{
		client: client,
	}
}

// ── Heartbeat ───────────────────────────────────────────────────────────────

// Heartbeat sends a single heartbeat to the mesh, optionally including metrics.
func (m *MeshClient) Heartbeat(
	ctx context.Context,
	metrics map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	var body map[string]any
	if metrics != nil {
		body = map[string]any{"metrics": metrics}
	}
	return m.client.requestJSON(ctx, http.MethodPost, "/v1/mesh/heartbeat", nil, body, options)
}

// HeartbeatOptions configures the background heartbeat goroutine.
type HeartbeatOptions struct {
	// Interval between heartbeats. Default: 30 seconds.
	Interval time.Duration
	// IncludeMetrics controls whether buffered metrics are flushed with each heartbeat.
	// Default: true.
	IncludeMetrics *bool
}

// StartHeartbeat launches a background goroutine that sends heartbeats at
// regular intervals. If a heartbeat goroutine is already running, this is a no-op.
// The goroutine stops when StopHeartbeat is called or the returned context is cancelled.
func (m *MeshClient) StartHeartbeat(opts HeartbeatOptions) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.heartbeatRunning {
		return
	}

	interval := opts.Interval
	if interval <= 0 {
		interval = 30 * time.Second
	}

	includeMetrics := true
	if opts.IncludeMetrics != nil {
		includeMetrics = *opts.IncludeMetrics
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.heartbeatCtx = ctx
	m.heartbeatStop = cancel
	m.heartbeatRunning = true

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var metrics map[string]any
				if includeMetrics {
					metrics = m.flushMetrics()
				}
				// Heartbeat failures are non-fatal; we silently continue.
				_, _ = m.Heartbeat(ctx, metrics, RequestOptions{})
			}
		}
	}()
}

// StopHeartbeat stops the background heartbeat goroutine. Safe to call even if
// no heartbeat is running.
func (m *MeshClient) StopHeartbeat() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.heartbeatRunning {
		return
	}

	m.heartbeatStop()
	m.heartbeatRunning = false
	m.heartbeatCtx = nil
	m.heartbeatStop = nil
}

// ── Metrics ─────────────────────────────────────────────────────────────────

// ReportMetric buffers a metric observation that will be flushed with the next
// heartbeat. Thread-safe.
func (m *MeshClient) ReportMetric(success bool, latencyMs *float64, costUSD *float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metricsBuffer.intentsTotal++
	if success {
		m.metricsBuffer.intentsSucceeded++
	} else {
		m.metricsBuffer.intentsFailed++
	}

	if latencyMs != nil {
		m.metricsBuffer.hasLatency = true
		// Running average: avg_new = avg_old + (x - avg_old) / n
		count := m.metricsBuffer.intentsTotal
		prevAvg := m.metricsBuffer.avgLatencyMs
		m.metricsBuffer.avgLatencyMs = prevAvg + (*latencyMs-prevAvg)/float64(count)
	}

	if costUSD != nil {
		m.metricsBuffer.hasCost = true
		m.metricsBuffer.costUSD += *costUSD
	}
}

// flushMetrics returns the accumulated metrics and resets the buffer.
// Returns nil if no metrics have been reported. Thread-safe.
func (m *MeshClient) flushMetrics() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.metricsBuffer.intentsTotal == 0 {
		return nil
	}

	metrics := map[string]any{
		"intents_total":     m.metricsBuffer.intentsTotal,
		"intents_succeeded": m.metricsBuffer.intentsSucceeded,
		"intents_failed":    m.metricsBuffer.intentsFailed,
	}
	if m.metricsBuffer.hasLatency {
		metrics["avg_latency_ms"] = m.metricsBuffer.avgLatencyMs
	}
	if m.metricsBuffer.hasCost {
		metrics["cost_usd"] = m.metricsBuffer.costUSD
	}

	// Reset buffer.
	m.metricsBuffer = metricsBuffer{}

	return metrics
}

// ── Agent Management ────────────────────────────────────────────────────────

// MeshListAgents lists all agents in the workspace with health status.
func (m *MeshClient) ListAgents(
	ctx context.Context,
	limit int,
	health string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if limit > 0 {
		query["limit"] = strconv.Itoa(limit)
	}
	if health != "" {
		query["health"] = health
	}
	return m.client.requestJSON(ctx, http.MethodGet, "/v1/mesh/agents", query, nil, options)
}

// GetAgent returns detail for a single agent including metrics and events.
func (m *MeshClient) GetAgent(
	ctx context.Context,
	addressID string,
	options RequestOptions,
) (map[string]any, error) {
	return m.client.requestJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/v1/mesh/agents/%s", addressID),
		nil,
		nil,
		options,
	)
}

// Kill blocks all intents to and from the specified agent.
func (m *MeshClient) Kill(
	ctx context.Context,
	addressID string,
	options RequestOptions,
) (map[string]any, error) {
	return m.client.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/mesh/agents/%s/kill", addressID),
		nil,
		nil,
		options,
	)
}

// Resume reactivates a previously killed agent.
func (m *MeshClient) Resume(
	ctx context.Context,
	addressID string,
	options RequestOptions,
) (map[string]any, error) {
	return m.client.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/mesh/agents/%s/resume", addressID),
		nil,
		nil,
		options,
	)
}

// ── Events ──────────────────────────────────────────────────────────────────

// ListEvents returns recent mesh events (kills, resumes, health changes).
func (m *MeshClient) ListEvents(
	ctx context.Context,
	limit int,
	eventType string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if limit > 0 {
		query["limit"] = strconv.Itoa(limit)
	}
	if eventType != "" {
		query["event_type"] = eventType
	}
	return m.client.requestJSON(ctx, http.MethodGet, "/v1/mesh/events", query, nil, options)
}
