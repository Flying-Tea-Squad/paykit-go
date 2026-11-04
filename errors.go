package paykit

import (
	"errors"
	"strings"
)

// Standardized error codes returned by PayKit.
const (
	// Mobile money and payment rail error codes.
	ErrUserCancelled      = "user_cancelled"
	ErrPinTimeout         = "pin_timeout"
	ErrInvalidPhoneNumber = "invalid_phone_number"
	ErrDailyLimitExceeded = "daily_limit_exceeded"
	ErrInvalidShortCode   = "invalid_shortcode"

	// General payment processing error codes.
	ErrProcessingError       = "processing_error"
	ErrDuplicate             = "duplicate_transaction"
	ErrAuthFailed            = "authentication_failed"
	ErrInsufficientFunds     = "insufficient_funds"
	ErrTimeout               = "request_timeout"
	ErrMissingIdempotencyKey = "missing_idempotency_key"

	// Card gateway error codes.
	ErrIncorrectNumber   = "incorrect_number"
	ErrInvalidNumber     = "invalid_number"
	ErrInvalidExpiryDate = "invalid_expiry_date"
	ErrInvalidCVC        = "invalid_cvc"
	ErrExpiredCard       = "expired_card"
	ErrCardDeclined      = "card_declined"
)

// Sentinel errors for programmatic error checks.
var (
	// Mobile money sentinels.
	ErrUserCancelledSentinel      = errors.New(ErrUserCancelled)
	ErrPinTimeoutSentinel         = errors.New(ErrPinTimeout)
	ErrInvalidPhoneNumberSentinel = errors.New(ErrInvalidPhoneNumber)
	ErrDailyLimitExceededSentinel = errors.New(ErrDailyLimitExceeded)
	ErrInvalidShortCodeSentinel   = errors.New(ErrInvalidShortCode)

	// General payment processing sentinels.
	ErrProcessingErrorSentinel       = errors.New(ErrProcessingError)
	ErrDuplicateSentinel             = errors.New(ErrDuplicate)
	ErrAuthFailedSentinel            = errors.New(ErrAuthFailed)
	ErrInsufficientFundsSentinel     = errors.New(ErrInsufficientFunds)
	ErrTimeoutSentinel               = errors.New(ErrTimeout)
	ErrMissingIdempotencyKeySentinel = errors.New(ErrMissingIdempotencyKey)

	// Card gateway sentinels.
	ErrIncorrectNumberSentinel   = errors.New(ErrIncorrectNumber)
	ErrInvalidNumberSentinel     = errors.New(ErrInvalidNumber)
	ErrInvalidExpiryDateSentinel = errors.New(ErrInvalidExpiryDate)
	ErrInvalidCVCSentinel        = errors.New(ErrInvalidCVC)
	ErrExpiredCardSentinel       = errors.New(ErrExpiredCard)
	ErrCardDeclinedSentinel      = errors.New(ErrCardDeclined)
)

// gatewayErrorMap maps provider-specific error codes to standardized PayKit codes
// by provider namespace.
var gatewayErrorMap = map[string]map[string]string{
	"mpesa": {
		"1032":           ErrUserCancelled,
		"1037":           ErrPinTimeout,
		"2001":           ErrInvalidPhoneNumber,
		"1025":           ErrTimeout,
		"SFC_IC_0000006": ErrDailyLimitExceeded,
		"SFC_IC_0000003": ErrInvalidShortCode,
	},
	"default": {
		"INVALID_PHONE":      ErrInvalidPhoneNumber,
		"INSUFFICIENT_FUNDS": ErrInsufficientFunds,
		"AUTH_FAILED":        ErrAuthFailed,
		"DUPLICATE":          ErrDuplicate,
		"TIMEOUT":            ErrTimeout,
		"USER_CANCELLED":     ErrUserCancelled,
	},
}

// MapGatewayErrorCode translates a provider-specific error code into a
// standardized PayKit error code.
//
// The provider identifier is case-insensitive; it is automatically trimmed of
// whitespace and lowercased before lookup (e.g., "Mpesa" resolves to "mpesa").
//
// If the error code is not found within the provider's namespace, the generic
// "default" namespace is searched as a fallback. If the code remains unknown,
// ErrProcessingError is returned.
func MapGatewayErrorCode(provider, code string) string {
	normalizedProvider := strings.ToLower(strings.TrimSpace(provider))

	if providerMap, ok := gatewayErrorMap[normalizedProvider]; ok {
		if standardized, ok := providerMap[code]; ok {
			return standardized
		}
	}

	if defaultMap, ok := gatewayErrorMap["default"]; ok {
		if standardized, ok := defaultMap[code]; ok {
			return standardized
		}
	}

	return ErrProcessingError
}
