package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func RandomToken(bytes int) (string, error) {
	if bytes <= 0 {
		return "", fmt.Errorf("invalid token length")
	}
	data := make([]byte, bytes)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
