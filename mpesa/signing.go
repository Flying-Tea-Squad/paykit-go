package mpesa

import (
	"encoding/base64"
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

func generatePassword(shortcode, passkey, timestamp string) string {
	raw := shortcode + passkey + timestamp

	return base64.StdEncoding.EncodeToString([]byte(raw))
}
