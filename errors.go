package paykit

import "errors"

// Standardized error codes returned by PayKit.
const (
	ErrIncorrectNumber   = "incorrect_number"
	ErrInvalidNumber     = "invalid_number"
	ErrInvalidExpiryDate = "invalid_expiry_date"
	ErrInvalidCVC        = "invalid_cvc"
	ErrExpiredCard       = "expired_card"
	ErrCardDeclined      = "card_declined"
	ErrProcessingError   = "processing_error"
	ErrDuplicate         = "duplicate_transaction"
	ErrAuthFailed        = "authentication_failed"
	ErrInsufficientFunds = "insufficient_funds"
	ErrTimeout           = "request_timeout"
)

// Sentinel errors for programmatic error checks.
var (
	ErrIncorrectNumberSentinel   = errors.New(ErrIncorrectNumber)
	ErrInvalidNumberSentinel     = errors.New(ErrInvalidNumber)
	ErrInvalidExpiryDateSentinel = errors.New(ErrInvalidExpiryDate)
	ErrInvalidCVCSentinel        = errors.New(ErrInvalidCVC)
	ErrExpiredCardSentinel       = errors.New(ErrExpiredCard)
	ErrCardDeclinedSentinel      = errors.New(ErrCardDeclined)
	ErrProcessingErrorSentinel   = errors.New(ErrProcessingError)
	ErrDuplicateSentinel         = errors.New(ErrDuplicate)
	ErrAuthFailedSentinel        = errors.New(ErrAuthFailed)
	ErrInsufficientFundsSentinel = errors.New(ErrInsufficientFunds)
	ErrTimeoutSentinel           = errors.New(ErrTimeout)
)

// gatewayErrorMap maps provider-specific error codes to standardized PayKit codes
// by provider namespace.
var gatewayErrorMap = map[string]map[string]string{
	"mpesa": {
		"1032": ErrCardDeclined,
		"2001": ErrInvalidNumber,
		"1025": ErrTimeout,
	},
	"default": {
		// Generic provider examples
		"INVALID_PHONE":      ErrInvalidNumber,
		"INSUFFICIENT_FUNDS": ErrInsufficientFunds,
		"AUTH_FAILED":        ErrAuthFailed,
		"DUPLICATE":          ErrDuplicate,
		"TIMEOUT":            ErrTimeout,
	},
}

// MapGatewayErrorCode translates a provider-specific error code into a
// standardized PayKit error code. If the code is unknown, it returns
// ErrProcessingError.
func MapGatewayErrorCode(provider, code string) string {
	// First check the provider-specific namespace
	if providerMap, ok := gatewayErrorMap[provider]; ok {
		if standardized, ok := providerMap[code]; ok {
			return standardized
		}
	}

	// Fallback to the default generic namespace
	if defaultMap, ok := gatewayErrorMap["default"]; ok {
		if standardized, ok := defaultMap[code]; ok {
			return standardized
		}
	}

	return ErrProcessingError
}
