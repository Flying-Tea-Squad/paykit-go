package paykit

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// ErrInvalidPhoneNumber is returned when a phone number cannot be parsed or normalized.
	ErrInvalidPhoneNumber = errors.New("invalid phone number")

	// kenyanMobilePrefixes are the valid mobile network prefixes in Kenya.
	// 7xx = Safaricom, 1xx = Airtel/Telkom, 0xx = legacy/other
	kenyanMobilePrefixes = map[string]bool{
		"70": true, "71": true, "72": true, "73": true, "74": true,
		"75": true, "76": true, "77": true, "78": true, "79": true,
		"10": true, "11": true, "12": true, "13": true, "14": true,
		"15": true, "16": true, "17": true, "18": true, "19": true,
	}

	// phoneRegex matches various Kenyan phone number formats.
	// Captures the 9-digit national significant number (after country code).
	phoneRegex = regexp.MustCompile(`^(?:\+?254|0)?([17]\d{8})$`)
)

// FormatKenyanPhone normalizes a Kenyan phone number to E.164 format (2547XXXXXXXX or 2541XXXXXXXX).
//
// Accepted input formats:
//   - 07XXXXXXXXX (local with leading zero)
//   - 01XXXXXXXXX (local with leading zero)
//   - 7XXXXXXXXX  (local without leading zero)
//   - 1XXXXXXXXX  (local without leading zero)
//   - 2547XXXXXXXX (E.164 without +)
//   - 2541XXXXXXXX (E.164 without +)
//   - +2547XXXXXXXX (full E.164)
//   - +2541XXXXXXXX (full E.164)
//
// Returns the normalized 12-digit E.164 string (e.g., "254700000001").
// Returns ErrInvalidPhoneNumber for invalid prefixes, wrong length, or non-Kenyan numbers.
func FormatKenyanPhone(phone string) (string, error) {
	cleaned := strings.TrimSpace(phone)
	if cleaned == "" {
		return "", ErrInvalidPhoneNumber
	}

	matches := phoneRegex.FindStringSubmatch(cleaned)
	if matches == nil {
		return "", ErrInvalidPhoneNumber
	}

	nationalNumber := matches[1] // 9 digits starting with 7 or 1
	prefix := nationalNumber[:2]

	if !kenyanMobilePrefixes[prefix] {
		return "", ErrInvalidPhoneNumber
	}

	return "254" + nationalNumber, nil
}

// IsValidKenyanPhone reports whether the phone number can be normalized to a valid Kenyan E.164 number.
func IsValidKenyanPhone(phone string) bool {
	_, err := FormatKenyanPhone(phone)
	return err == nil
}
