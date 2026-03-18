package axme

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.cloud.axme.ai"

type ClientConfig struct {
	BaseURL     string
	APIKey      string
	ActorToken  string
	BearerToken string
	HTTPClient  *http.Client
}

type RequestOptions struct {
	IdempotencyKey string
	TraceID        string
	OwnerAgent     string
	XOwnerAgent    string
	Authorization  string
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("axme request failed with status %d", e.StatusCode)
}

type Client struct {
	baseURL    string
	apiKey     string
	actorToken string
	httpClient *http.Client
}

func NewClient(config ClientConfig) (*Client, error) {
	baseURL := strings.TrimSpace(config.BaseURL)
	apiKey := strings.TrimSpace(config.APIKey)
	actorToken := strings.TrimSpace(config.ActorToken)
	bearerToken := strings.TrimSpace(config.BearerToken)

	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if actorToken != "" && bearerToken != "" && actorToken != bearerToken {
		return nil, fmt.Errorf("actorToken and bearerToken must match when both are provided")
	}
	if actorToken == "" && bearerToken != "" {
		actorToken = bearerToken
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		actorToken: actorToken,
		httpClient: httpClient,
	}, nil
}

func (c *Client) RegisterNick(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/users/register-nick", nil, payload, options)
}

func (c *Client) CheckNick(
	ctx context.Context,
	nick string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		"/v1/users/check-nick",
		map[string]string{"nick": nick},
		nil,
		options,
	)
}

func (c *Client) RenameNick(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/users/rename-nick", nil, payload, options)
}

func (c *Client) GetUserProfile(
	ctx context.Context,
	ownerAgent string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		"/v1/users/profile",
		map[string]string{"owner_agent": ownerAgent},
		nil,
		options,
	)
}

func (c *Client) UpdateUserProfile(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/users/profile/update", nil, payload, options)
}

func (c *Client) CreateServiceAccount(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/service-accounts", nil, payload, options)
}

func (c *Client) ListServiceAccounts(
	ctx context.Context,
	orgID string,
	workspaceID string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{"org_id": orgID}
	if strings.TrimSpace(workspaceID) != "" {
		query["workspace_id"] = workspaceID
	}
	return c.requestJSON(ctx, http.MethodGet, "/v1/service-accounts", query, nil, options)
}

func (c *Client) GetServiceAccount(
	ctx context.Context,
	serviceAccountID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/v1/service-accounts/%s", serviceAccountID),
		nil,
		nil,
		options,
	)
}

func (c *Client) CreateServiceAccountKey(
	ctx context.Context,
	serviceAccountID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/service-accounts/%s/keys", serviceAccountID),
		nil,
		payload,
		options,
	)
}

func (c *Client) RevokeServiceAccountKey(
	ctx context.Context,
	serviceAccountID string,
	keyID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/service-accounts/%s/keys/%s/revoke", serviceAccountID, keyID),
		nil,
		nil,
		options,
	)
}

func (c *Client) ListAgents(
	ctx context.Context,
	orgID string,
	workspaceID string,
	limit *int,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{"org_id": orgID, "workspace_id": workspaceID}
	if limit != nil && *limit > 0 {
		query["limit"] = strconv.Itoa(*limit)
	}
	return c.requestJSON(ctx, http.MethodGet, "/v1/agents", query, nil, options)
}

func (c *Client) GetAgent(
	ctx context.Context,
	address string,
	options RequestOptions,
) (map[string]any, error) {
	pathPart := strings.TrimPrefix(strings.TrimSpace(address), "agent://")
	return c.requestJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/v1/agents/%s", pathPart),
		nil,
		nil,
		options,
	)
}

func (c *Client) CreateIntent(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/intents", nil, payload, options)
}

// ApplyScenario submits a ScenarioBundle to POST /v1/scenarios/apply.
//
// The server provisions missing agents, compiles the workflow, and creates the intent in one
// atomic operation.  Returns the full bundle response including intent_id, compile_id,
// and agents_provisioned.
func (c *Client) ApplyScenario(
	ctx context.Context,
	bundle map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/scenarios/apply", nil, bundle, options)
}

