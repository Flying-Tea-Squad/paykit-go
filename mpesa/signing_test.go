package mpesa

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGeneratePassword(t *testing.T) {
	const (
		shortCode = "174379"
		passKey   = "abc123"
		timestamp = "20260721110304"
	)

	got := generatePassword(shortCode, passKey, timestamp)
	want := base64.StdEncoding.EncodeToString([]byte(shortCode + passKey + timestamp))

	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestGeneratePassword_Table(t *testing.T) {
	tests := []struct {
		name      string
		shortcode string
		passkey   string
		timestamp string
		want      string
	}{
		{
			name:      "standard inputs",
			shortcode: "174379",
			passkey:   "bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada1ed2c919",
			timestamp: "20260922130000",
			want:      base64.StdEncoding.EncodeToString([]byte("174379bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada1ed2c91920260922130000")),
		},
		{
			name:      "empty inputs",
			shortcode: "",
			passkey:   "",
			timestamp: "",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generatePassword(tt.shortcode, tt.passkey, tt.timestamp)
			if got != tt.want {
				t.Errorf("generatePassword(%q, %q, %q) = %q; want %q", tt.shortcode, tt.passkey, tt.timestamp, got, tt.want)
			}
		})
	}
}

func TestGenerateTimestamp(t *testing.T) {
	before := time.Now().Add(-time.Second)
	ts := generateTimestamp()
	after := time.Now().Add(time.Second)

	if len(ts) != 14 {
		t.Fatalf("expected timestamp to have 14 characters, got %d (%q)", len(ts), ts)
	}

	parsed, err := time.ParseInLocation("20060102150405", ts, time.Local)
	if err != nil {
		t.Fatalf("failed to parse timestamp %q with layout 20060102150405: %v", ts, err)
	}

	if parsed.Before(before.Truncate(time.Second)) || parsed.After(after.Add(time.Second)) {
		t.Errorf("timestamp %v outside expected range [%v, %v]", parsed, before, after)
	}
}

func TestFormatTimestamp_EAT(t *testing.T) {
	// Fixed UTC time: 2026-01-15 12:00:00 UTC
	fixedUTC := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	// Expected in EAT (UTC+3): 2026-01-15 15:00:00
	expected := "20260115150000"

	got := formatTimestamp(fixedUTC)
	if got != expected {
		t.Errorf("formatTimestamp(%v) = %q; want %q", fixedUTC, got, expected)
	}

	// Verify it's NOT in UTC (would be 20260115120000)
	gotUTC := fixedUTC.Format("20060102150405")
	if gotUTC == got {
		t.Error("output should be in EAT, not UTC")
	}
}

func TestGenerateTimestamp_EAT(t *testing.T) {
	// Integration test: verify generateTimestamp() uses formatTimestamp internally
	ts := generateTimestamp()

	// Parse as EAT
	eatLoc := time.FixedZone("EAT", 3*60*60)
	parsed, err := time.ParseInLocation("20060102150405", ts, eatLoc)
	if err != nil {
		t.Fatalf("failed to parse timestamp %q: %v", ts, err)
	}

	// Verify it's reasonable (within a few seconds of now in EAT)
	nowEAT := time.Now().In(eatLoc)
	diff := parsed.Sub(nowEAT)
	if diff < 0 {
		diff = -diff
	}
	if diff > 5*time.Second {
		t.Errorf("timestamp %v outside expected range (diff=%v)", parsed, diff)
	}

	// Verify it's NOT in UTC (would be 3 hours behind)
	parsedUTC, _ := time.Parse("20060102150405", ts)
	diffUTC := parsedUTC.Sub(nowEAT)
	if diffUTC < 0 {
		diffUTC = -diffUTC
	}
	if diffUTC <= 2*time.Hour {
		t.Error("timestamp should be in EAT, not UTC")
	}
}

// --- GenerateSecurityCredential tests ---

func TestGenerateSecurityCredential_X509Cert(t *testing.T) {
	privateKey, certPEM := generateTestCertificate(t)
	password := "TestInitiatorPassword123"

	credential, err := GenerateSecurityCredential(password, certPEM)
	require.NoError(t, err)
	require.NotEmpty(t, credential)

	// Verify round-trip decryption
	ciphertext, err := base64.StdEncoding.DecodeString(credential)
	require.NoError(t, err)
	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
	require.NoError(t, err)
	require.Equal(t, password, string(decrypted))
}

func TestGenerateSecurityCredential_PKIXKey(t *testing.T) {
	privateKey, pubPEM := generateTestPKIXKey(t)
	password := "TestInitiatorPassword123"

	credential, err := GenerateSecurityCredential(password, pubPEM)
	require.NoError(t, err)
	require.NotEmpty(t, credential)

	// Verify round-trip decryption
	ciphertext, err := base64.StdEncoding.DecodeString(credential)
	require.NoError(t, err)
	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
	require.NoError(t, err)
	require.Equal(t, password, string(decrypted))
}

func TestGenerateSecurityCredential_EmptyPassword(t *testing.T) {
	_, certPEM := generateTestCertificate(t)
	_, err := GenerateSecurityCredential("", certPEM)
	require.ErrorIs(t, err, ErrEmptyInitiatorPassword)
}

