package callback

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"testing"
)

func TestVerifyHMACSHA256_RFC4231(t *testing.T) {
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
			err := VerifyHMACSHA256(dataBytes, tt.signature, string(keyBytes))
			if err != nil {
				t.Fatalf("VerifyHMACSHA256() error = %v, want nil", err)
			}
		})
	}
}

func TestVerifyHMACSHA256_PesapalStyle(t *testing.T) {
	payload := []byte(`{"id":"123","amount":1000,"currency":"KES","status":"COMPLETED"}`)
	secret := "pesapal_consumer_secret_12345"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	err := VerifyHMACSHA256(payload, expectedSig, secret)
	if err != nil {
		t.Fatalf("VerifyHMACSHA256() with correct signature error = %v", err)
	}

	err = VerifyHMACSHA256(payload, "wrongsignature", secret)
	if err == nil {
		t.Fatal("VerifyHMACSHA256() with wrong signature should fail")
	}
}

func TestVerifyHMACSHA512_RFC4231(t *testing.T) {
	key := "0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b0b"
	data := "4869205468657265"
	keyBytes, _ := hex.DecodeString(key)
	dataBytes, _ := hex.DecodeString(data)

	mac := hmac.New(sha512.New, keyBytes)
	mac.Write(dataBytes)
	sig := hex.EncodeToString(mac.Sum(nil))

	err := VerifyHMACSHA512(dataBytes, sig, string(keyBytes))
	if err != nil {
		t.Fatalf("VerifyHMACSHA512() error = %v", err)
	}
}

func TestVerifyHMAC_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		payload   []byte
		signature string
		secret    string
		wantError bool
	}{
		{"empty payload", []byte{}, "abc", "secret", true},
		{"empty signature", []byte("data"), "", "secret", true},
		{"empty secret", []byte("data"), "abc", "", true},
		{"invalid hex signature", []byte("data"), "zzzz", "secret", true},
		{"odd length hex", []byte("data"), "abc", "secret", true},
		{"wrong secret", []byte("data"), "abcdef", "secret", true},
		{"nil payload", nil, "abcdef", "secret", true},
		{"invalid hex first char", []byte("data"), "g0", "secret", true},
		{"invalid hex second char", []byte("data"), "0g", "secret", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyHMACSHA256(tt.payload, tt.signature, tt.secret)
			if tt.wantError && err == nil {
				t.Fatalf("expected error but got nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestVerifyHMAC_ConstantTime(t *testing.T) {
	secret := "testsecret"
	payload := []byte("test payload")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	correctSig := hex.EncodeToString(mac.Sum(nil))

	wrongSig := correctSig[:len(correctSig)-1] + "0"
	if wrongSig == correctSig {
		wrongSig = correctSig[:len(correctSig)-1] + "1"
	}

	err1 := VerifyHMACSHA256(payload, correctSig, secret)
	err2 := VerifyHMACSHA256(payload, wrongSig, secret)

	if err1 != nil {
		t.Fatalf("correct signature should pass: %v", err1)
	}
	if err2 == nil {
		t.Fatal("wrong signature should fail")
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

func TestVerifyRSAPublicKey_PSS(t *testing.T) {
	privateKey, pubPEM := generateRSAKeyPair(t)
	payload := []byte("test payload for RSA-PSS")

	hashed := sha256.Sum256(payload)
	sig, err := rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, hashed[:], nil)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	err = VerifyRSAPublicKey(payload, sig, pubPEM, nil)
	if err != nil {
		t.Fatalf("VerifyRSAPublicKey() with PSS error = %v", err)
	}
}

func TestVerifyRSAPublicKey_PKCS1v15(t *testing.T) {
	privateKey, pubPEM := generateRSAKeyPair(t)
	payload := []byte("test payload for PKCS1v15")

	hashed := sha256.Sum256(payload)
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	opts := &RSAVerifyOptions{Padding: RSAPaddingPKCS1v15}
	err = VerifyRSAPublicKey(payload, sig, pubPEM, opts)
	if err != nil {
		t.Fatalf("VerifyRSAPublicKey() with PKCS1v15 error = %v", err)
	}
}

func encodePEM(blockType string, data []byte) string {
	return string(pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: data}))
}

func generateECDSAPublicKeyPEM(t *testing.T) string {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ECDSA key: %v", err)
	}
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal ECDSA public key: %v", err)
	}
	return encodePEM("PUBLIC KEY", pubKeyBytes)
}

func TestVerifyRSAPublicKey_CrossPaddingFails(t *testing.T) {
	privateKey, pubPEM := generateRSAKeyPair(t)
	payload := []byte("test payload")

	hashed := sha256.Sum256(payload)
	pssSig, _ := rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, hashed[:], nil)

	opts := &RSAVerifyOptions{Padding: RSAPaddingPKCS1v15}
	err := VerifyRSAPublicKey(payload, pssSig, pubPEM, opts)
	if err == nil {
		t.Fatal("PKCS1v15 verification of PSS signature should fail")
	}
}

func TestVerifyRSAPublicKey_Errors(t *testing.T) {
	privateKey, pubPEM := generateRSAKeyPair(t)
	payload := []byte("test")

	hashed := sha256.Sum256(payload)
	sig, _ := rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, hashed[:], nil)

	tests := []struct {
		name         string
		payload      []byte
		signature    []byte
		publicKeyPEM string
		opts         *RSAVerifyOptions
	}{
		{"empty payload", nil, sig, pubPEM, nil},
		{"empty signature", payload, nil, pubPEM, nil},
		{"empty public key", payload, sig, "", nil},
		{"invalid PEM", payload, sig, "not a valid pem", nil},
		{"wrong key type", payload, sig, "-----BEGIN PRIVATE KEY-----\ninvalid\n-----END PRIVATE KEY-----", nil},
		{"invalid PKIX bytes", payload, sig, encodePEM("PUBLIC KEY", []byte{0x01, 0x02, 0x03}), nil},
		{"non-RSA public key", payload, sig, generateECDSAPublicKeyPEM(t), nil},
		{"wrong hash", payload, sig, pubPEM, &RSAVerifyOptions{Hash: crypto.SHA512}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyRSAPublicKey(tt.payload, tt.signature, tt.publicKeyPEM, tt.opts)
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
		})
	}
}

func TestVerifySafaricomIP_Production(t *testing.T) {
	validIPs := []string{
		"196.201.214.1",
		"196.201.214.254",
		"196.201.213.100",
		"196.201.212.50",
	}

	for _, ip := range validIPs {
		t.Run(ip, func(t *testing.T) {
			if !VerifySafaricomIP(ip) {
				t.Errorf("VerifySafaricomIP(%s) = false, want true", ip)
			}
		})
	}
}

func TestVerifySafaricomIP_Invalid(t *testing.T) {
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
			if VerifySafaricomIP(ip) {
				t.Errorf("VerifySafaricomIP(%s) = true, want false", ip)
			}
		})
	}
}

func TestSafaricomCIDRs_Override(t *testing.T) {
	original := SafaricomCIDRs
	defer func() { SafaricomCIDRs = original }()

	SafaricomCIDRs = []string{"10.0.0.0/8"}

	if !VerifySafaricomIP("10.1.2.3") {
		t.Error("VerifySafaricomIP should work with overridden CIDRs")
	}

	if VerifySafaricomIP("192.168.1.1") {
		t.Error("VerifySafaricomIP should reject IPs outside overridden CIDRs")
	}
}