// ValidateScenario performs a dry-run validation of a ScenarioBundle without creating any
// resources.  Returns {"valid": bool, "errors": []string}.
func (c *Client) ValidateScenario(
	ctx context.Context,
	bundle map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/scenarios/validate", nil, bundle, options)
}

// ListenOptions controls how Listen reconnects and yields events.
type ListenOptions struct {
	// Since is the minimum seq value from which to start receiving events (default 0).
	Since int
	// WaitSeconds is how long the server should hold the connection open before
	// sending a keepalive/timeout event.  Must be >= 1 (default 15).
	WaitSeconds int
	// TraceID is an optional request-level trace identifier.
	TraceID string
}

// Listen streams incoming intents for the given agent address via SSE and sends each
// intent payload on the returned channel.  The channel is closed when ctx is cancelled
// or a non-recoverable error occurs; the error is returned by the companion errCh channel.
//
// The caller should select on both channels:
//
//	intents, errCh := client.Listen(ctx, "agent://acme/main/validator", axme.ListenOptions{})
//	for {
//	    select {
//	    case intent, ok := <-intents:
//	        if !ok { return }
//	        process(intent)
//	    case err := <-errCh:
//	        if err != nil { log.Fatal(err) }
//	    }
//	}
//
// The since cursor is advanced automatically, so reconnects replay from the last seen
// sequence number and no events are missed.
func (c *Client) Listen(
	ctx context.Context,
	address string,
	options ListenOptions,
) (<-chan map[string]any, <-chan error) {
	intents := make(chan map[string]any, 16)
	errCh := make(chan error, 1)

	waitSeconds := options.WaitSeconds
	if waitSeconds < 1 {
		waitSeconds = 15
	}

	pathPart := strings.TrimPrefix(strings.TrimSpace(address), "agent://")

	go func() {
		defer close(intents)
		defer close(errCh)

		nextSince := options.Since

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			events, err := c.fetchAgentIntentStream(ctx, pathPart, nextSince, waitSeconds, options.TraceID)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				errCh <- err
				return
			}

			for _, event := range events {
				if seq, ok := event["seq"]; ok {
					if seqNum, ok := seq.(float64); ok {
						s := int(seqNum)
						if s > nextSince {
							nextSince = s
						}
					}
				}
				select {
				case intents <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return intents, errCh
}

// fetchAgentIntentStream fetches one SSE batch from GET /v1/agents/{path}/intents/stream
// and returns the parsed intent payloads (keepalive events are filtered out).
func (c *Client) fetchAgentIntentStream(
	ctx context.Context,
	pathPart string,
	since int,
	waitSeconds int,
	traceID string,
) ([]map[string]any, error) {
	streamURL, err := url.Parse(fmt.Sprintf("%s/v1/agents/%s/intents/stream", c.baseURL, pathPart))
	if err != nil {
		return nil, err
	}
	params := streamURL.Query()
	params.Set("since", strconv.Itoa(since))
	params.Set("wait_seconds", strconv.Itoa(waitSeconds))
	streamURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.apiKey)
	if c.actorToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.actorToken)
	}
	req.Header.Set("Accept", "text/event-stream")
	if strings.TrimSpace(traceID) != "" {
		req.Header.Set("X-Trace-Id", traceID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: string(bodyBytes)}
	}

	return parseAgentSseEvents(string(bodyBytes)), nil
}

// parseAgentSseEvents parses an SSE response body and returns intent payloads.
// Events with event type "stream.timeout" (keepalives) are ignored.
func parseAgentSseEvents(body string) []map[string]any {
	var events []map[string]any
	lines := strings.Split(body, "\n")
	var currentEvent string
	var dataLines []string

	flush := func() {
		if strings.HasPrefix(currentEvent, "intent.") && len(dataLines) > 0 {
			raw := strings.Join(dataLines, "\n")
			var payload map[string]any
			if err := json.Unmarshal([]byte(raw), &payload); err == nil {
				events = append(events, payload)
			}
		}
		currentEvent = ""
		dataLines = nil
	}

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	flush()

	return events
}



