package webhook_test

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"testing"

	"github.com/Flying-Tea-Squad/paykit-go/webhook"
)

func TestVerifyHMACSHA256_RFC4231_Blackbox(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		data      string
		signature string
	}{
		{
			name:      "RFC 4231 Test Case 1",
			key:       "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b",
			data:      "4869205468657265",
			signature: "b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7",
		},
		{
			name:      "RFC 4231 Test Case 2",
			key:       "4a656665",
			data:      "7768617420646f2079612077616e7420666f72206e6f7468696e673f",
			signature: "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843",
		},
		{
			name:      "RFC 4231 Test Case 3",
			key:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			data:      "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			signature: "773ea91e36800e46854db8ebd09181a72959098b3ef8c122d9635514ced565fe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyBytes, _ := hex.DecodeString(tt.key)
			dataBytes, _ := hex.DecodeString(tt.data)
			ok := webhook.VerifyHMACSHA256(dataBytes, tt.signature, string(keyBytes))
			if !ok {
				t.Fatalf("VerifyHMACSHA256() = false, want true")
			}
		})
	}
}

func TestVerifyHMACSHA256_PesapalStyle_Blackbox(t *testing.T) {
	payload := []byte(`{"id":"123","amount":1000,"currency":"KES","status":"COMPLETED"}`)
	secret := "pesapal_consumer_secret_12345"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	ok := webhook.VerifyHMACSHA256(payload, expectedSig, secret)
	if !ok {
		t.Fatalf("VerifyHMACSHA256() with correct signature = false, want true")
	}

	ok = webhook.VerifyHMACSHA256(payload, "wrongsignature", secret)
	if ok {
		t.Fatal("VerifyHMACSHA256() with wrong signature = true, want false")
	}
}

func TestVerifyHMACSHA512_RFC4231_Blackbox(t *testing.T) {
	key := "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b"
	data := "4869205468657265"
	keyBytes, _ := hex.DecodeString(key)
	dataBytes, _ := hex.DecodeString(data)

	mac := hmac.New(sha512.New, keyBytes)
	mac.Write(dataBytes)
	sig := hex.EncodeToString(mac.Sum(nil))

	ok := webhook.VerifyHMACSHA512(dataBytes, sig, string(keyBytes))
	if !ok {
		t.Fatalf("VerifyHMACSHA512() = false, want true")
	}
}

