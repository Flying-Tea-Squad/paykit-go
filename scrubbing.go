package paykit

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// MaskReplacement is the replacement text used for redacted credentials, tokens, and secrets.
const MaskReplacement = "[REDACTED]"

// PhoneMaskReplacement is the replacement string used for masked phone number digits.
const PhoneMaskReplacement = "***"

// Scrubber defines an interface for redacting sensitive Personally Identifiable Information (PII)
// and credentials from strings, logs, and HTTP wire dumps.
//
// Implementations must be safe for concurrent use by multiple goroutines.
type Scrubber interface {
	// Scrub returns a sanitized copy of input with sensitive data (such as passwords,
	// authentication tokens, consumer secrets, and customer MSISDNs) masked or redacted.
	Scrub(input string) string
}

// NopScrubber is a no-op implementation of Scrubber that leaves input strings unchanged.
// It is useful in testing scenarios or when callers explicitly opt out of log sanitization.
type NopScrubber struct{}

// Scrub returns input unmodified.
func (NopScrubber) Scrub(input string) string {
	return input
}

// scrubRule binds a precompiled regular expression to a string replacement template or dynamic replacer.
type scrubRule struct {
	regex       *regexp.Regexp
	replacement string
	replaceFunc func(string) string
}

// RegexScrubber implements Scrubber using precompiled regular expressions
// to redact sensitive credentials and mask customer phone numbers while preserving diagnostic structure.
type RegexScrubber struct {
	rules []scrubRule
}

var (
	// authHeaderRegex matches Authorization and Proxy-Authorization headers with optional auth schemes (Bearer, Basic, etc.).
	authHeaderRegex = regexp.MustCompile(`(?i)((?:Proxy-)?Authorization:\s*(?:Bearer\s+|Basic\s+)?)[^\r\n,;]+`)

	// apiKeyHeaderRegex matches standard API key headers.
	apiKeyHeaderRegex = regexp.MustCompile(`(?i)((?:X-)?Api-Key:\s*)[^\r\n,;]+`)

	// standaloneBearerRegex matches Bearer tokens appearing standalone in messages or payloads.
	standaloneBearerRegex = regexp.MustCompile(`(?i)\b(Bearer\s+)[A-Za-z0-9\-_.~+/]{8,}=*`)

	// standaloneBasicRegex matches Basic authentication tokens (base64) appearing standalone.
	standaloneBasicRegex = regexp.MustCompile(`\b(Basic\s+)[A-Za-z0-9+/]{6,}={0,2}`)

	// jsonCredentialStringRegex matches JSON fields carrying sensitive credentials with string values.
	jsonCredentialStringRegex = regexp.MustCompile(
		`(?i)("(?:\w*[._-])?(?:password|passkey|pass_key|consumer_secret|consumerSecret|client_secret|clientSecret|security_credential|securityCredential|access_token|accessToken|refresh_token|refreshToken|token|pin|api_key|apiKey|secret)"\s*:\s*)"(?:[^"\\]|\\.)*"`,
	)

	// jsonCredentialNumericRegex matches JSON fields carrying numeric sensitive credentials (such as PINs).
	jsonCredentialNumericRegex = regexp.MustCompile(
		`(?i)("(?:\w*[._-])?(?:pin|password|secret)"\s*:\s*)\d+`,
	)

	// queryFormCredentialRegex matches sensitive key-value pairs in query strings and form-encoded payloads.
	queryFormCredentialRegex = regexp.MustCompile(
		`(?i)(^|[?&;])((?:[a-zA-Z0-9_.-]*[._-])?(?:password|passkey|pass_key|consumer_secret|consumerSecret|client_secret|clientSecret|security_credential|securityCredential|access_token|accessToken|refresh_token|refreshToken|token|pin|api_key|apiKey|secret)=)[^&; \r\n\t]+`,
	)

	// xmlCredentialRegex matches sensitive tags in XML / SOAP payloads.
	xmlCredentialRegex = regexp.MustCompile(
		`(?i)(<(?:\w+:)?(?:Password|SecurityCredential|PassKey|ConsumerSecret|Token|PIN)(?:\s+[^>]*)?>)[^<]+(</(?:\w+:)?(?:Password|SecurityCredential|PassKey|ConsumerSecret|Token|PIN)>)`,
	)

	// jsonPhoneFieldRegex matches JSON fields explicitly designated for phone numbers or MSISDNs.
	jsonPhoneFieldRegex = regexp.MustCompile(
		`(?i)("(?:\w*[._-])?(?:phone|phoneNumber|msisdn|party_?a|party_?b|sender_?phone|receiver_?phone)"\s*:\s*)(?:"([^"\r\n]*)"|(\+?\d+))`,
	)

	// standalonePhoneRegex matches 10-12 digit Kenyan phone numbers or common formatted variants in text.
	standalonePhoneRegex = regexp.MustCompile(
		`(?i)(^|[^\d+])(\+?254[17]\d{8}|0[17]\d{8}|\+?254[ -]?[17]\d{2,3}[ -]?\d{2,3}[ -]?\d{4}|0[17]\d{2}[ -]?\d{3}[ -]?\d{3})\b`,
	)

	// sensitiveKeyRegex identifies attribute keys that hold sensitive credentials.
	sensitiveKeyRegex = regexp.MustCompile(
		`(?i)(?:^|[._-])(?:password|passkey|pass_key|consumer_secret|consumerSecret|client_secret|clientSecret|security_credential|securityCredential|access_token|accessToken|refresh_token|refreshToken|token|pin|api_key|apiKey|secret|authorization)$`,
	)

	// phoneKeyRegex identifies attribute keys that hold phone numbers or MSISDNs.
	phoneKeyRegex = regexp.MustCompile(
		`(?i)(?:^|[._-])(?:phone|phoneNumber|msisdn|party_?a|party_?b|sender_?phone|receiver_?phone)$`,
	)
)