func (c *Client) GetIntent(
	ctx context.Context,
	intentID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/v1/intents/%s", intentID),
		nil,
		nil,
		options,
	)
}

func (c *Client) ListIntentEvents(
	ctx context.Context,
	intentID string,
	since *int,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if since != nil && *since >= 0 {
		query["since"] = strconv.Itoa(*since)
	}
	return c.requestJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/v1/intents/%s/events", intentID),
		query,
		nil,
		options,
	)
}

func (c *Client) ResolveIntent(
	ctx context.Context,
	intentID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(options.OwnerAgent) != "" {
		query["owner_agent"] = options.OwnerAgent
	}
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/intents/%s/resolve", intentID),
		query,
		payload,
		options,
	)
}

func (c *Client) ResumeIntent(
	ctx context.Context,
	intentID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(options.OwnerAgent) != "" {
		query["owner_agent"] = options.OwnerAgent
	}
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/intents/%s/resume", intentID),
		query,
		payload,
		options,
	)
}

func (c *Client) UpdateIntentControls(
	ctx context.Context,
	intentID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(options.OwnerAgent) != "" {
		query["owner_agent"] = options.OwnerAgent
	}
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/intents/%s/controls", intentID),
		query,
		payload,
		options,
	)
}

func (c *Client) UpdateIntentPolicy(
	ctx context.Context,
	intentID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(options.OwnerAgent) != "" {
		query["owner_agent"] = options.OwnerAgent
	}
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/intents/%s/policy", intentID),
		query,
		payload,
		options,
	)
}

func (c *Client) CreateAccessRequest(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/access-requests", nil, payload, options)
}

func (c *Client) ListAccessRequests(
	ctx context.Context,
	orgID string,
	workspaceID string,
	state string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(orgID) != "" {
		query["org_id"] = orgID
	}
	if strings.TrimSpace(workspaceID) != "" {
		query["workspace_id"] = workspaceID
	}
	if strings.TrimSpace(state) != "" {
		query["state"] = state
	}
	return c.requestJSON(ctx, http.MethodGet, "/v1/access-requests", query, nil, options)
}

func (c *Client) GetAccessRequest(
	ctx context.Context,
	accessRequestID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/v1/access-requests/%s", accessRequestID),
		nil,
		nil,
		options,
	)
}

func (c *Client) ReviewAccessRequest(
	ctx context.Context,
	accessRequestID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/access-requests/%s/review", accessRequestID),
		nil,
		payload,
		options,
	)
}

func (c *Client) BindAlias(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/aliases", nil, payload, options)
}

func (c *Client) ListAliases(
	ctx context.Context,
	orgID string,
	workspaceID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		"/v1/aliases",
		map[string]string{"org_id": orgID, "workspace_id": workspaceID},
		nil,
		options,
	)
}

func (c *Client) RevokeAlias(
	ctx context.Context,
	aliasID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/aliases/%s/revoke", aliasID),
		nil,
		nil,
		options,
	)
}

func (c *Client) ResolveAlias(
	ctx context.Context,
	orgID string,
	workspaceID string,
	alias string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		"/v1/aliases/resolve",
		map[string]string{"org_id": orgID, "workspace_id": workspaceID, "alias": alias},
		nil,
		options,
	)
}

func (c *Client) DecideApproval(
	ctx context.Context,
	approvalID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/approvals/%s/decision", approvalID),
		nil,
		payload,
		options,
	)
}

func (c *Client) GetCapabilities(
	ctx context.Context,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodGet, "/v1/capabilities", nil, nil, options)
}

func (c *Client) ListInbox(
	ctx context.Context,
	ownerAgent string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(ownerAgent) != "" {
		query["owner_agent"] = ownerAgent
	}
	return c.requestJSON(ctx, http.MethodGet, "/v1/inbox", query, nil, options)
}