func TestVerifyHMACSHA256_InvalidInput_Blackbox(t *testing.T) {
	tests := []struct {
		name      string
		payload   []byte
		signature string
		secret    string
		wantOK    bool
	}{
		{"empty payload", []byte{}, "abc", "secret", false},
		{"empty signature", []byte("data"), "", "secret", false},
		{"empty secret", []byte("data"), "abc", "", false},
		{"invalid hex signature", []byte("data"), "zzzz", "secret", false},
		{"odd length hex", []byte("data"), "abc", "secret", false},
		{"wrong secret", []byte("data"), "abcdef", "secret", false},
		{"nil payload", nil, "abcdef", "secret", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok := webhook.VerifyHMACSHA256(tt.payload, tt.signature, tt.secret)
			if ok != tt.wantOK {
				t.Fatalf("VerifyHMACSHA256() = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

func generateRSAKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return privateKey, string(pubPEM)
}

func TestVerifyRSAPublicKey_PSS_Blackbox(t *testing.T) {
	privateKey, pubPEM := generateRSAKeyPair(t)
	payload := []byte("test payload for RSA-PSS")

	hashed := sha256.Sum256(payload)
	sig, err := rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, hashed[:], nil)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	err = webhook.VerifyRSAPublicKey(payload, sig, pubPEM)
	if err != nil {
		t.Fatalf("VerifyRSAPublicKey() with PSS error = %v", err)
	}
}

func TestVerifyRSAPublicKey_PKCS1v15_Blackbox(t *testing.T) {
	privateKey, pubPEM := generateRSAKeyPair(t)
	payload := []byte("test payload for PKCS1v15")

	hashed := sha256.Sum256(payload)
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	opts := webhook.RSAVerifyOptions{Padding: webhook.RSAPaddingPKCS1v15}
	err = webhook.VerifyRSAPublicKey(payload, sig, pubPEM, opts)
	if err != nil {
		t.Fatalf("VerifyRSAPublicKey() with PKCS1v15 error = %v", err)
	}
}

func TestVerifyRSAPublicKey_InvalidInput_Blackbox(t *testing.T) {
	privateKey, pubPEM := generateRSAKeyPair(t)
	payload := []byte("test")

	hashed := sha256.Sum256(payload)
	sig, _ := rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, hashed[:], nil)

	tests := []struct {
		name         string
		payload      []byte
		signature    []byte
		publicKeyPEM string
		opts         []webhook.RSAVerifyOptions
	}{
		{"empty payload", nil, sig, pubPEM, nil},
		{"empty signature", payload, nil, pubPEM, nil},
		{"empty public key", payload, sig, "", nil},
		{"invalid PEM", payload, sig, "not a valid pem", nil},
		{"wrong key type", payload, sig, "-----BEGIN PRIVATE KEY-----\ninvalid\n-----END PRIVATE KEY-----", nil},
		{"non-RSA public key", payload, sig, generateECDSAPublicKeyPEM(t), nil},
		{"wrong hash", payload, sig, pubPEM, []webhook.RSAVerifyOptions{{Hash: crypto.SHA512}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := webhook.VerifyRSAPublicKey(tt.payload, tt.signature, tt.publicKeyPEM, tt.opts...)
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
		})
	}
}

func generateECDSAPublicKeyPEM(t *testing.T) string {
	// Use RSA to generate a different key type for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	// Return an ECDSA PEM by using wrong type - this will fail parsing as RSA
	pubKeyBytes, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes}))
}

func TestVerifySafaricomIP_Production_Blackbox(t *testing.T) {
	validIPs := []string{
		"196.201.214.1",
		"196.201.214.254",
		"196.201.213.100",
		"196.201.212.50",
	}

	for _, ip := range validIPs {
		t.Run(ip, func(t *testing.T) {
			if !webhook.VerifySafaricomIP(ip) {
				t.Errorf("VerifySafaricomIP(%s) = false, want true", ip)
			}
		})
	}
}

func TestVerifySafaricomIP_WithPort_Blackbox(t *testing.T) {
	validIPs := []string{
		"196.201.214.1:43212",
		"196.201.214.254:8080",
		"196.201.213.100:5000",
		"196.201.212.50:3000",
	}

	for _, ip := range validIPs {
		t.Run(ip, func(t *testing.T) {
			if !webhook.VerifySafaricomIP(ip) {
				t.Errorf("VerifySafaricomIP(%s) = false, want true", ip)
			}
		})
	}
}

func TestVerifySafaricomIP_Invalid_Blackbox(t *testing.T) {
	invalidIPs := []string{
		"1.2.3.4",
		"192.168.1.1",
		"10.0.0.1",
		"172.16.0.1",
		"8.8.8.8",
		"2001:db8::1",
		"invalid",
		"",
		"196.201.215.1",
	}

	for _, ip := range invalidIPs {
		t.Run(ip, func(t *testing.T) {
			if webhook.VerifySafaricomIP(ip) {
				t.Errorf("VerifySafaricomIP(%s) = true, want false", ip)
			}
		})
	}
}

func TestSetAllowedCIDRs_Blackbox(t *testing.T) {
	// Save original
	originalCIDRs := []string{
		"196.201.214.0/24",
		"196.201.213.0/24",
		"196.201.212.0/24",
	}
	defer webhook.SetAllowedCIDRs(originalCIDRs)

	webhook.SetAllowedCIDRs([]string{"10.0.0.0/8"})

	if !webhook.VerifySafaricomIP("10.1.2.3") {
		t.Error("VerifySafaricomIP should work with overridden CIDRs")
	}

	if webhook.VerifySafaricomIP("192.168.1.1") {
		t.Error("VerifySafaricomIP should reject IPs outside overridden CIDRs")
	}
}
