package airtel

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
)

const tokenExpiryBuffer = 60 * time.Second

// AuthResponse represents an Airtel Money OAuth2 token response.
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// TokenManager fetches, caches, and refreshes Airtel Money access tokens.
type TokenManager struct {
	mu           sync.RWMutex
	client       *paykit.HTTPClient
	clientID     string
	clientSecret string
	baseURL      string
	accessToken  string
	expiry       time.Time
}

// NewTokenManager creates an Airtel Money OAuth2 token manager.
func NewTokenManager(client *paykit.HTTPClient, clientID, clientSecret, baseURL string) *TokenManager {
	if client == nil {
		client = paykit.NewHTTPClient(paykit.HTTPClientConfig{
			IdempotencyPolicy: paykit.IdempotencyPolicyOptional,
		})
	}

	return &TokenManager{
		client:       client,
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      baseURL,
	}
}

// GetAccessToken returns a cached token or fetches a fresh one.
func (tm *TokenManager) GetAccessToken(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	tm.mu.RLock()
	if tm.accessToken != "" && time.Now().Add(tokenExpiryBuffer).Before(tm.expiry) {
		token := tm.accessToken
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Another goroutine may have refreshed the token.
	if tm.accessToken != "" && time.Now().Add(tokenExpiryBuffer).Before(tm.expiry) {
		return tm.accessToken, nil
	}

	token, expiry, err := tm.fetchAccessToken(ctx)
	if err != nil {
		return "", err
	}

	tm.accessToken = token
	tm.expiry = expiry

	return token, nil
}

// fetchAccessToken requests a token using OAuth2 client credentials.
func (tm *TokenManager) fetchAccessToken(ctx context.Context) (string, time.Time, error) {
	if tm.clientID == "" || tm.clientSecret == "" {
		return "", time.Time{}, errors.New("airtel: client ID and client secret are required")
	}

	payload, err := json.Marshal(struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		GrantType    string `json:"grant_type"`
	}{
		ClientID:     tm.clientID,
		ClientSecret: tm.clientSecret,
		GrantType:    "client_credentials",
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("airtel: encode auth request: %w", err)
	}

	endpoint := strings.TrimRight(tm.baseURL, "/") + "/auth/oauth2/token"

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("airtel: create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Generate a unique UUIDv4 idempotency key for each token request.
	// This supports HTTP clients configured to require idempotency keys.
	var keyBytes [16]byte
	if _, err := rand.Read(keyBytes[:]); err != nil {
		return "", time.Time{}, fmt.Errorf(
			"airtel: generate idempotency key: %w", err,
		)
	}

	keyBytes[6] = (keyBytes[6] & 0x0f) | 0x40
	keyBytes[8] = (keyBytes[8] & 0x3f) | 0x80

	idempotencyKey := fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		keyBytes[0:4],
		keyBytes[4:6],
		keyBytes[6:8],
		keyBytes[8:10],
		keyBytes[10:16],
	)

	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := tm.client.Do(ctx, req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf(
			"airtel: oauth request failed: %w", err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Avoid exposing response bodies containing sensitive data.
		return "", time.Time{}, fmt.Errorf(
			"airtel: oauth failed with HTTP status %d",
			resp.StatusCode,
		)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(
		io.LimitReader(resp.Body, 1<<20),
	).Decode(&authResp); err != nil {
		return "", time.Time{}, fmt.Errorf(
			"airtel: decode auth response: %w", err,
		)
	}

	if authResp.AccessToken == "" {
		return "", time.Time{}, errors.New(
			"airtel: empty access token",
		)
	}

	if authResp.ExpiresIn <= 0 {
		return "", time.Time{}, errors.New(
			"airtel: invalid token expiry",
		)
	}

	expiry := time.Now().Add(
		time.Duration(authResp.ExpiresIn) * time.Second,
	)

	return authResp.AccessToken, expiry, nil
}
