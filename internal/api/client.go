package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	errTokenRefreshFailed = errors.New("token refresh failed")
	errNoRefreshToken     = errors.New("no refresh token available")
)

// TokenStore is called by the client to read/write OAuth tokens.
type TokenStore interface {
	GetCredential(key string) (string, error)
	SetCredential(key, value string) error
}

// Client is an HTTP client with OAuth 2.0 Bearer token authentication
// and automatic token refresh.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	userAgent    string
	accessToken  string
	tokenURL     string
	clientID     string
	clientSecret string
	refreshToken string
	tokenStore   TokenStore
	mu           sync.Mutex
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithBaseURL sets the API base URL.
func WithBaseURL(u string) ClientOption {
	return func(c *Client) {
		c.baseURL = u
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) ClientOption {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithTokenStore sets the token store for persisting refreshed tokens.
func WithTokenStore(store TokenStore) ClientOption {
	return func(c *Client) {
		c.tokenStore = store
	}
}

// WithOAuth sets the OAuth credentials for automatic token refresh.
func WithOAuth(clientID, clientSecret, refreshToken, tokenURL string) ClientOption {
	return func(c *Client) {
		c.clientID = clientID
		c.clientSecret = clientSecret
		c.refreshToken = refreshToken
		c.tokenURL = tokenURL
	}
}

// NewClient creates a new API client with Bearer token auth.
func NewClient(accessToken string, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		accessToken: accessToken,
		userAgent:   "quickbooks-cli/1.0",
		baseURL:     "https://quickbooks.api.intuit.com",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Request describes an HTTP request.
type Request struct {
	Method  string
	Path    string
	Body    any
	Headers map[string]string
}

// Do executes an HTTP request with Bearer token auth.
func (c *Client) Do(ctx context.Context, req Request) (*http.Response, error) {
	var bodyReader io.Reader

	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}

		bodyReader = bytes.NewReader(bodyBytes)
	}

	reqURL := c.baseURL + req.Path

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", c.userAgent)

	// Set Bearer token
	if c.accessToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	// Auto-refresh on 401 if we have refresh credentials
	if resp.StatusCode == http.StatusUnauthorized && c.refreshToken != "" {
		_ = resp.Body.Close()

		if refreshErr := c.RefreshAccessToken(ctx); refreshErr != nil {
			return nil, fmt.Errorf("%w: %w", errTokenRefreshFailed, refreshErr)
		}

		// Retry the request with the new token
		return c.retryRequest(ctx, req)
	}

	return resp, nil
}

func (c *Client) retryRequest(ctx context.Context, req Request) (*http.Response, error) {
	var bodyReader io.Reader

	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}

		bodyReader = bytes.NewReader(bodyBytes)
	}

	reqURL := c.baseURL + req.Path

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", c.userAgent)

	if c.accessToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return resp, nil
}

// RefreshAccessToken exchanges the refresh token for a new access token.
func (c *Client) RefreshAccessToken(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.refreshToken == "" {
		return errNoRefreshToken
	}

	if c.tokenURL == "" {
		return fmt.Errorf("%w: token URL not configured", errTokenRefreshFailed)
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {c.refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Basic auth with client credentials
	credentials := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	req.Header.Set("Authorization", "Basic "+credentials)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("%w: status %d: %s", errTokenRefreshFailed, resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken

	if tokenResp.RefreshToken != "" {
		c.refreshToken = tokenResp.RefreshToken
	}

	// Persist tokens to store if available
	if c.tokenStore != nil {
		if storeErr := c.tokenStore.SetCredential("access_token", tokenResp.AccessToken); storeErr != nil {
			return fmt.Errorf("store access token: %w", storeErr)
		}

		expiry := fmt.Sprintf("%d", time.Now().Add(time.Duration(tokenResp.ExpiresIn)*time.Second).Unix())

		if storeErr := c.tokenStore.SetCredential("access_token_expiry", expiry); storeErr != nil {
			return fmt.Errorf("store token expiry: %w", storeErr)
		}

		if tokenResp.RefreshToken != "" {
			if storeErr := c.tokenStore.SetCredential("refresh_token", tokenResp.RefreshToken); storeErr != nil {
				return fmt.Errorf("store refresh token: %w", storeErr)
			}
		}
	}

	return nil
}

// TokenResponse is the OAuth token endpoint response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Get performs a GET request and decodes the JSON response.
func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.doJSON(ctx, Request{Method: http.MethodGet, Path: path}, result)
}

// Post performs a POST request with a JSON body and decodes the response.
func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	return c.doJSON(ctx, Request{Method: http.MethodPost, Path: path, Body: body}, result)
}

// Put performs a PUT request with a JSON body and decodes the response.
func (c *Client) Put(ctx context.Context, path string, body, result any) error {
	return c.doJSON(ctx, Request{Method: http.MethodPut, Path: path, Body: body}, result)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	resp, err := c.Do(ctx, Request{Method: http.MethodDelete, Path: path})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}

	return nil
}

func (c *Client) doJSON(ctx context.Context, req Request, result any) error {
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
}

func parseAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	// QuickBooks returns errors in a Fault structure
	var qbErr struct {
		Fault struct {
			Error []struct {
				Message string `json:"Message"`
				Detail  string `json:"Detail"`
				Code    string `json:"code"`
			} `json:"Error"`
			Type string `json:"type"`
		} `json:"Fault"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if json.Unmarshal(body, &qbErr) == nil {
		// Check QuickBooks Fault format first
		if len(qbErr.Fault.Error) > 0 {
			msg := qbErr.Fault.Error[0].Message
			if qbErr.Fault.Error[0].Detail != "" {
				msg += ": " + qbErr.Fault.Error[0].Detail
			}

			return &APIError{StatusCode: resp.StatusCode, Message: msg}
		}

		// Check standard error fields
		msg := qbErr.Message
		if msg == "" {
			msg = qbErr.Error
		}

		if msg != "" {
			return &APIError{StatusCode: resp.StatusCode, Message: msg}
		}
	}

	// Fallback to status text
	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    http.StatusText(resp.StatusCode),
	}
}
