package mpesa

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetAccessToken_Success(t *testing.T) {
	consumerKey := "test_key"
	consumerSecret := "test_secret"
	expectedToken := "test_access_token_12345"

	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		if r.URL.Path != "/oauth/v1/generate" {
			t.Errorf("expected path /oauth/v1/generate, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("grant_type") != "client_credentials" {
			t.Errorf("expected grant_type=client_credentials, got %s", r.URL.Query().Get("grant_type"))
		}

		authHeader := r.Header.Get("Authorization")
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(consumerKey+":"+consumerSecret))
		if authHeader != expectedAuth {
			t.Errorf("expected Auth header %s, got %s", expectedAuth, authHeader)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "` + expectedToken + `", "expires_in": "3599"}`))
	}))
	defer server.Close()

	tm := NewTokenManager(server.Client(), consumerKey, consumerSecret, server.URL)

	token, err := tm.GetAccessToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token != expectedToken {
		t.Errorf("expected token %s, got %s", expectedToken, token)
	}

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected 1 HTTP request, got %d", requestCount)
	}
}

func TestGetAccessToken_UnexportedMethod(t *testing.T) {
	consumerKey := "test_key"
	consumerSecret := "test_secret"
	expectedToken := "unexported_token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "` + expectedToken + `", "expires_in": "3599"}`))
	}))
	defer server.Close()

	tm := NewTokenManager(server.Client(), consumerKey, consumerSecret, server.URL)

	token, err := tm.getAccessToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error calling unexported getAccessToken: %v", err)
	}

	if token != expectedToken {
		t.Errorf("expected token %s, got %s", expectedToken, token)
	}
}

func TestGetAccessToken_Caching(t *testing.T) {
	consumerKey := "test_key"
	consumerSecret := "test_secret"
	expectedToken := "cached_token_abc"

	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "` + expectedToken + `", "expires_in": "3599"}`))
	}))
	defer server.Close()

	tm := NewTokenManager(server.Client(), consumerKey, consumerSecret, server.URL)

	// First call - fetches token from server
	token1, err := tm.GetAccessToken(context.Background())
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Second call - should return cached token
	token2, err := tm.GetAccessToken(context.Background())
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if token1 != token2 {
		t.Errorf("expected cached token %s, got %s", token1, token2)
	}

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected 1 HTTP request due to caching, got %d", requestCount)
	}
}

func TestGetAccessToken_RefreshOnExpiry(t *testing.T) {
	consumerKey := "test_key"
	consumerSecret := "test_secret"

	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if count == 1 {
			_, _ = w.Write([]byte(`{"access_token": "token_1", "expires_in": "1"}`))
		} else {
			_, _ = w.Write([]byte(`{"access_token": "token_2", "expires_in": "3599"}`))
		}
	}))
	defer server.Close()

	tm := NewTokenManager(server.Client(), consumerKey, consumerSecret, server.URL)
	tm.expiryBuffer = 100 * time.Millisecond

	// First call returns token_1
	token1, err := tm.GetAccessToken(context.Background())
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if token1 != "token_1" {
		t.Errorf("expected token_1, got %s", token1)
	}

	// Manually expire token
	tm.mu.Lock()
	tm.expiry = time.Now().Add(-1 * time.Second)
	tm.mu.Unlock()

	// Next call should fetch a fresh token (token_2)
	token2, err := tm.GetAccessToken(context.Background())
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if token2 != "token_2" {
		t.Errorf("expected token_2 after refresh, got %s", token2)
	}

	if atomic.LoadInt32(&requestCount) != 2 {
		t.Errorf("expected 2 HTTP requests, got %d", requestCount)
	}
}

func TestGetAccessToken_Concurrency(t *testing.T) {
	consumerKey := "test_key"
	consumerSecret := "test_secret"

	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		time.Sleep(10 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "concurrent_token", "expires_in": "3599"}`))
	}))
	defer server.Close()

	tm := NewTokenManager(server.Client(), consumerKey, consumerSecret, server.URL)

	var wg sync.WaitGroup
	workers := 20
	tokens := make([]string, workers)
	errs := make([]error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tok, err := tm.GetAccessToken(context.Background())
			tokens[idx] = tok
			errs[idx] = err
		}(i)
	}

	wg.Wait()

	for i := 0; i < workers; i++ {
		if errs[i] != nil {
			t.Errorf("worker %d failed: %v", i, errs[i])
		}
		if tokens[i] != "concurrent_token" {
			t.Errorf("worker %d got token %s, expected concurrent_token", i, tokens[i])
		}
	}

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected only 1 request due to lock double-check, got %d", requestCount)
	}
}

func TestGetAccessToken_MissingCredentials(t *testing.T) {
	tm := NewTokenManager(nil, "", "", "http://localhost")
	_, err := tm.GetAccessToken(context.Background())
	if err == nil {
		t.Error("expected error for missing credentials, got nil")
	}
}

func TestGetAccessToken_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errorMessage": "Invalid Credentials"}`))
	}))
	defer server.Close()

	tm := NewTokenManager(server.Client(), "key", "secret", server.URL)
	_, err := tm.GetAccessToken(context.Background())
	if err == nil {
		t.Error("expected error for 401 response, got nil")
	}
}

func TestGetAccessToken_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid_json_body`))
	}))
	defer server.Close()

	tm := NewTokenManager(server.Client(), "key", "secret", server.URL)
	_, err := tm.GetAccessToken(context.Background())
	if err == nil {
		t.Error("expected error for invalid JSON body, got nil")
	}
}

func TestGetAccessToken_EmptyToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "", "expires_in": "3600"}`))
	}))
	defer server.Close()

	tm := NewTokenManager(server.Client(), "key", "secret", server.URL)
	_, err := tm.GetAccessToken(context.Background())
	if err == nil {
		t.Error("expected error for empty access token, got nil")
	}
}

func TestGetAccessToken_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel context immediately

	tm := NewTokenManager(nil, "key", "secret", "http://localhost")
	_, err := tm.GetAccessToken(ctx)
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}

func TestAuthResponse_UnmarshalJSON_NumericExpiresIn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "num_token", "expires_in": 3600}`))
	}))
	defer server.Close()

	tmServer := NewTokenManager(server.Client(), "key", "secret", server.URL)
	tok, err := tmServer.GetAccessToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "num_token" {
		t.Errorf("expected num_token, got %s", tok)
	}
}

func TestNewTokenManager_NilClient(t *testing.T) {
	tm := NewTokenManager(nil, "key", "secret", "http://localhost")
	if tm.client == nil {
		t.Error("expected default HTTP client when nil is passed, got nil")
	}
}