// NewDefaultScrubber returns a Scrubber configured with default precompiled rules to mask
// authorization headers, passwords, consumer secrets, passkeys, security credentials,
// tokens, PINs, and customer phone numbers.
func NewDefaultScrubber() *RegexScrubber {
	return &RegexScrubber{
		rules: []scrubRule{
			// 1. Authorization and API Key headers
			{regex: authHeaderRegex, replacement: "${1}" + MaskReplacement},
			{regex: apiKeyHeaderRegex, replacement: "${1}" + MaskReplacement},

			// 2. Standalone Bearer and Basic tokens
			{regex: standaloneBearerRegex, replacement: "${1}" + MaskReplacement},
			{regex: standaloneBasicRegex, replacement: "${1}" + MaskReplacement},

			// 3. JSON credential fields
			{regex: jsonCredentialStringRegex, replacement: "${1}\"" + MaskReplacement + "\""},
			{regex: jsonCredentialNumericRegex, replacement: "${1}\"" + MaskReplacement + "\""},

			// 4. URL query strings and form-encoded credentials
			{regex: queryFormCredentialRegex, replacement: "${1}${2}" + MaskReplacement},

			// 5. XML / SOAP credential tags
			{regex: xmlCredentialRegex, replacement: "${1}" + MaskReplacement + "${2}"},

			// 6. Explicit JSON phone / MSISDN fields (ensuring valid JSON quoting for both quoted and raw numbers)
			{
				regex: jsonPhoneFieldRegex,
				replaceFunc: func(match string) string {
					submatches := jsonPhoneFieldRegex.FindStringSubmatch(match)
					if len(submatches) != 4 {
						return match
					}
					keyPart := submatches[1]
					quotedVal := submatches[2]
					numericVal := submatches[3]

					if numericVal != "" {
						masked := MaskPhone(numericVal)
						return fmt.Sprintf("%s\"%s\"", keyPart, masked)
					}

					if quotedVal != "" {
						masked := MaskPhone(quotedVal)
						return fmt.Sprintf("%s\"%s\"", keyPart, masked)
					}

					return match
				},
			},

			// 7. Standalone customer phone numbers anywhere in wire dumps or log text
			{
				regex: standalonePhoneRegex,
				replaceFunc: func(match string) string {
					submatches := standalonePhoneRegex.FindStringSubmatch(match)
					if len(submatches) != 3 {
						return match
					}
					leading := submatches[1]
					candidate := submatches[2]
					masked := MaskPhone(candidate)
					return leading + masked
				},
			},
		},
	}
}

