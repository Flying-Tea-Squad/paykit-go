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
			name:     "Provider-specific lookup for user cancellation",
			provider: "mpesa",
			code:     "1032",
			expected: ErrUserCancelled,
		},
		{
			name:     "Provider-specific lookup for PIN timeout",
			provider: "mpesa",
			code:     "1037",
			expected: ErrPinTimeout,
		},
		{
			name:     "Provider-specific lookup for invalid phone number",
			provider: "mpesa",
			code:     "2001",
			expected: ErrInvalidPhoneNumber,
		},
		{
			name:     "Provider-specific lookup for daily limit exceeded",
			provider: "mpesa",
			code:     "SFC_IC_0000006",
			expected: ErrDailyLimitExceeded,
		},
		{
			name:     "Provider-specific lookup for invalid shortcode",
			provider: "mpesa",
			code:     "SFC_IC_0000003",
			expected: ErrInvalidShortCode,
		},
		{
			name:     "Casing and whitespace resilience in provider name",
			provider: " Mpesa ",
			code:     "1032",
			expected: ErrUserCancelled,
		},
		{
			name:     "Uppercase provider name",
			provider: "MPESA",
			code:     "1025",
			expected: ErrTimeout,
		},
		{
			name:     "Fallback to default generic code",
			provider: "bogus",
			code:     "INVALID_PHONE",
			expected: ErrInvalidPhoneNumber,
		},
		{
			name:     "Fallback to default generic code for unknown provider",
			provider: "airtel",
			code:     "INSUFFICIENT_FUNDS",
			expected: ErrInsufficientFunds,
		},
		{
			name:     "Empty provider falls back to default",
			provider: "",
			code:     "TIMEOUT",
			expected: ErrTimeout,
		},
		{
			name:     "Unknown error code for known provider returns ErrProcessingError",
			provider: "mpesa",
			code:     "UNKNOWN_999",
			expected: ErrProcessingError,
		},
		{
			name:     "Unknown error code for unknown provider returns ErrProcessingError",
			provider: "unknown_gateway",
			code:     "RANDOM_ERR",
			expected: ErrProcessingError,
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
	sentinels := []struct {
		sentinel error
		expected string
	}{
		{ErrUserCancelledSentinel, ErrUserCancelled},
		{ErrPinTimeoutSentinel, ErrPinTimeout},
		{ErrInvalidPhoneNumberSentinel, ErrInvalidPhoneNumber},
		{ErrDailyLimitExceededSentinel, ErrDailyLimitExceeded},
		{ErrInvalidShortCodeSentinel, ErrInvalidShortCode},
		{ErrProcessingErrorSentinel, ErrProcessingError},
		{ErrDuplicateSentinel, ErrDuplicate},
		{ErrAuthFailedSentinel, ErrAuthFailed},
		{ErrInsufficientFundsSentinel, ErrInsufficientFunds},
		{ErrTimeoutSentinel, ErrTimeout},
		{ErrMissingIdempotencyKeySentinel, ErrMissingIdempotencyKey},
		{ErrIncorrectNumberSentinel, ErrIncorrectNumber},
		{ErrInvalidNumberSentinel, ErrInvalidNumber},
		{ErrInvalidExpiryDateSentinel, ErrInvalidExpiryDate},
		{ErrInvalidCVCSentinel, ErrInvalidCVC},
		{ErrExpiredCardSentinel, ErrExpiredCard},
		{ErrCardDeclinedSentinel, ErrCardDeclined},
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
