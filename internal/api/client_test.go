package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestClient_Get(t *testing.T) {
	type response struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		if r.URL.Path != "/test/resource" {
			t.Errorf("expected path /test/resource, got %s", r.URL.Path)
		}

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer token, got %q", r.Header.Get("Authorization"))
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept application/json, got %q", r.Header.Get("Accept"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{ID: "123", Name: "test"})
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	var result response

	err := client.Get(context.Background(), "/test/resource", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "123" {
		t.Errorf("expected ID '123', got %q", result.ID)
	}

	if result.Name != "test" {
		t.Errorf("expected Name 'test', got %q", result.Name)
	}
}

func TestClient_Post(t *testing.T) {
	type requestBody struct {
		CustomerRef map[string]string `json:"CustomerRef"`
	}

	type response struct {
		ID string `json:"id"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{ID: "new-123"})
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	req := requestBody{CustomerRef: map[string]string{"value": "42"}}

	var result response

	err := client.Post(context.Background(), "/create", req, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "new-123" {
		t.Errorf("expected ID 'new-123', got %q", result.ID)
	}
}

func TestClient_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	err := client.Delete(context.Background(), "/resource/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_APIError_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid token",
		})
	}))
	defer server.Close()

	// Client without refresh token, so no auto-refresh
	client := NewClient("bad-token", WithBaseURL(server.URL))

	var result map[string]any

	err := client.Get(context.Background(), "/test", &result)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}

	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
}

func TestClient_APIError_QuickBooksFault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Fault": map[string]any{
				"Error": []map[string]string{
					{
						"Message": "Object Not Found",
						"Detail":  "Something went wrong",
						"code":    "610",
					},
				},
				"type": "ValidationFault",
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	var result map[string]any

	err := client.Get(context.Background(), "/test", &result)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}

	if apiErr.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}

	expected := "Object Not Found: Something went wrong"
	if apiErr.Message != expected {
		t.Errorf("expected message %q, got %q", expected, apiErr.Message)
	}
}

func TestClient_APIError_500_NoJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	var result map[string]any

	err := client.Get(context.Background(), "/test", &result)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}

	if apiErr.Message != "Internal Server Error" {
		t.Errorf("expected 'Internal Server Error', got %q", apiErr.Message)
	}
}

func TestClient_AutoRefresh(t *testing.T) {
	var mu sync.Mutex

	callCount := 0

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST for token refresh, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    3600,
		})
	}))
	defer tokenServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		currentCall := callCount
		mu.Unlock()

		if currentCall == 1 {
			// First call: return 401 to trigger refresh
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "expired"})

			return
		}

		// Second call: verify new token
		if r.Header.Get("Authorization") != "Bearer new-access-token" {
			t.Errorf("expected new token, got %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer apiServer.Close()

	client := NewClient("old-token",
		WithBaseURL(apiServer.URL),
		WithOAuth("client-id", "client-secret", "refresh-token", tokenServer.URL),
	)

	var result map[string]string

	err := client.Get(context.Background(), "/test", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", result["status"])
	}

	mu.Lock()
	if callCount != 2 {
		t.Errorf("expected 2 API calls (initial + retry), got %d", callCount)
	}
	mu.Unlock()
}

func TestClient_Put(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	var result map[string]string

	err := client.Put(context.Background(), "/resource/123", map[string]string{"name": "new"}, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["status"] != "updated" {
		t.Errorf("expected status 'updated', got %q", result["status"])
	}
}

func TestClient_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "value" {
			t.Errorf("expected X-Custom header 'value', got %q", r.Header.Get("X-Custom"))
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	resp, err := client.Do(context.Background(), Request{
		Method:  http.MethodGet,
		Path:    "/test",
		Headers: map[string]string{"X-Custom": "value"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp.Body.Close()
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{StatusCode: 404, Message: "Not Found"}
	expected := "API error (404): Not Found"

	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestWithTimeout(t *testing.T) {
	client := NewClient("key")
	WithTimeout(5000000000)(client) // 5 seconds

	if client.httpClient.Timeout != 5000000000 {
		t.Errorf("expected timeout 5s, got %v", client.httpClient.Timeout)
	}
}
