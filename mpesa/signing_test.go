package mpesa

import (
	"encoding/base64"
	"testing"
	"time"
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
