package mpesa

import (
	"encoding/base64"
)

func generatePassword(shortCode, passKey, timeStamp string) string {
	// store the passwords into data
	data := shortCode + passKey + timeStamp
	return base64.StdEncoding.EncodeToString([]byte(data))
}