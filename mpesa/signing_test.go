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