func (c *Client) GetInboxThread(
	ctx context.Context,
	threadID string,
	ownerAgent string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(ownerAgent) != "" {
		query["owner_agent"] = ownerAgent
	}
	return c.requestJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/v1/inbox/%s", threadID),
		query,
		nil,
		options,
	)
}

func (c *Client) ListInboxChanges(
	ctx context.Context,
	ownerAgent string,
	cursor string,
	limit *int,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(ownerAgent) != "" {
		query["owner_agent"] = ownerAgent
	}
	if strings.TrimSpace(cursor) != "" {
		query["cursor"] = cursor
	}
	if limit != nil && *limit >= 0 {
		query["limit"] = strconv.Itoa(*limit)
	}
	return c.requestJSON(ctx, http.MethodGet, "/v1/inbox/changes", query, nil, options)
}

func (c *Client) ReplyInboxThread(
	ctx context.Context,
	threadID string,
	message string,
	ownerAgent string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(ownerAgent) != "" {
		query["owner_agent"] = ownerAgent
	}
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/inbox/%s/reply", threadID),
		query,
		map[string]any{"message": message},
		options,
	)
}

func (c *Client) DelegateInboxThread(
	ctx context.Context,
	threadID string,
	payload map[string]any,
	ownerAgent string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(ownerAgent) != "" {
		query["owner_agent"] = ownerAgent
	}
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/inbox/%s/delegate", threadID),
		query,
		payload,
		options,
	)
}

func (c *Client) ApproveInboxThread(
	ctx context.Context,
	threadID string,
	payload map[string]any,
	ownerAgent string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(ownerAgent) != "" {
		query["owner_agent"] = ownerAgent
	}
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/inbox/%s/approve", threadID),
		query,
		payload,
		options,
	)
}

func (c *Client) RejectInboxThread(
	ctx context.Context,
	threadID string,
	payload map[string]any,
	ownerAgent string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(ownerAgent) != "" {
		query["owner_agent"] = ownerAgent
	}
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/inbox/%s/reject", threadID),
		query,
		payload,
		options,
	)
}

func (c *Client) DeleteInboxMessages(
	ctx context.Context,
	threadID string,
	payload map[string]any,
	ownerAgent string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(ownerAgent) != "" {
		query["owner_agent"] = ownerAgent
	}
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/inbox/%s/messages/delete", threadID),
		query,
		payload,
		options,
	)
}

func (c *Client) CreateInvite(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/invites/create", nil, payload, options)
}

func (c *Client) GetInvite(
	ctx context.Context,
	token string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodGet, fmt.Sprintf("/v1/invites/%s", token), nil, nil, options)
}

func (c *Client) AcceptInvite(
	ctx context.Context,
	token string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/invites/%s/accept", token),
		nil,
		payload,
		options,
	)
}

func (c *Client) CreateMediaUpload(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/media/create-upload", nil, payload, options)
}

func (c *Client) GetMediaUpload(
	ctx context.Context,
	uploadID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodGet, fmt.Sprintf("/v1/media/%s", uploadID), nil, nil, options)
}

func (c *Client) FinalizeMediaUpload(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/media/finalize-upload", nil, payload, options)
}

func (c *Client) UpsertSchema(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/schemas", nil, payload, options)
}

func (c *Client) GetSchema(
	ctx context.Context,
	semanticType string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodGet, fmt.Sprintf("/v1/schemas/%s", semanticType), nil, nil, options)
}

func (c *Client) UpsertWebhookSubscription(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/webhooks/subscriptions", nil, payload, options)
}

func (c *Client) ListWebhookSubscriptions(
	ctx context.Context,
	ownerAgent string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(ownerAgent) != "" {
		query["owner_agent"] = ownerAgent
	}
	return c.requestJSON(ctx, http.MethodGet, "/v1/webhooks/subscriptions", query, nil, options)
}

