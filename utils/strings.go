package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

// IsStringEmpty checks if given string is empty
func IsStringEmpty(s string) bool { return len(strings.TrimSpace(s)) == 0 }

// RandomString generates random string og given length
func RandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("utils: failed to generate random token")
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
