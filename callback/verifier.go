package callback

import (
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/subtle"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net"
	"strings"
)

// ============================================================================
// HMAC Verification (timing-safe)
// ============================================================================

// VerifyHMAC verifies an HMAC signature using the specified hash function.
// signature and secret are hex-encoded strings.
// Returns nil on success, error on mismatch or invalid input.
func VerifyHMAC(payload []byte, signature, secret string, h crypto.Hash) error {
	if len(payload) == 0 || signature == "" || secret == "" {
		return errors.New("callback: missing payload, signature, or secret")
	}

	mac := hmac.New(h.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)

	sigBytes, err := decodeHex(signature)
	if err != nil {
		return errors.New("callback: invalid signature hex encoding")
	}

	if subtle.ConstantTimeCompare(sigBytes, expected) != 1 {
		return errors.New("callback: signature mismatch")
	}
	return nil
}

// VerifyHMACSHA256 verifies HMAC-SHA256 signature (hex-encoded).
func VerifyHMACSHA256(payload []byte, signature, secret string) error {
	return VerifyHMAC(payload, signature, secret, crypto.SHA256)
}

// VerifyHMACSHA512 verifies HMAC-SHA512 signature (hex-encoded).
func VerifyHMACSHA512(payload []byte, signature, secret string) error {
	return VerifyHMAC(payload, signature, secret, crypto.SHA512)
}

// ============================================================================
// RSA Verification (PSS default, PKCS#1 v1.5 optional)
// ============================================================================

type RSAPadding int

const (
	RSAPaddingPSS RSAPadding = iota
	RSAPaddingPKCS1v15
)

type RSAVerifyOptions struct {
	Padding RSAPadding
	Hash    crypto.Hash
}

// VerifyRSAPublicKey verifies an RSA signature using a PEM-encoded public key.
// signature is raw bytes (not hex). publicKeyPEM is PKIX PEM format.
// opts can be nil for defaults (PSS + SHA256).
func VerifyRSAPublicKey(payload, signature []byte, publicKeyPEM string, opts *RSAVerifyOptions) error {
	if len(payload) == 0 || len(signature) == 0 || publicKeyPEM == "" {
		return errors.New("callback: missing payload, signature, or public key")
	}

	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil || block.Type != "PUBLIC KEY" {
		return errors.New("callback: invalid PEM public key")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return errors.New("callback: failed to parse public key")
	}

	rsaPub, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("callback: not an RSA public key")
	}

	h := crypto.SHA256
	padding := RSAPaddingPSS
	if opts != nil {
		if opts.Hash != 0 {
			h = opts.Hash
		}
		padding = opts.Padding
	}

	if padding == RSAPaddingPSS {
		return rsa.VerifyPSS(rsaPub, h, payload, signature, nil)
	}
	return rsa.VerifyPKCS1v15(rsaPub, h, payload, signature)
}

// ============================================================================
// Safaricom Daraja IP Whitelist
// ============================================================================

// SafaricomCIDRs contains the CIDR ranges for Safaricom Daraja callback servers.
// Override in tests or for future range changes.
var SafaricomCIDRs = []string{
	"196.201.214.0/24",
	"196.201.213.0/24",
	"196.201.212.0/24",
}

// VerifySafaricomIP returns true if remoteIP matches any SafaricomCIDRs.
// remoteIP should be the client IP from r.RemoteAddr (host only, no port).
func VerifySafaricomIP(remoteIP string) bool {
	ip := net.ParseIP(remoteIP)
	if ip == nil {
		return false
	}
	for _, cidr := range SafaricomCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

// ============================================================================
// Helpers
// ============================================================================

func decodeHex(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if len(s)%2 != 0 {
		return nil, errors.New("odd length")
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		var v byte
		if s[i] >= '0' && s[i] <= '9' {
			v = (s[i] - '0') << 4
		} else if s[i] >= 'a' && s[i] <= 'f' {
			v = (s[i] - 'a' + 10) << 4
		} else if s[i] >= 'A' && s[i] <= 'F' {
			v = (s[i] - 'A' + 10) << 4
		} else {
			return nil, errors.New("invalid hex")
		}
		if s[i+1] >= '0' && s[i+1] <= '9' {
			v |= s[i+1] - '0'
		} else if s[i+1] >= 'a' && s[i+1] <= 'f' {
			v |= s[i+1] - 'a' + 10
		} else if s[i+1] >= 'A' && s[i+1] <= 'F' {
			v |= s[i+1] - 'A' + 10
		} else {
			return nil, errors.New("invalid hex")
		}
		b[i/2] = v
	}
	return b, nil
}