func (c *Client) DeleteWebhookSubscription(
	ctx context.Context,
	subscriptionID string,
	ownerAgent string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(ownerAgent) != "" {
		query["owner_agent"] = ownerAgent
	}
	return c.requestJSON(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("/v1/webhooks/subscriptions/%s", subscriptionID),
		query,
		nil,
		options,
	)
}

func (c *Client) PublishWebhookEvent(
	ctx context.Context,
	payload map[string]any,
	ownerAgent string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(ownerAgent) != "" {
		query["owner_agent"] = ownerAgent
	}
	return c.requestJSON(ctx, http.MethodPost, "/v1/webhooks/events", query, payload, options)
}

func (c *Client) ReplayWebhookEvent(
	ctx context.Context,
	eventID string,
	ownerAgent string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(ownerAgent) != "" {
		query["owner_agent"] = ownerAgent
	}
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/webhooks/events/%s/replay", eventID),
		query,
		nil,
		options,
	)
}

func (c *Client) CreateOrganization(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/organizations", nil, payload, options)
}

func (c *Client) GetOrganization(
	ctx context.Context,
	orgID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodGet, fmt.Sprintf("/v1/organizations/%s", orgID), nil, nil, options)
}

func (c *Client) UpdateOrganization(
	ctx context.Context,
	orgID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("/v1/organizations/%s", orgID),
		nil,
		payload,
		options,
	)
}

func (c *Client) CreateWorkspace(
	ctx context.Context,
	orgID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/organizations/%s/workspaces", orgID),
		nil,
		payload,
		options,
	)
}

func (c *Client) ListWorkspaces(
	ctx context.Context,
	orgID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/v1/organizations/%s/workspaces", orgID),
		nil,
		nil,
		options,
	)
}

func (c *Client) UpdateWorkspace(
	ctx context.Context,
	orgID string,
	workspaceID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("/v1/organizations/%s/workspaces/%s", orgID, workspaceID),
		nil,
		payload,
		options,
	)
}

func (c *Client) ListOrganizationMembers(
	ctx context.Context,
	orgID string,
	workspaceID string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{}
	if strings.TrimSpace(workspaceID) != "" {
		query["workspace_id"] = workspaceID
	}
	return c.requestJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/v1/organizations/%s/members", orgID),
		query,
		nil,
		options,
	)
}

func (c *Client) AddOrganizationMember(
	ctx context.Context,
	orgID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/organizations/%s/members", orgID),
		nil,
		payload,
		options,
	)
}

func (c *Client) UpdateOrganizationMember(
	ctx context.Context,
	orgID string,
	memberID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("/v1/organizations/%s/members/%s", orgID, memberID),
		nil,
		payload,
		options,
	)
}

func (c *Client) RemoveOrganizationMember(
	ctx context.Context,
	orgID string,
	memberID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("/v1/organizations/%s/members/%s", orgID, memberID),
		nil,
		nil,
		options,
	)
}

func (c *Client) UpdateQuota(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPatch, "/v1/quotas", nil, payload, options)
}

func (c *Client) GetQuota(
	ctx context.Context,
	orgID string,
	workspaceID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		"/v1/quotas",
		map[string]string{"org_id": orgID, "workspace_id": workspaceID},
		nil,
		options,
	)
}

func (c *Client) GetUsageSummary(
	ctx context.Context,
	orgID string,
	workspaceID string,
	window string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{"org_id": orgID, "workspace_id": workspaceID}
	if strings.TrimSpace(window) != "" {
		query["window"] = window
	}
	return c.requestJSON(ctx, http.MethodGet, "/v1/usage/summary", query, nil, options)
}

func (c *Client) GetUsageTimeseries(
	ctx context.Context,
	orgID string,
	workspaceID string,
	windowDays *int,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{"org_id": orgID, "workspace_id": workspaceID}
	if windowDays != nil && *windowDays >= 0 {
		query["window_days"] = strconv.Itoa(*windowDays)
	}
	return c.requestJSON(ctx, http.MethodGet, "/v1/usage/timeseries", query, nil, options)
}

