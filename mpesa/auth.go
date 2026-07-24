package mpesa

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// HTTPClient defines the contract for executing HTTP requests.
// It is implemented by *http.Client and foundation HTTP client wrappers.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// AuthResponse represents Daraja's OAuth token API response.
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

// UnmarshalJSON handles expires_in as string or numeric values.
func (a *AuthResponse) UnmarshalJSON(data []byte) error {
	type Alias AuthResponse
	aux := &struct {
		ExpiresInRaw interface{} `json:"expires_in"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch v := aux.ExpiresInRaw.(type) {
	case string:
		a.ExpiresIn = v
	case float64:
		a.ExpiresIn = strconv.Itoa(int(v))
	case int:
		a.ExpiresIn = strconv.Itoa(v)
	}

	return nil
}

// TokenManager handles fetching, caching, and automatic refreshing of M-Pesa OAuth2 tokens.
type TokenManager struct {
	mu             sync.RWMutex
	client         HTTPClient
	consumerKey    string
	consumerSecret string
	baseURL        string
	accessToken    string
	expiry         time.Time
	expiryBuffer   time.Duration
}

// NewTokenManager creates a new TokenManager instance.
func NewTokenManager(client HTTPClient, consumerKey, consumerSecret, baseURL string) *TokenManager {
	if client == nil {
		client = http.DefaultClient
	}
	return &TokenManager{
		client:         client,
		consumerKey:    consumerKey,
		consumerSecret: consumerSecret,
		baseURL:        baseURL,
		expiryBuffer:   60 * time.Second,
	}
}

// GetAccessToken returns a valid M-Pesa access token, refreshing it if expired or near expiry.
func (tm *TokenManager) GetAccessToken(ctx context.Context) (string, error) {
	return tm.getAccessToken(ctx)
}

// getAccessToken returns a valid M-Pesa access token using consumer key/secret.
func (tm *TokenManager) getAccessToken(ctx context.Context) (string, error) {
	now := time.Now()
	buffer := tm.expiryBuffer

	// Check cache under read lock
	tm.mu.RLock()
	if tm.accessToken != "" && now.Add(buffer).Before(tm.expiry) {
		token := tm.accessToken
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	// Acquire write lock to update token
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Double-check after acquiring write lock
	now = time.Now()
	if tm.accessToken != "" && now.Add(buffer).Before(tm.expiry) {
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

// fetchAccessToken executes the HTTP request to Daraja OAuth endpoint to retrieve a fresh access token.
func (tm *TokenManager) fetchAccessToken(ctx context.Context) (string, time.Time, error) {
	if tm.consumerKey == "" || tm.consumerSecret == "" {
		return "", time.Time{}, errors.New("mpesa: consumer key and consumer secret are required")
	}

	endpoint := strings.TrimRight(tm.baseURL, "/") + "/oauth/v1/generate?grant_type=client_credentials"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("mpesa: failed to create auth request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(tm.consumerKey + ":" + tm.consumerSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	client := tm.client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("mpesa: oauth request failed: %w", err)
	}
	defer func() {
	if err := resp.Body.Close(); err != nil {
		return "", time.Time{}, fmt.Errorf("mpesa: failed to read oauth response: %w", err)
	}
}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("mpesa: failed to read oauth response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("mpesa: oauth failed with status %d: %s", resp.StatusCode, string(body))
	}

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return "", time.Time{}, fmt.Errorf("mpesa: failed to decode oauth response: %w", err)
	}

	if authResp.AccessToken == "" {
		return "", time.Time{}, errors.New("mpesa: empty access_token received from oauth endpoint")
	}

	expiresInSec := 3599
	if authResp.ExpiresIn != "" {
		if sec, err := strconv.Atoi(authResp.ExpiresIn); err == nil && sec > 0 {
			expiresInSec = sec
		}
	}

	expiry := time.Now().Add(time.Duration(expiresInSec) * time.Second)
	return authResp.AccessToken, expiry, nil
}
