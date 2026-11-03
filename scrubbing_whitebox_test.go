package paykit

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegexScrubber_NilAndEmptyDefenses(t *testing.T) {
	t.Parallel()

	var nilScrubber *RegexScrubber
	assert.Equal(t, "test", nilScrubber.Scrub("test"))
	assert.Equal(t, "", nilScrubber.Scrub(""))

	emptyScrubber := &RegexScrubber{}
	assert.Equal(t, "test", emptyScrubber.Scrub("test"))

	defaultScrubber := NewDefaultScrubber()
	assert.Equal(t, "", defaultScrubber.Scrub(""))
}

func TestScrubbingHandler_NilDefenses(t *testing.T) {
	t.Parallel()

	var nilHandler *ScrubbingHandler
	assert.False(t, nilHandler.Enabled(context.Background(), slog.LevelInfo))
	assert.NoError(t, nilHandler.Handle(context.Background(), slog.Record{}))
	assert.Nil(t, nilHandler.WithAttrs([]slog.Attr{slog.String("k", "v")}))
	assert.Nil(t, nilHandler.WithGroup("grp"))

	assert.Nil(t, NewScrubbingHandler(nil, nil))
}

func TestScrubbingHandler_EmptyFastPaths(t *testing.T) {
	t.Parallel()

	baseHandler := slog.NewTextHandler(io.Discard, nil)
	handler := NewScrubbingHandler(baseHandler, nil)

	// WithAttrs with 0 attributes returns receiver directly without allocating
	assert.Same(t, handler, handler.WithAttrs(nil))
	assert.Same(t, handler, handler.WithAttrs([]slog.Attr{}))

	// WithGroup with empty name returns receiver directly without allocating
	assert.Same(t, handler, handler.WithGroup(""))
}

func TestIsSensitiveKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key       string
		sensitive bool
	}{
		{"password", true},
		{"Password", true},
		{"user_password", true},
		{"passkey", true},
		{"pass_key", true},
		{"consumer_secret", true},
		{"consumerSecret", true},
		{"client_secret", true},
		{"clientSecret", true},
		{"security_credential", true},
		{"SecurityCredential", true},
		{"access_token", true},
		{"accessToken", true},
		{"refresh_token", true},
		{"token", true},
		{"pin", true},
		{"api_key", true},
		{"apiKey", true},
		{"secret", true},
		{"authorization", true},
		// Non-sensitive keys
		{"amount", false},
		{"currency", false},
		{"account_reference", false},
		{"transaction_id", false},
		{"status", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.sensitive, isSensitiveKey(tt.key))
		})
	}
}

func TestIsPhoneKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key     string
		isPhone bool
	}{
		{"phone", true},
		{"phoneNumber", true},
		{"PhoneNumber", true},
		{"customer_phone", true},
		{"msisdn", true},
		{"Msisdn", true},
		{"party_a", true},
		{"party_b", true},
		{"partyA", true},
		{"partyB", true},
		{"sender_phone", true},
		{"receiver_phone", true},
		// Non-phone keys
		{"amount", false},
		{"shortcode", false},
		{"account", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.isPhone, isPhoneKey(tt.key))
		})
	}
}