// Scrub sanitizes input by sequentially evaluating all precompiled scrubbing rules.
func (s *RegexScrubber) Scrub(input string) string {
	if s == nil || len(s.rules) == 0 || input == "" {
		return input
	}

	result := input
	for _, rule := range s.rules {
		if rule.replaceFunc != nil {
			result = rule.regex.ReplaceAllStringFunc(result, rule.replaceFunc)
		} else {
			result = rule.regex.ReplaceAllString(result, rule.replacement)
		}
	}
	return result
}

// MaskPhone masks a phone number or MSISDN, preserving diagnostic prefix and suffix
// while replacing middle identifying digits with asterisks (e.g., "254712345678" -> "2547***5678").
//
// Shortcodes (such as Safaricom till or paybill numbers under 8 digits) and non-numeric
// strings are returned unmodified to preserve operational routing context.
func MaskPhone(phone string) string {
	cleaned := strings.TrimSpace(phone)
	if len(cleaned) == 0 {
		return phone
	}

	// Strip common delimiters for normalization
	digitsOnly := strings.NewReplacer(" ", "", "-", "", ".", "", "(", "", ")", "").Replace(cleaned)
	hasPlus := strings.HasPrefix(digitsOnly, "+")
	rawDigits := strings.TrimPrefix(digitsOnly, "+")

	// Shortcodes (typically 5 to 7 digits), invalid lengths, or non-numeric strings should not be masked
	if len(rawDigits) < 8 || !isAllDigits(rawDigits) {
		return phone
	}

	// 12-digit Kenyan E.164 without plus (e.g., "254712345678" -> "2547***5678")
	if len(rawDigits) == 12 && strings.HasPrefix(rawDigits, "254") {
		prefix := rawDigits[:4]
		suffix := rawDigits[8:]
		if hasPlus {
			return "+" + prefix + PhoneMaskReplacement + suffix
		}
		return prefix + PhoneMaskReplacement + suffix
	}

	// 10-digit local Kenyan format with leading zero (e.g., "0712345678" -> "07***5678")
	if len(rawDigits) == 10 && strings.HasPrefix(rawDigits, "0") {
		prefix := rawDigits[:2]
		suffix := rawDigits[6:]
		return prefix + PhoneMaskReplacement + suffix
	}

	// General international MSISDN formats (9 to 15 digits)
	if len(rawDigits) >= 9 {
		var prefix, suffix string
		if len(rawDigits) <= 10 {
			prefix = rawDigits[:2]
			suffix = rawDigits[len(rawDigits)-4:]
		} else {
			prefix = rawDigits[:4]
			suffix = rawDigits[len(rawDigits)-4:]
		}
		if hasPlus {
			return "+" + prefix + PhoneMaskReplacement + suffix
		}
		return prefix + PhoneMaskReplacement + suffix
	}

	return phone
}

// ScrubbingHandler is an slog.Handler middleware that masks sensitive credentials and PII
// in log records before delegating them to an underlying handler.
type ScrubbingHandler struct {
	next     slog.Handler
	scrubber Scrubber
}

var _ slog.Handler = (*ScrubbingHandler)(nil)

// NewScrubbingHandler wraps next with automated sensitive data scrubbing.
// If scrubber is nil, NewDefaultScrubber is used. If next is already a *ScrubbingHandler,
// it is returned directly to prevent redundant wrapping layers.
func NewScrubbingHandler(next slog.Handler, scrubber Scrubber) *ScrubbingHandler {
	if next == nil {
		return nil
	}
	if scrubber == nil {
		scrubber = NewDefaultScrubber()
	}
	if existing, ok := next.(*ScrubbingHandler); ok {
		return existing
	}
	return &ScrubbingHandler{
		next:     next,
		scrubber: scrubber,
	}
}

