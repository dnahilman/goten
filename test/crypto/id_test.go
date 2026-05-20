package crypto_test

import (
	"strings"
	"testing"

	"github.com/dnahilman/goten/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewID_Prefix(t *testing.T) {
	id := crypto.NewID()
	assert.True(t, strings.HasPrefix(id, crypto.Prefix), "ID must start with %q, got %q", crypto.Prefix, id)
}

func TestNewID_Format(t *testing.T) {
	id := crypto.NewID()
	// format: g10_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx  (4 + 36 = 40 chars)
	require.Equal(t, 40, len(id), "ID length should be 40 chars, got %d: %s", len(id), id)
	uuidPart := strings.TrimPrefix(id, crypto.Prefix)
	parts := strings.Split(uuidPart, "-")
	require.Equal(t, 5, len(parts), "UUID part should have 5 segments separated by '-'")
}

func TestNewID_Unique(t *testing.T) {
	ids := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id := crypto.NewID()
		_, dup := ids[id]
		assert.False(t, dup, "duplicate ID generated: %s", id)
		ids[id] = struct{}{}
	}
}

func TestNewID_TimeSortable(t *testing.T) {
	a := crypto.NewID()
	b := crypto.NewID()
	assert.LessOrEqual(t, a, b, "IDs should be lexicographically ordered (time-sortable)")
}
