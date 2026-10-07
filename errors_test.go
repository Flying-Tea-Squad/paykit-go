package paykit

import (
	"testing"
)

func TestMapGatewayErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		code     string
		expected string
	}{
		{
			name:     "Provider-specific lookup",
			provider: "mpesa",
			code:     "1032",
			expected: ErrCardDeclined,
		},
		{
			name:     "Casing resilience in provider name",
			provider: " Mpesa ",
			code:     "1032",
			expected: ErrCardDeclined,
		},
		{
			name:     "Fallback to default generic code",
			provider: "bogus",
			code:     "INVALID_PHONE",
			expected: ErrInvalidNumber,
		},
		{
			name:     "Unknown error code returns ErrProcessingError",
			provider: "mpesa",
			code:     "UNKNOWN_999",
			expected: ErrProcessingError,
		},
		{
			name:     "Empty provider falls back to default",
			provider: "",
			code:     "TIMEOUT",
			expected: ErrTimeout,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := MapGatewayErrorCode(tc.provider, tc.code)
			if actual != tc.expected {
				t.Errorf("MapGatewayErrorCode(%q, %q) = %q, expected %q", tc.provider, tc.code, actual, tc.expected)
			}
		})
	}
}

func TestSentinelErrorsIntegrity(t *testing.T) {
	// Verify that the sentinel error variables match the constant strings.
	sentinels := []struct {
		sentinel error
		expected string
	}{
		{ErrIncorrectNumberSentinel, ErrIncorrectNumber},
		{ErrInvalidNumberSentinel, ErrInvalidNumber},
		{ErrInvalidCVCSentinel, ErrInvalidCVC},
		{ErrExpiredCardSentinel, ErrExpiredCard},
		{ErrCardDeclinedSentinel, ErrCardDeclined},
		{ErrProcessingErrorSentinel, ErrProcessingError},
		{ErrDuplicateSentinel, ErrDuplicate},
		{ErrAuthFailedSentinel, ErrAuthFailed},
		{ErrInsufficientFundsSentinel, ErrInsufficientFunds},
		{ErrTimeoutSentinel, ErrTimeout},
	}

	for _, s := range sentinels {
		if s.sentinel == nil {
			t.Errorf("Sentinel error for %q is nil", s.expected)
			continue
		}
		if s.sentinel.Error() != s.expected {
			t.Errorf("Sentinel error string = %q, expected %q", s.sentinel.Error(), s.expected)
		}
	}
}