func (c *Client) CreatePrincipal(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/principals", nil, payload, options)
}

func (c *Client) GetPrincipal(
	ctx context.Context,
	principalID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/v1/principals/%s", principalID),
		nil,
		nil,
		options,
	)
}

func (c *Client) RegisterRoutingEndpoint(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/routing/endpoints", nil, payload, options)
}

func (c *Client) ListRoutingEndpoints(
	ctx context.Context,
	orgID string,
	workspaceID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		"/v1/routing/endpoints",
		map[string]string{"org_id": orgID, "workspace_id": workspaceID},
		nil,
		options,
	)
}

func (c *Client) UpdateRoutingEndpoint(
	ctx context.Context,
	routeID string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("/v1/routing/endpoints/%s", routeID),
		nil,
		payload,
		options,
	)
}

func (c *Client) RemoveRoutingEndpoint(
	ctx context.Context,
	routeID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("/v1/routing/endpoints/%s", routeID),
		nil,
		nil,
		options,
	)
}

func (c *Client) ResolveRouting(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/routing/resolve", nil, payload, options)
}

func (c *Client) UpsertTransportBinding(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/transports/bindings", nil, payload, options)
}

func (c *Client) ListTransportBindings(
	ctx context.Context,
	orgID string,
	workspaceID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		"/v1/transports/bindings",
		map[string]string{"org_id": orgID, "workspace_id": workspaceID},
		nil,
		options,
	)
}

func (c *Client) RemoveTransportBinding(
	ctx context.Context,
	bindingID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("/v1/transports/bindings/%s", bindingID),
		nil,
		nil,
		options,
	)
}

func (c *Client) SubmitDelivery(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/deliveries", nil, payload, options)
}

func (c *Client) ListDeliveries(
	ctx context.Context,
	orgID string,
	workspaceID string,
	principalID string,
	status string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{"org_id": orgID, "workspace_id": workspaceID}
	if strings.TrimSpace(principalID) != "" {
		query["principal_id"] = principalID
	}
	if strings.TrimSpace(status) != "" {
		query["status"] = status
	}
	return c.requestJSON(ctx, http.MethodGet, "/v1/deliveries", query, nil, options)
}

func (c *Client) GetDelivery(
	ctx context.Context,
	deliveryID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/v1/deliveries/%s", deliveryID),
		nil,
		nil,
		options,
	)
}

func (c *Client) ReplayDelivery(
	ctx context.Context,
	deliveryID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/v1/deliveries/%s/replay", deliveryID),
		nil,
		nil,
		options,
	)
}

func (c *Client) UpdateBillingPlan(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPatch, "/v1/billing/plan", nil, payload, options)
}

func (c *Client) GetBillingPlan(
	ctx context.Context,
	orgID string,
	workspaceID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		"/v1/billing/plan",
		map[string]string{"org_id": orgID, "workspace_id": workspaceID},
		nil,
		options,
	)
}

func (c *Client) ListBillingInvoices(
	ctx context.Context,
	orgID string,
	workspaceID string,
	billingStatus string,
	options RequestOptions,
) (map[string]any, error) {
	query := map[string]string{"org_id": orgID, "workspace_id": workspaceID}
	if strings.TrimSpace(billingStatus) != "" {
		query["status"] = billingStatus
	}
	return c.requestJSON(ctx, http.MethodGet, "/v1/billing/invoices", query, nil, options)
}

func (c *Client) GetBillingInvoice(
	ctx context.Context,
	invoiceID string,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/v1/billing/invoices/%s", invoiceID),
		nil,
		nil,
		options,
	)
}

// generateUUID returns a new random UUID v4 string.
func generateUUID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}

// terminalStatuses is the set of intent statuses that indicate the intent lifecycle is over.
var terminalStatuses = map[string]bool{
	"COMPLETED": true,
	"FAILED":    true,
	"CANCELED":  true,
	"TIMED_OUT": true,
}

