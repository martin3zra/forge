package str

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
)

func Uuid() string {
	return uuid.NewString()
}

func GenerateRandom(byteLength ...int) string {
	var defaultByteLength = 64
	if len(byteLength) > 0 {
		defaultByteLength = byteLength[0]
	}
	// Create a byte slice of the desired length
	randomBytes := make([]byte, defaultByteLength)

	// Fill it with secure random data
	_, err := rand.Read(randomBytes)
	if err != nil {
		return ""
	}

	// Encode to base64
	return "base64:" + base64.StdEncoding.EncodeToString(randomBytes)
}
