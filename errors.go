package paykit

import "errors"

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

// gatewayErrorMap maps provider-specific error codes to standardized PayKit codes.
var gatewayErrorMap = map[string]string{
	// M-Pesa examples
	"1032": ErrCardDeclined,
	"2001": ErrInvalidNumber,
	"1025": ErrTimeout,

	// Generic provider examples
	"INVALID_PHONE":      ErrInvalidNumber,
	"INSUFFICIENT_FUNDS": ErrInsufficientFunds,
	"AUTH_FAILED":        ErrAuthFailed,
	"DUPLICATE":          ErrDuplicate,
	"TIMEOUT":            ErrTimeout,
}

// MapGatewayErrorCode translates a provider-specific error code into a
// standardized PayKit error code. If the code is unknown, it returns
// ErrProcessingError.
func MapGatewayErrorCode(code string) string {
	if standardized, ok := gatewayErrorMap[code]; ok {
		return standardized
	}

	return ErrProcessingError
}