// SendIntent is a convenience wrapper around CreateIntent that auto-generates a
// correlation_id (UUID) if one is not already present in the payload.  Returns
// the intent_id string from the response.
func (c *Client) SendIntent(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (string, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	if _, ok := payload["correlation_id"]; !ok {
		payload["correlation_id"] = generateUUID()
	}

	result, err := c.CreateIntent(ctx, payload, options)
	if err != nil {
		return "", err
	}

	intentID, ok := result["intent_id"].(string)
	if !ok {
		return "", fmt.Errorf("response missing intent_id string field")
	}
	return intentID, nil
}

// ObserveOptions controls how Observe polls intent lifecycle events.
type ObserveOptions struct {
	// Since is the minimum seq value from which to start receiving events (default 0).
	Since int
	// WaitSeconds is how long the server should hold the connection open before
	// sending a keepalive/timeout event.  Must be >= 1 (default 15).
	WaitSeconds int
	// TimeoutMs is the maximum time in milliseconds to observe before stopping.
	// 0 means no timeout.
	TimeoutMs int
	// TraceID is an optional request-level trace identifier.
	TraceID string
}

// Observe yields intent lifecycle events via a channel by polling ListIntentEvents.
// The channel is closed when the intent reaches a terminal status (COMPLETED, FAILED,
// CANCELED, TIMED_OUT), when ctx is cancelled, or when the timeout expires.  Errors
// are sent on the companion error channel.
func (c *Client) Observe(
	ctx context.Context,
	intentID string,
	options ObserveOptions,
) (<-chan map[string]any, <-chan error) {
	events := make(chan map[string]any, 16)
	errCh := make(chan error, 1)

	waitSeconds := options.WaitSeconds
	if waitSeconds < 1 {
		waitSeconds = 15
	}

	go func() {
		defer close(events)
		defer close(errCh)

		var deadline <-chan time.Time
		if options.TimeoutMs > 0 {
			timer := time.NewTimer(time.Duration(options.TimeoutMs) * time.Millisecond)
			defer timer.Stop()
			deadline = timer.C
		}

		nextSince := options.Since
		reqOpts := RequestOptions{TraceID: options.TraceID}

		for {
			select {
			case <-ctx.Done():
				return
			case <-deadline:
				return
			default:
			}

			sinceVal := nextSince
			result, err := c.ListIntentEvents(ctx, intentID, &sinceVal, reqOpts)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				errCh <- err
				return
			}

			// Extract the events array from the response.
			items, _ := result["events"].([]any)
			for _, item := range items {
				event, ok := item.(map[string]any)
				if !ok {
					continue
				}

				// Advance the cursor.
				if seq, ok := event["seq"]; ok {
					if seqNum, ok := seq.(float64); ok {
						s := int(seqNum)
						if s > nextSince {
							nextSince = s
						}
					}
				}

				select {
				case events <- event:
				case <-ctx.Done():
					return
				}

				// Check for terminal status.
				if status, ok := event["status"].(string); ok && terminalStatuses[status] {
					return
				}
			}

			// Pause before next poll.
			select {
			case <-ctx.Done():
				return
			case <-deadline:
				return
			case <-time.After(1 * time.Second):
			}
		}
	}()

	return events, errCh
}

// WaitFor blocks until the intent reaches a terminal state (COMPLETED, FAILED,
// CANCELED, TIMED_OUT) and returns the terminal event.
func (c *Client) WaitFor(
	ctx context.Context,
	intentID string,
	options ObserveOptions,
) (map[string]any, error) {
	eventsCh, errCh := c.Observe(ctx, intentID, options)

	var lastEvent map[string]any
	for {
		select {
		case event, ok := <-eventsCh:
			if !ok {
				// Channel closed — return whatever we have.
				if lastEvent != nil {
					return lastEvent, nil
				}
				return nil, fmt.Errorf("observe channel closed without terminal event")
			}
			lastEvent = event
			if status, ok := event["status"].(string); ok && terminalStatuses[status] {
				return event, nil
			}
		case err := <-errCh:
			if err != nil {
				return nil, err
			}
		}
	}
}

