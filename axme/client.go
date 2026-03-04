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