// Enabled reports whether the handler emits log records at the given level.
func (h *ScrubbingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h == nil || h.next == nil {
		return false
	}
	return h.next.Enabled(ctx, level)
}

// Handle sanitizes the record's message and attributes before passing it downstream.
func (h *ScrubbingHandler) Handle(ctx context.Context, r slog.Record) error {
	if h == nil || h.next == nil {
		return nil
	}
	if h.scrubber == nil {
		return h.next.Handle(ctx, r)
	}

	// Scrub record message
	scrubbedMsg := h.scrubber.Scrub(r.Message)
	scrubbedRecord := slog.NewRecord(r.Time, r.Level, scrubbedMsg, r.PC)

	// Scrub record attributes
	r.Attrs(func(a slog.Attr) bool {
		scrubbedRecord.AddAttrs(h.scrubAttr(a))
		return true
	})

	return h.next.Handle(ctx, scrubbedRecord)
}

// WithAttrs returns a new ScrubbingHandler whose attributes are pre-sanitized.
func (h *ScrubbingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h == nil || h.next == nil {
		return nil
	}
	if len(attrs) == 0 {
		return h
	}
	if h.scrubber == nil {
		return &ScrubbingHandler{
			next:     h.next.WithAttrs(attrs),
			scrubber: h.scrubber,
		}
	}

	scrubbed := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		scrubbed[i] = h.scrubAttr(a)
	}

	return &ScrubbingHandler{
		next:     h.next.WithAttrs(scrubbed),
		scrubber: h.scrubber,
	}
}

// WithGroup returns a new ScrubbingHandler with the specified group name.
func (h *ScrubbingHandler) WithGroup(name string) slog.Handler {
	if h == nil || h.next == nil {
		return nil
	}
	if name == "" {
		return h
	}
	return &ScrubbingHandler{
		next:     h.next.WithGroup(name),
		scrubber: h.scrubber,
	}
}

func (h *ScrubbingHandler) scrubAttr(a slog.Attr) slog.Attr {
	a.Value = a.Value.Resolve()
	key := a.Key

	// If attribute key indicates credentials, redact value unconditionally
	if isSensitiveKey(key) {
		return slog.String(key, MaskReplacement)
	}

	// If attribute key indicates a phone number, mask value
	if isPhoneKey(key) {
		switch a.Value.Kind() {
		case slog.KindString:
			return slog.String(key, MaskPhone(a.Value.String()))
		case slog.KindInt64, slog.KindUint64, slog.KindFloat64:
			return slog.String(key, MaskPhone(a.Value.String()))
		case slog.KindAny:
			val := a.Value.Any()
			if val != nil {
				return slog.String(key, MaskPhone(fmt.Sprint(val)))
			}
		}
	}

	switch a.Value.Kind() {
	case slog.KindString:
		return slog.String(key, h.scrubber.Scrub(a.Value.String()))

	case slog.KindGroup:
		attrs := a.Value.Group()
		scrubbed := make([]any, len(attrs))
		for i, sub := range attrs {
			scrubbed[i] = h.scrubAttr(sub)
		}
		return slog.Group(key, scrubbed...)

	case slog.KindAny:
		val := a.Value.Any()
		if val == nil {
			return a
		}
		switch v := val.(type) {
		case error:
			return slog.String(key, h.scrubber.Scrub(v.Error()))
		case fmt.Stringer:
			return slog.String(key, h.scrubber.Scrub(v.String()))
		case string:
			return slog.String(key, h.scrubber.Scrub(v))
		case []byte:
			return slog.String(key, h.scrubber.Scrub(string(v)))
		case map[string]any:
			subAttrs := make([]any, 0, len(v))
			for subK, subV := range v {
				subAttrs = append(subAttrs, h.scrubAttr(slog.Any(subK, subV)))
			}
			return slog.Group(key, subAttrs...)
		default:
			return a
		}

	default:
		return a
	}
}

func isSensitiveKey(key string) bool {
	return sensitiveKeyRegex.MatchString(key)
}

func isPhoneKey(key string) bool {
	return phoneKeyRegex.MatchString(key)
}

func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
