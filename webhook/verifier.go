package webhook

import (
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/subtle"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
)

// ============================================================================
// HMAC Verification (timing-safe)
// ============================================================================

// VerifyHMAC verifies an HMAC signature using the specified hash function.
// signature is a hex-encoded string. secret is the raw secret key.
// Returns nil on success, error on mismatch or invalid input.
func VerifyHMAC(payload []byte, signature, secret string, h crypto.Hash) error {
	if len(payload) == 0 || signature == "" || secret == "" {
		return errors.New("webhook: missing payload, signature, or secret")
	}

	mac := hmac.New(h.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)

	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("webhook: invalid signature hex encoding: %w", err)
	}

	if subtle.ConstantTimeCompare(sigBytes, expected) != 1 {
		return errors.New("webhook: signature mismatch")
	}
	return nil
}

// VerifyHMACSHA256 verifies an HMAC-SHA256 signature (hex-encoded).
// signature is a hex-encoded string. secret is the raw secret key.
// Returns true if the signature is valid, false otherwise.
func VerifyHMACSHA256(payload []byte, signature, secret string) bool {
	return VerifyHMAC(payload, signature, secret, crypto.SHA256) == nil
}

// VerifyHMACSHA512 verifies an HMAC-SHA512 signature (hex-encoded).
// signature is a hex-encoded string. secret is the raw secret key.
// Returns true if the signature is valid, false otherwise.
func VerifyHMACSHA512(payload []byte, signature, secret string) bool {
	return VerifyHMAC(payload, signature, secret, crypto.SHA512) == nil
}

// ============================================================================
// RSA Verification (PSS default, PKCS#1 v1.5 optional)
// ============================================================================

// RSAPadding specifies the RSA padding scheme to use for signature verification.
type RSAPadding int

const (
	// RSAPaddingPSS uses RSA-PSS padding (default, recommended).
	RSAPaddingPSS RSAPadding = iota
	// RSAPaddingPKCS1v15 uses RSA PKCS#1 v1.5 padding.
	RSAPaddingPKCS1v15
)

// RSAVerifyOptions configures RSA signature verification behavior.
// Zero values use defaults: PSS padding with SHA-256.
type RSAVerifyOptions struct {
	// Padding specifies the RSA padding scheme. Defaults to RSAPaddingPSS.
	Padding RSAPadding
	// Hash specifies the hash algorithm used for signing. Defaults to crypto.SHA256.
	Hash crypto.Hash
}

// VerifyRSAPublicKey verifies an RSA signature using a PEM-encoded public key.
// signature is raw bytes (not hex). publicKeyPEM is PKIX PEM format.
// opts is optional; when omitted or empty, defaults to PSS padding with SHA-256.
// The payload is hashed using the specified hash function before verification.
func VerifyRSAPublicKey(payload, signature []byte, publicKeyPEM string, opts ...RSAVerifyOptions) error {
	if len(payload) == 0 || len(signature) == 0 || publicKeyPEM == "" {
		return errors.New("webhook: missing payload, signature, or public key")
	}

	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil || block.Type != "PUBLIC KEY" {
		return errors.New("webhook: invalid PEM public key")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return errors.New("webhook: failed to parse public key")
	}

	rsaPub, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("webhook: not an RSA public key")
	}

	h := crypto.SHA256
	padding := RSAPaddingPSS
	if len(opts) > 0 {
		if opts[0].Hash != 0 {
			h = opts[0].Hash
		}
		padding = opts[0].Padding
	}

	if !h.Available() {
		return fmt.Errorf("webhook: requested hash algorithm unavailable: %v", h)
	}

	hasher := h.New()
	hasher.Write(payload)
	hashedPayload := hasher.Sum(nil)

	if padding == RSAPaddingPSS {
		return rsa.VerifyPSS(rsaPub, h, hashedPayload, signature, nil)
	}
	return rsa.VerifyPKCS1v15(rsaPub, h, hashedPayload, signature)
}

// ============================================================================
// Safaricom Daraja IP Whitelist
// ============================================================================

// IPVerifierConfig holds the parsed CIDR blocks for IP verification.
// It is safe for concurrent use.
type IPVerifierConfig struct {
	mu    sync.RWMutex
	cidrs []*net.IPNet
}

var defaultIPVerifier = &IPVerifierConfig{}

func init() {
	// Pre-parse default Safaricom Daraja CIDR ranges at package initialization
	defaultSafaricomCIDRs := []string{
		"196.201.214.0/24",
		"196.201.213.0/24",
		"196.201.212.0/24",
	}
	cidrs := make([]*net.IPNet, 0, len(defaultSafaricomCIDRs))
	for _, c := range defaultSafaricomCIDRs {
		_, network, err := net.ParseCIDR(c)
		if err == nil {
			cidrs = append(cidrs, network)
		}
	}
	defaultIPVerifier.cidrs = cidrs
}

// IPVerifierOption configures an IPVerifierConfig.
type IPVerifierOption func(*IPVerifierConfig)

// WithAllowedCIDRs sets the allowed CIDR ranges for IP verification.
// CIDRs are parsed once at configuration time for efficiency.
func WithAllowedCIDRs(cidrs []string) IPVerifierOption {
	return func(v *IPVerifierConfig) {
		v.mu.Lock()
		defer v.mu.Unlock()
		parsed := make([]*net.IPNet, 0, len(cidrs))
		for _, c := range cidrs {
			_, network, err := net.ParseCIDR(c)
			if err == nil {
				parsed = append(parsed, network)
			}
		}
		v.cidrs = parsed
	}
}

// VerifySafaricomIP returns true if remoteIP matches any allowed CIDR ranges.
// remoteIP should be the client IP (e.g., from r.RemoteAddr). It accepts both
// bare IPs ("196.201.214.1") and host:port formats ("196.201.214.1:43212").
func VerifySafaricomIP(remoteIP string) bool {
	return defaultIPVerifier.Verify(remoteIP)
}

// Verify checks if the given IP matches any allowed CIDR ranges.
func (v *IPVerifierConfig) Verify(remoteIP string) bool {
	host := strings.TrimSpace(remoteIP)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	v.mu.RLock()
	cidrs := v.cidrs
	v.mu.RUnlock()

	for _, network := range cidrs {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// SetAllowedCIDRs updates the allowed CIDR ranges for the default IP verifier.
// This is a convenience function for tests and simple configurations.
// For more control, create a custom IPVerifierConfig and use its Verify method.
func SetAllowedCIDRs(cidrs []string) {
	WithAllowedCIDRs(cidrs)(defaultIPVerifier)
}
