package mpesa

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"time"
)

// eatZone is East Africa Time (UTC+3) — no DST, fixed offset.
// Uses time.FixedZone to avoid external tzdata dependency in minimal containers.
// Must be a package-level var (not const) because time.FixedZone returns *time.Location.
var eatZone = time.FixedZone("EAT", 3*60*60)

// formatTimestamp formats the given time in East Africa Time (UTC+3) as YYYYMMDDHHMMSS.
// Exists to enable deterministic testing with fixed timestamps regardless of host timezone.
func formatTimestamp(t time.Time) string {
	return t.In(eatZone).Format("20060102150405")
}

// generateTimestamp returns the current time formatted as YYYYMMDDHHMMSS in East Africa Time (UTC+3).
// Used for STK Push password generation and other Daraja timestamps.
func generateTimestamp() string {
	return formatTimestamp(time.Now())
}

// GenerateSecurityCredentialErrors for type-safe error handling
var (
	ErrEmptyInitiatorPassword = errors.New("mpesa: initiator password cannot be empty")
	ErrEmptyCertificate       = errors.New("mpesa: certificate PEM cannot be empty")
	ErrInvalidCertificatePEM  = errors.New("mpesa: invalid certificate PEM")
	ErrNonRSAKey              = errors.New("mpesa: certificate does not contain a valid RSA public key")
	ErrEncryptionFailed       = errors.New("mpesa: security credential encryption failed")
)

// GenerateSecurityCredential encrypts the initiator password using Safaricom's X.509
// public key certificate with RSA PKCS#1 v1.5 padding and returns the Base64-encoded ciphertext.
//
// This implements the Daraja SecurityCredential required for B2C, Reversal, Account Balance,
// and Transaction Status APIs.
//
// Parameters:
//   - initiatorPassword: The plaintext initiator password (e.g., "testapi" password from Daraja portal).
//     Must not be empty. Must not contain trailing newlines (use printf '%s' when generating from files).
//   - certPEM: Safaricom's public certificate in PEM format. Accepts either:
//   - "CERTIFICATE" block (X.509 certificate) — parsed via x509.ParseCertificate
//   - "PUBLIC KEY" block (PKIX public key) — parsed via x509.ParsePKIXPublicKey
//     Must not be empty and must contain a valid RSA public key. Only the first PEM block is used;
//     trailing blocks or garbage bytes after the first block cause an error. Encrypted PEM blocks
//     (with Proc-Type headers) are rejected. Private keys and PKCS#1 "RSA PUBLIC KEY" format
//     are not accepted.
//
// Returns:
//   - Base64-encoded RSA PKCS#1 v1.5 encrypted password (single line, no newlines)
//   - Error if inputs are invalid, PEM parsing fails, key is not RSA, key < 2048 bits, or encryption fails.
//
// Security Notes:
//   - Uses crypto/rand.Reader for encryption randomness (Go 1.26+ ignores reader, uses secure default).
//   - PKCS#1 v1.5 encryption is required by Daraja API specification (not OAEP).
//   - Encryption is not vulnerable to Bleichenbacher attacks (those target decryption, not encryption).
//   - Caller should cache the result; re-encrypting per request adds latency without benefit.
//
// Example:
//
//	certPEM := `-----BEGIN CERTIFICATE-----
//	MIID...
//	-----END CERTIFICATE-----`
//	credential, err := mpesa.GenerateSecurityCredential("MyInitiatorPassword", []byte(certPEM))
func GenerateSecurityCredential(initiatorPassword string, certPEM []byte) (string, error) {
	// 1. Validate inputs
	if initiatorPassword == "" {
		return "", ErrEmptyInitiatorPassword
	}
	if len(certPEM) == 0 {
		return "", ErrEmptyCertificate
	}

	// 2. Decode first PEM block only
	block, rest := pem.Decode(certPEM)
	if block == nil {
		return "", ErrInvalidCertificatePEM
	}

	// Reject encrypted PEM blocks (e.g., Proc-Type: 4,ENCRYPTED headers)
	if len(block.Headers) > 0 {
		return "", fmt.Errorf("%w: encrypted PEM blocks are not supported", ErrInvalidCertificatePEM)
	}

	// Reject trailing PEM blocks or garbage bytes
	if len(bytes.TrimSpace(rest)) > 0 {
		return "", ErrInvalidCertificatePEM
	}

	var rsaPub *rsa.PublicKey

	// 3. Parse based on PEM block type — explicit cases for clear errors
	switch block.Type {
	case "CERTIFICATE":
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrInvalidCertificatePEM, err)
		}
		pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return "", ErrNonRSAKey
		}
		rsaPub = pubKey

	case "PUBLIC KEY": // PKIX (SPKI) format
		pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrInvalidCertificatePEM, err)
		}
		pubKeyRSA, ok := pubKey.(*rsa.PublicKey)
		if !ok {
			return "", ErrNonRSAKey
		}
		rsaPub = pubKeyRSA

	case "RSA PUBLIC KEY": // PKCS#1 format — not accepted
		return "", fmt.Errorf("%w: PKCS#1 RSA PUBLIC KEY format not supported; use PKIX PUBLIC KEY or CERTIFICATE", ErrInvalidCertificatePEM)

	case "PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY": // Private keys — never accepted
		return "", fmt.Errorf("%w: private keys are not accepted; provide a public certificate or public key", ErrInvalidCertificatePEM)

	default:
		return "", fmt.Errorf("%w: unsupported PEM block type %q", ErrInvalidCertificatePEM, block.Type)
	}

	// 4. Validate RSA public key structure and minimum key size
	// rsaPub.E must be >= 2 (crypto/rsa returns ErrPublicExponentSmall during encryption)
	if rsaPub == nil || rsaPub.N == nil || rsaPub.E < 2 {
		return "", ErrNonRSAKey
	}
	if rsaPub.N.BitLen() < 2048 {
		return "", fmt.Errorf("%w: RSA key size %d bits is too small (minimum 2048)", ErrNonRSAKey, rsaPub.N.BitLen())
	}

	// 5. Encrypt with RSA PKCS#1 v1.5
	// Note: Daraja requires PKCS#1 v1.5, not OAEP. This is encryption (not decryption),
	// so Bleichenbacher padding oracle attacks do not apply.
	//nolint:staticcheck // Required by Safaricom Daraja API specification (PKCS#1 v1.5 encryption)
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPub, []byte(initiatorPassword))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
	}

	// 6. Return Base64-encoded result (single line, no newlines)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func generatePassword(shortcode, passkey, timestamp string) string {
	raw := shortcode + passkey + timestamp

	return base64.StdEncoding.EncodeToString([]byte(raw))
}
