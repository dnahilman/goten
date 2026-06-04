package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

// ErrInvalidCiphertext is returned when a ciphertext cannot be decrypted
// (malformed, truncated, or authentication failed).
var ErrInvalidCiphertext = errors.New("crypto: invalid ciphertext")

// DeriveKey turns an arbitrary-length secret into a fixed 32-byte AES-256 key
// via SHA-256. The Config.Secret is validated to be at least 32 bytes elsewhere;
// this makes any such secret usable as an encryption key.
func DeriveKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

// Encrypt seals plaintext with AES-256-GCM and returns base64url(nonce || ciphertext).
// key must be 32 bytes (use DeriveKey). Used for opt-in OAuth token-at-rest encryption.
func Encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. Returns ErrInvalidCiphertext on any failure.
func Decrypt(ciphertext string, key []byte) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", ErrInvalidCiphertext
	}
	nonce, sealed := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	return string(plaintext), nil
}

// Sign returns "value.signature" where signature is base64url(HMAC-SHA256(value)).
// Used for tamper-evident cookies (e.g. the OAuth state nonce cookie).
func Sign(value, secret string) string {
	return value + "." + sign(value, secret)
}

// Verify validates a "value.signature" string produced by Sign and returns the
// value with ok=true when the signature matches (constant-time).
func Verify(signed, secret string) (string, bool) {
	i := strings.LastIndexByte(signed, '.')
	if i < 0 {
		return "", false
	}
	value, sig := signed[:i], signed[i+1:]
	expected := sign(value, secret)
	if subtle.ConstantTimeCompare([]byte(sig), []byte(expected)) != 1 {
		return "", false
	}
	return value, true
}

func sign(value, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
