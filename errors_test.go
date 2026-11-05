package paykit_test

import (
	"testing"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
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
			expected: paykit.ErrUserCancelled,
		},
		{
			name:     "Provider-specific lookup for PIN timeout",
			provider: "mpesa",
			code:     "1037",
			expected: paykit.ErrPinTimeout,
		},
		{
			name:     "Provider-specific lookup for invalid phone number",
			provider: "mpesa",
			code:     "2001",
			expected: paykit.ErrInvalidPhoneNumber,
		},
		{
			name:     "Provider-specific lookup for daily limit exceeded",
			provider: "mpesa",
			code:     "SFC_IC_0000006",
			expected: paykit.ErrDailyLimitExceeded,
		},
		{
			name:     "Provider-specific lookup for invalid shortcode",
			provider: "mpesa",
			code:     "SFC_IC_0000003",
			expected: paykit.ErrInvalidShortCode,
		},
		{
			name:     "Casing and whitespace resilience in provider name",
			provider: " Mpesa ",
			code:     "1032",
			expected: paykit.ErrUserCancelled,
		},
		{
			name:     "Uppercase provider name",
			provider: "MPESA",
			code:     "1025",
			expected: paykit.ErrTimeout,
		},
		{
			name:     "Fallback to default generic code",
			provider: "bogus",
			code:     "INVALID_PHONE",
			expected: paykit.ErrInvalidPhoneNumber,
		},
		{
			name:     "Fallback to default generic code for unknown provider",
			provider: "airtel",
			code:     "INSUFFICIENT_FUNDS",
			expected: paykit.ErrInsufficientFunds,
		},
		{
			name:     "Empty provider falls back to default",
			provider: "",
			code:     "TIMEOUT",
			expected: paykit.ErrTimeout,
		},
		{
			name:     "Unknown error code for known provider returns ErrProcessingError",
			provider: "mpesa",
			code:     "UNKNOWN_999",
			expected: paykit.ErrProcessingError,
		},
		{
			name:     "Unknown error code for unknown provider returns ErrProcessingError",
			provider: "unknown_gateway",
			code:     "RANDOM_ERR",
			expected: paykit.ErrProcessingError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := paykit.MapGatewayErrorCode(tc.provider, tc.code)
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
		{paykit.ErrUserCancelledSentinel, paykit.ErrUserCancelled},
		{paykit.ErrPinTimeoutSentinel, paykit.ErrPinTimeout},
		{paykit.ErrInvalidPhoneNumberSentinel, paykit.ErrInvalidPhoneNumber},
		{paykit.ErrDailyLimitExceededSentinel, paykit.ErrDailyLimitExceeded},
		{paykit.ErrInvalidShortCodeSentinel, paykit.ErrInvalidShortCode},
		{paykit.ErrProcessingErrorSentinel, paykit.ErrProcessingError},
		{paykit.ErrDuplicateSentinel, paykit.ErrDuplicate},
		{paykit.ErrAuthFailedSentinel, paykit.ErrAuthFailed},
		{paykit.ErrInsufficientFundsSentinel, paykit.ErrInsufficientFunds},
		{paykit.ErrTimeoutSentinel, paykit.ErrTimeout},
		{paykit.ErrMissingIdempotencyKeySentinel, paykit.ErrMissingIdempotencyKey},
		{paykit.ErrIncorrectNumberSentinel, paykit.ErrIncorrectNumber},
		{paykit.ErrInvalidNumberSentinel, paykit.ErrInvalidNumber},
		{paykit.ErrInvalidExpiryDateSentinel, paykit.ErrInvalidExpiryDate},
		{paykit.ErrInvalidCVCSentinel, paykit.ErrInvalidCVC},
		{paykit.ErrExpiredCardSentinel, paykit.ErrExpiredCard},
		{paykit.ErrCardDeclinedSentinel, paykit.ErrCardDeclined},
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