// Health performs a GET request to /v1/health and returns the response body.
func (c *Client) Health(
	ctx context.Context,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodGet, "/v1/health", nil, nil, options)
}

const mcpEndpointPath = "/mcp"

// mcpRequest sends a JSON-RPC 2.0 request to the MCP endpoint and returns the
// result field from the response.  If the response contains an error field, it
// is returned as an error.
func (c *Client) mcpRequest(
	ctx context.Context,
	method string,
	params map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	rpcID := generateUUID()
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      rpcID,
		"method":  method,
		"params":  params,
	}

	resp, err := c.requestJSON(ctx, http.MethodPost, mcpEndpointPath, nil, body, options)
	if err != nil {
		return nil, err
	}

	// Check for JSON-RPC error.
	if rpcErr, ok := resp["error"]; ok {
		errMap, _ := rpcErr.(map[string]any)
		if errMap != nil {
			msg, _ := errMap["message"].(string)
			code, _ := errMap["code"].(float64)
			return nil, fmt.Errorf("mcp error %d: %s", int(code), msg)
		}
		return nil, fmt.Errorf("mcp error: %v", rpcErr)
	}

	if result, ok := resp["result"].(map[string]any); ok {
		return result, nil
	}

	return resp, nil
}

// McpInitialize sends a JSON-RPC 2.0 "initialize" request to the MCP endpoint.
func (c *Client) McpInitialize(
	ctx context.Context,
	options RequestOptions,
) (map[string]any, error) {
	return c.mcpRequest(ctx, "initialize", map[string]any{}, options)
}

// McpListTools sends a JSON-RPC 2.0 "tools/list" request to the MCP endpoint.
func (c *Client) McpListTools(
	ctx context.Context,
	options RequestOptions,
) (map[string]any, error) {
	return c.mcpRequest(ctx, "tools/list", map[string]any{}, options)
}

// McpCallToolOptions controls a tools/call MCP request.
type McpCallToolOptions struct {
	Arguments      map[string]any
	OwnerAgent     string
	IdempotencyKey string
	TraceID        string
}

// McpCallTool sends a JSON-RPC 2.0 "tools/call" request to the MCP endpoint.
func (c *Client) McpCallTool(
	ctx context.Context,
	name string,
	options McpCallToolOptions,
) (map[string]any, error) {
	params := map[string]any{
		"name": name,
	}
	if options.Arguments != nil {
		params["arguments"] = options.Arguments
	}

	reqOpts := RequestOptions{
		OwnerAgent:     options.OwnerAgent,
		IdempotencyKey: options.IdempotencyKey,
		TraceID:        options.TraceID,
	}

	return c.mcpRequest(ctx, "tools/call", params, reqOpts)
}

func (c *Client) requestJSON(
	ctx context.Context,
	method string,
	path string,
	query map[string]string,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	endpointURL, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, err
	}

	if len(query) > 0 {
		params := endpointURL.Query()
		for k, v := range query {
			if strings.TrimSpace(v) != "" {
				params.Set(k, v)
			}
		}
		endpointURL.RawQuery = params.Encode()
	}

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpointURL.String(), body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("x-api-key", c.apiKey)
	if strings.TrimSpace(options.Authorization) != "" {
		request.Header.Set("Authorization", options.Authorization)
	} else if c.actorToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.actorToken)
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(options.XOwnerAgent) != "" {
		request.Header.Set("x-owner-agent", options.XOwnerAgent)
	}
	if strings.TrimSpace(options.IdempotencyKey) != "" {
		request.Header.Set("Idempotency-Key", options.IdempotencyKey)
	}
	if strings.TrimSpace(options.TraceID) != "" {
		request.Header.Set("X-Trace-Id", options.TraceID)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, &HTTPError{StatusCode: response.StatusCode, Body: string(responseBody)}
	}

	if len(responseBody) == 0 {
		return map[string]any{}, nil
	}

	var out map[string]any
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}
