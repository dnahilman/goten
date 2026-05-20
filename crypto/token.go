package crypto

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateSessionToken returns "g10_<base64url(32 random bytes)>".
func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return Prefix + base64.RawURLEncoding.EncodeToString(b), nil
}
