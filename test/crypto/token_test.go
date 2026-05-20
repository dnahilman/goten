package crypto_test

import (
	"strings"
	"testing"

	"github.com/dnahilman/goten/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSessionToken_Prefix(t *testing.T) {
	tok, err := crypto.GenerateSessionToken()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(tok, crypto.Prefix), "token must start with %q, got %q", crypto.Prefix, tok)
}

func TestGenerateSessionToken_Length(t *testing.T) {
	tok, err := crypto.GenerateSessionToken()
	require.NoError(t, err)
	// Prefix(4) + base64url(32 bytes) = 4 + 43 = 47 chars minimum
	assert.GreaterOrEqual(t, len(tok), 40, "token too short: %s", tok)
}

func TestGenerateSessionToken_Unique(t *testing.T) {
	tokens := make(map[string]struct{}, 500)
	for i := 0; i < 500; i++ {
		tok, err := crypto.GenerateSessionToken()
		require.NoError(t, err)
		_, dup := tokens[tok]
		assert.False(t, dup, "duplicate token generated: %s", tok)
		tokens[tok] = struct{}{}
	}
}