func TestGenerateSecurityCredential_EmptyCert(t *testing.T) {
	_, err := GenerateSecurityCredential("password", nil)
	require.ErrorIs(t, err, ErrEmptyCertificate)
}

func TestGenerateSecurityCredential_InvalidPEM(t *testing.T) {
	_, err := GenerateSecurityCredential("password", []byte("not a pem"))
	require.ErrorIs(t, err, ErrInvalidCertificatePEM)
}

func TestGenerateSecurityCredential_EncryptedPEM(t *testing.T) {
	// Create a PEM with Proc-Type header (encrypted)
	encryptedPEM := []byte(`-----BEGIN CERTIFICATE-----
Proc-Type: 4,ENCRYPTED
DEK-Info: DES-EDE3-CBC,1234567890ABCDEF

abcdefghijklmnop
-----END CERTIFICATE-----`)

	_, err := GenerateSecurityCredential("password", encryptedPEM)
	require.ErrorIs(t, err, ErrInvalidCertificatePEM)
}

func TestGenerateSecurityCredential_NonRSAKey(t *testing.T) {
	// Generate ECDSA key and create CERTIFICATE PEM
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Test ECDSA"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	_, err = GenerateSecurityCredential("password", certPEM)
	require.ErrorIs(t, err, ErrNonRSAKey)
}

func TestGenerateSecurityCredential_SmallRSAKey(t *testing.T) {
	// 1024-bit key (below minimum)
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes})

	_, err = GenerateSecurityCredential("password", pubPEM)
	require.ErrorIs(t, err, ErrNonRSAKey)
}

func TestGenerateSecurityCredential_SmallExponent(t *testing.T) {
	// Create a key with E=1 (invalid)
	// Use a valid modulus but E=1
	n := new(big.Int).SetUint64(1234567890123456789)
	privateKey := &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: n,
			E: 1, // Invalid exponent
		},
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes})

	_, err = GenerateSecurityCredential("password", pubPEM)
	require.ErrorIs(t, err, ErrNonRSAKey)
}

func TestGenerateSecurityCredential_PKCS1PublicKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// PKCS#1 format (not PKIX)
	pubKeyBytes := x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: pubKeyBytes})

	_, err = GenerateSecurityCredential("password", pubPEM)
	require.ErrorIs(t, err, ErrInvalidCertificatePEM)
	require.Contains(t, err.Error(), "PKCS#1 RSA PUBLIC KEY format not supported")
}

func TestGenerateSecurityCredential_PrivateKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// PKCS#8 private key
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privKeyBytes})

	_, err = GenerateSecurityCredential("password", privPEM)
	require.ErrorIs(t, err, ErrInvalidCertificatePEM)
	require.Contains(t, err.Error(), "private keys are not accepted")
}

func TestGenerateSecurityCredential_TrailingPEMBlock(t *testing.T) {
	_, certPEM := generateTestCertificate(t)
	_, pubPEM := generateTestPKIXKey(t)

	// Concatenate two PEM blocks
	combined := append(certPEM, pubPEM...)

	_, err := GenerateSecurityCredential("password", combined)
	require.ErrorIs(t, err, ErrInvalidCertificatePEM)
}

func TestGenerateSecurityCredential_TrailingGarbage(t *testing.T) {
	_, certPEM := generateTestCertificate(t)
	garbage := append(certPEM, []byte("trailing garbage")...)

	_, err := GenerateSecurityCredential("password", garbage)
	require.ErrorIs(t, err, ErrInvalidCertificatePEM)
}

func TestGenerateSecurityCredential_DeterministicRandomness(t *testing.T) {
	privateKey, certPEM := generateTestCertificate(t)
	password := "TestInitiatorPassword123"

	cred1, err := GenerateSecurityCredential(password, certPEM)
	require.NoError(t, err)

	cred2, err := GenerateSecurityCredential(password, certPEM)
	require.NoError(t, err)

	// PKCS#1 v1.5 encryption is randomized - same input should produce different ciphertexts
	require.NotEqual(t, cred1, cred2, "PKCS#1 v1.5 encryption should produce different ciphertexts each call")

	// But both should decrypt correctly
	ct1, _ := base64.StdEncoding.DecodeString(cred1)
	ct2, _ := base64.StdEncoding.DecodeString(cred2)
	dec1, _ := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ct1)
	dec2, _ := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ct2)
	require.Equal(t, password, string(dec1))
	require.Equal(t, password, string(dec2))
}

func TestGenerateSecurityCredential_NoTrailingNewline(t *testing.T) {
	_, certPEM := generateTestCertificate(t)

	passwordWithNewline := "password\n"
	passwordWithoutNewline := "password"

	credWithNL, _ := GenerateSecurityCredential(passwordWithNewline, certPEM)
	credWithoutNL, _ := GenerateSecurityCredential(passwordWithoutNewline, certPEM)

	require.NotEqual(t, credWithNL, credWithoutNL, "trailing newline should produce different credential")
}

// Test helpers

func generateTestCertificate(t *testing.T) (*rsa.PrivateKey, []byte) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Test CA"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return privateKey, certPEM
}

func generateTestPKIXKey(t *testing.T) (*rsa.PrivateKey, []byte) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes})
	return privateKey, pubPEM
}
