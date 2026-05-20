package crypto_test

import (
	"testing"

	"github.com/dnahilman/goten/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword_Roundtrip(t *testing.T) {
	pwd := "correcthorsebatterystaple"
	hash, err := crypto.HashPassword(pwd)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, pwd, hash)

	ok, err := crypto.VerifyPassword(hash, pwd)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, err := crypto.HashPassword("correct")
	require.NoError(t, err)

	ok, err := crypto.VerifyPassword(hash, "wrong")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestHashPassword_DifferentHashPerCall(t *testing.T) {
	pwd := "samepassword"
	h1, err := crypto.HashPassword(pwd)
	require.NoError(t, err)
	h2, err := crypto.HashPassword(pwd)
	require.NoError(t, err)
	assert.NotEqual(t, h1, h2, "bcrypt should produce different hashes due to random salt")
}
