package mpesa

import (
	"encoding/base64"
	"time"
)

func generateTimestamp() string {
	return time.Now().Format("20060102150405")
}

func generatePassword(shortcode, passkey, timestamp string) string {
	raw := shortcode + passkey + timestamp

	return base64.StdEncoding.EncodeToString([]byte(raw))
}
