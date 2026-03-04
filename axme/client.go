package axme

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type ClientConfig struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
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
	httpClient *http.Client
}

func NewClient(config ClientConfig) (*Client, error) {
	baseURL := strings.TrimSpace(config.BaseURL)
	apiKey := strings.TrimSpace(config.APIKey)

	if baseURL == "" {
		return nil, fmt.Errorf("baseURL is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
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

func (c *Client) CreateIntent(
	ctx context.Context,
	payload map[string]any,
	options RequestOptions,
) (map[string]any, error) {
	return c.requestJSON(ctx, http.MethodPost, "/v1/intents", nil, payload, options)
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

	if strings.TrimSpace(options.Authorization) != "" {
		request.Header.Set("Authorization", options.Authorization)
	} else {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
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
