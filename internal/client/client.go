// Package client is the thin, impure shell around the LINE Messaging API and
// LIFF server API HTTP endpoints. It converts between internal/lineapi's
// pure domain types and the wire JSON format, and owns all network I/O:
// authentication, retries, and error decoding. Business logic (validation,
// diffing, hashing) lives in internal/lineapi instead, so it can be tested
// without an HTTP server.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/cuzic/terraform-provider-line/internal/lineapi"
)

const (
	// DefaultAPIBaseURL is the base URL for most Messaging API and LIFF
	// server API endpoints.
	DefaultAPIBaseURL = "https://api.line.me"
	// DefaultDataBaseURL is the base URL LINE uses for binary/blob
	// endpoints (e.g. rich menu image upload and download).
	DefaultDataBaseURL = "https://api-data.line.me"

	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
)

// Logf is a redaction-aware logging hook. The client never logs the bearer
// token itself; callers (the provider package, wiring this to tflog) get a
// pre-redacted message.
type Logf func(ctx context.Context, format string, args ...any)

// Client is a LINE API client covering the subset of the Messaging API and
// LIFF server API this provider needs: webhook endpoint, LIFF apps, and rich
// menus.
type Client struct {
	httpClient  *http.Client
	apiBaseURL  string
	dataBaseURL string
	token       string
	userAgent   string
	maxRetries  int
	log         Logf
}

// Option configures a Client constructed with New.
type Option func(*Client)

func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }
func WithAPIBaseURL(u string) Option       { return func(c *Client) { c.apiBaseURL = u } }
func WithDataBaseURL(u string) Option      { return func(c *Client) { c.dataBaseURL = u } }
func WithUserAgent(ua string) Option       { return func(c *Client) { c.userAgent = ua } }
func WithLogger(f Logf) Option             { return func(c *Client) { c.log = f } }
func WithMaxRetries(n int) Option          { return func(c *Client) { c.maxRetries = n } }

// New creates a Client authenticated with the given channel access token.
func New(token string, opts ...Option) *Client {
	c := &Client{
		httpClient:  &http.Client{Timeout: defaultTimeout},
		apiBaseURL:  DefaultAPIBaseURL,
		dataBaseURL: DefaultDataBaseURL,
		token:       token,
		userAgent:   "terraform-provider-line",
		maxRetries:  defaultMaxRetries,
		log:         func(context.Context, string, ...any) {},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// APIError represents a non-2xx response from the LINE API.
type APIError struct {
	StatusCode int
	Message    string
	Details    []APIErrorDetail
	RawBody    string
}

// APIErrorDetail is one entry of a LINE API error response's "details" array.
type APIErrorDetail struct {
	Message  string `json:"message"`
	Property string `json:"property"`
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("line api: status %d", e.StatusCode)
	}
	return fmt.Sprintf("line api: status %d: %s", e.StatusCode, e.Message)
}

// IsNotFound reports whether err is an *APIError with a 404 status code.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

type errorResponse struct {
	Message string           `json:"message"`
	Details []APIErrorDetail `json:"details"`
}

// doJSON performs a request with a JSON-encoded body (or no body when
// reqBody is nil) and returns the raw response body on success.
func (c *Client) doJSON(ctx context.Context, method, baseURL, path string, reqBody any) ([]byte, error) {
	var bodyBytes []byte
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyBytes = b
	}
	return c.do(ctx, method, baseURL, path, bodyBytes, "application/json")
}

// do performs an HTTP request, retrying only on HTTP 429 (rate limited) up
// to maxRetries times, honoring a Retry-After header when present.
//
// Deliberately not retried: transport-level errors (a dropped connection or
// client-side timeout doesn't tell us whether the server ever received or
// processed the request) and any other status code. Several of this
// client's callers issue non-idempotent POSTs (AddLiffApp, CreateRichMenu,
// UploadRichMenuImage) — blindly retrying after an ambiguous transport error
// risks creating the same object twice, which is worse than surfacing the
// error and letting the caller decide.
func (c *Client) do(ctx context.Context, method, baseURL, path string, body []byte, contentType string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			c.log(ctx, "retrying %s %s%s (attempt %d/%d) after error: %v", method, baseURL, path, attempt, c.maxRetries, lastErr)
		}

		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reqBody)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("User-Agent", c.userAgent)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		c.log(ctx, "%s", lineapi.RedactToken(fmt.Sprintf("-> %s %s%s", method, baseURL, path), c.token))

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("%s %s%s: %w", method, baseURL, path, err)
		}

		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read response body: %w", readErr)
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < c.maxRetries {
			lastErr = decodeAPIError(resp.StatusCode, respBody)
			wait := retryAfterDelay(resp.Header.Get("Retry-After"), attempt)
			c.log(ctx, "rate limited by LINE API, waiting %s before retry", wait)
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}

		if resp.StatusCode >= 300 {
			return nil, decodeAPIError(resp.StatusCode, respBody)
		}

		return respBody, nil
	}

	return nil, lastErr
}

func decodeAPIError(statusCode int, body []byte) error {
	var er errorResponse
	_ = json.Unmarshal(body, &er) // best-effort; RawBody preserves the original either way
	return &APIError{
		StatusCode: statusCode,
		Message:    er.Message,
		Details:    er.Details,
		RawBody:    string(body),
	}
}

// retryAfterDelay computes how long to wait before retrying a 429 response:
// the server-provided Retry-After (in seconds) when present and valid,
// otherwise exponential backoff starting at 1s.
func retryAfterDelay(retryAfterHeader string, attempt int) time.Duration {
	if retryAfterHeader != "" {
		if secs, err := strconv.Atoi(retryAfterHeader); err == nil && secs >= 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Duration(1<<uint(attempt)) * time.Second
}
