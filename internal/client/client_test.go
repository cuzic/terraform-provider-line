package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientDoJSON_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-token")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New("test-token", WithAPIBaseURL(srv.URL), WithMaxRetries(0))
	body, err := c.doJSON(context.Background(), http.MethodGet, srv.URL, "/anything", nil)
	if err != nil {
		t.Fatalf("doJSON returned error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestClientDoJSON_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	c := New("test-token", WithAPIBaseURL(srv.URL))
	_, err := c.doJSON(context.Background(), http.MethodGet, srv.URL, "/missing", nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected IsNotFound(err) to be true, err = %v", err)
	}
}

func TestClientDoJSON_RetriesOn429ThenSucceeds(t *testing.T) {
	t.Parallel()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New("test-token", WithAPIBaseURL(srv.URL), WithMaxRetries(2))
	body, err := c.doJSON(context.Background(), http.MethodGet, srv.URL, "/rate-limited", nil)
	if err != nil {
		t.Fatalf("doJSON returned error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", body)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected exactly 2 calls (1 rate-limited + 1 success), got %d", got)
	}
}

func TestClientDoJSON_GivesUpAfterMaxRetries(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := New("test-token", WithAPIBaseURL(srv.URL), WithMaxRetries(1))
	_, err := c.doJSON(context.Background(), http.MethodGet, srv.URL, "/always-limited", nil)
	if err == nil {
		t.Fatal("expected an error after exhausting retries, got nil")
	}
}

func TestClientLogging_NeverLeaksToken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	var logs []string
	c := New("super-secret-token", WithAPIBaseURL(srv.URL), WithLogger(func(_ context.Context, format string, args ...any) {
		logs = append(logs, format)
	}))

	if _, err := c.doJSON(context.Background(), http.MethodGet, srv.URL, "/anything", nil); err != nil {
		t.Fatalf("doJSON returned error: %v", err)
	}

	for _, line := range logs {
		if strings.Contains(line, "super-secret-token") {
			t.Fatalf("log line leaked the token: %q", line)
		}
	}
}

func TestRetryAfterDelay(t *testing.T) {
	t.Parallel()

	if got := retryAfterDelay("2", 0); got != 2*time.Second {
		t.Fatalf("retryAfterDelay(\"2\", 0) = %v, want 2s", got)
	}
	if got := retryAfterDelay("", 0); got != 1*time.Second {
		t.Fatalf("retryAfterDelay(\"\", 0) = %v, want 1s", got)
	}
	if got := retryAfterDelay("", 2); got != 4*time.Second {
		t.Fatalf("retryAfterDelay(\"\", 2) = %v, want 4s", got)
	}
	if got := retryAfterDelay("not-a-number", 1); got != 2*time.Second {
		t.Fatalf("retryAfterDelay(\"not-a-number\", 1) = %v, want fallback 2s", got)
	}
}
