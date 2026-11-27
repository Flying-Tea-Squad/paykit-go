package airtel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
)

func newAuthTestManager(server *httptest.Server) *TokenManager {
	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{
		IdempotencyPolicy: paykit.IdempotencyPolicyOptional,
	})
	client.SetTransport(server.Client().Transport)
	return NewTokenManager(client, "test-id", "test-secret", server.URL)
}

func TestGetAccessTokenSuccess(t *testing.T) {
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)

		if r.Method != http.MethodPost || r.URL.Path != "/auth/oauth2/token" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected JSON content type")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("expected JSON accept header")
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Error("expected idempotency key")
		}

		var body struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			GrantType    string `json:"grant_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.ClientID != "test-id" || body.ClientSecret != "test-secret" ||
			body.GrantType != "client_credentials" {
			t.Error("unexpected OAuth credentials or grant type")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"airtel-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer server.Close()

	tm := newAuthTestManager(server)
	token, err := tm.GetAccessToken(context.Background())
	if err != nil || token != "airtel-token" {
		t.Fatalf("token=%q, error=%v", token, err)
	}
	if requests.Load() != 1 {
		t.Fatalf("expected 1 request, got %d", requests.Load())
	}
}

func TestGetAccessTokenTrailingSlashBaseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/oauth2/token" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600}`))
	}))
	defer server.Close()

	client := paykit.NewHTTPClient(paykit.HTTPClientConfig{
		IdempotencyPolicy: paykit.IdempotencyPolicyOptional,
	})
	client.SetTransport(server.Client().Transport)
	tm := NewTokenManager(client, "id", "secret", server.URL+"/")

	if _, err := tm.GetAccessToken(context.Background()); err != nil {
		t.Fatalf("GetAccessToken() error = %v", err)
	}
}

func TestGetAccessTokenCaching(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"access_token":"cached","expires_in":3600}`))
	}))
	defer server.Close()

	tm := newAuthTestManager(server)
	for i := 0; i < 3; i++ {
		token, err := tm.GetAccessToken(context.Background())
		if err != nil || token != "cached" {
			t.Fatalf("token=%q, error=%v", token, err)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("expected 1 request, got %d", requests.Load())
	}
}

func TestGetAccessTokenRefresh(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requests.Add(1)
		if n == 1 {
			_, _ = w.Write([]byte(`{"access_token":"old","expires_in":3600}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"new","expires_in":3600}`))
	}))
	defer server.Close()

	tm := newAuthTestManager(server)
	if _, err := tm.GetAccessToken(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Simulate a token approaching expiration.
	tm.mu.Lock()
	tm.expiry = time.Now().Add(30 * time.Second)
	tm.mu.Unlock()

	token, err := tm.GetAccessToken(context.Background())
	if err != nil || token != "new" {
		t.Fatalf("token=%q, error=%v", token, err)
	}
	if requests.Load() != 2 {
		t.Fatalf("expected 2 requests, got %d", requests.Load())
	}
}

func TestGetAccessTokenConcurrent(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"access_token":"shared","expires_in":3600}`))
	}))
	defer server.Close()

	tm := newAuthTestManager(server)
	var wg sync.WaitGroup
	errs := make(chan error, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := tm.GetAccessToken(context.Background())
			if err != nil {
				errs <- err
			} else if token != "shared" {
				errs <- &unexpectedTokenError{}
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent retrieval: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("expected 1 request, got %d", requests.Load())
	}
}

type unexpectedTokenError struct{}

func (*unexpectedTokenError) Error() string {
	return "unexpected token"
}

func TestGetAccessTokenErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		notContain string
	}{
		{
			"unauthorized",
			http.StatusUnauthorized,
			`{"error":"invalid_client","detail":"sensitive upstream response"}`,
			"sensitive upstream response",
		},
		{"invalid JSON", http.StatusOK, `{invalid`, ""},
		{"empty token", http.StatusOK, `{"access_token":"","expires_in":3600}`, ""},
		{"invalid expiry", http.StatusOK, `{"access_token":"token","expires_in":0}`, ""},
		{"negative expiry", http.StatusOK, `{"access_token":"token","expires_in":-1}`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			_, err := newAuthTestManager(server).GetAccessToken(context.Background())
			if err == nil {
				t.Fatal("expected an error")
			}
			if tt.notContain != "" && strings.Contains(err.Error(), tt.notContain) {
				t.Fatalf("error exposes response body: %v", err)
			}
		})
	}
}

func TestGetAccessTokenMissingCredentials(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		want         string
	}{
		{"client ID", "", "secret", "client ID is required"},
		{"client secret", "id", "", "client secret is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTokenManager(nil, tt.clientID, tt.clientSecret, "http://localhost")
			_, err := tm.GetAccessToken(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("GetAccessToken() error = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestGetAccessTokenInvalidBaseURL(t *testing.T) {
	for _, baseURL := range []string{"", " \t\r\n "} {
		t.Run(fmt.Sprintf("%q", baseURL), func(t *testing.T) {
			tm := NewTokenManager(nil, "id", "secret", baseURL)
			_, err := tm.GetAccessToken(context.Background())
			if err == nil || !strings.Contains(err.Error(), "base URL is required") {
				t.Fatalf("GetAccessToken() error = %v, want base URL error", err)
			}
		})
	}
}

func TestGetAccessTokenNilContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600}`))
	}))
	defer server.Close()

	token, err := newAuthTestManager(server).GetAccessToken(nil) //nolint:staticcheck // intentionally testing nil context handling
	if err != nil || token != "token" {
		t.Fatalf("token=%q, error=%v", token, err)
	}
}

func TestNewTokenManagerNilClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"token","expires_in":3600}`))
	}))
	defer server.Close()

	tm := NewTokenManager(nil, "id", "secret", server.URL)
	token, err := tm.GetAccessToken(context.Background())
	if err != nil || token != "token" {
		t.Fatalf("token=%q, error=%v", token, err)
	}
}

func TestGetAccessTokenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tm := NewTokenManager(nil, "id", "secret", "http://localhost")
	if _, err := tm.GetAccessToken(ctx); err == nil {
		t.Fatal("expected context cancellation error")
	}
}
