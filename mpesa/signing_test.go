package mpesa

import (
	"encoding/base64"
	"testing"
)


//test file
func TestGeneratePassword(t *testing.T) {
	got := generatePassword(
		"174379",
		"abc123",
		"202607211103045",
	)
	want := base64.StdEncoding.EncodeToString(
		[]byte("174379abc123202607211103045"),
	)
	if got != want {
		t.Errorf("expected %q, got %q", want ,got)
	}
}
