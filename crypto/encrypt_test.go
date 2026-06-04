package crypto

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := DeriveKey("a-secret-that-is-at-least-32-bytes!!")
	plaintext := "ya29.super-secret-access-token"

	enc, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc == plaintext {
		t.Errorf("ciphertext equals plaintext")
	}

	dec, err := Decrypt(enc, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if dec != plaintext {
		t.Errorf("roundtrip mismatch: %q != %q", dec, plaintext)
	}
}

func TestEncryptNonceUnique(t *testing.T) {
	key := DeriveKey("a-secret-that-is-at-least-32-bytes!!")
	a, _ := Encrypt("same", key)
	b, _ := Encrypt("same", key)
	if a == b {
		t.Errorf("expected unique ciphertexts (random nonce)")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	enc, _ := Encrypt("secret", DeriveKey("key-one-that-is-at-least-32-bytes!!!"))
	if _, err := Decrypt(enc, DeriveKey("key-two-that-is-at-least-32-bytes!!!")); err == nil {
		t.Errorf("expected error decrypting with wrong key")
	}
}

func TestDecryptMalformed(t *testing.T) {
	key := DeriveKey("a-secret-that-is-at-least-32-bytes!!")
	if _, err := Decrypt("not-base64!!", key); err == nil {
		t.Errorf("expected error for malformed ciphertext")
	}
	if _, err := Decrypt("YWJj", key); err == nil { // valid b64, too short for nonce
		t.Errorf("expected error for truncated ciphertext")
	}
}

func TestSignVerify(t *testing.T) {
	secret := "signing-secret"
	signed := Sign("state-token-123", secret)
	value, ok := Verify(signed, secret)
	if !ok || value != "state-token-123" {
		t.Errorf("verify failed: value=%q ok=%v", value, ok)
	}
}

func TestVerifyTampered(t *testing.T) {
	secret := "signing-secret"
	signed := Sign("state-token-123", secret)
	if _, ok := Verify(signed+"x", secret); ok {
		t.Errorf("tampered signature should not verify")
	}
	if _, ok := Verify("state-token-123.deadbeef", secret); ok {
		t.Errorf("forged signature should not verify")
	}
	if _, ok := Verify("no-dot-here", secret); ok {
		t.Errorf("malformed signed value should not verify")
	}
}
