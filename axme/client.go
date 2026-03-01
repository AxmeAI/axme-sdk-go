package axme

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
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
