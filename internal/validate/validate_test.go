package validate_test

import (
	"testing"

	"github.com/dnahilman/goten/internal/validate"
	"github.com/stretchr/testify/assert"
)

func TestIsValidEmail(t *testing.T) {
	valid := []string{"a@b.com", "user+tag@example.org", "x@x.io"}
	for _, e := range valid {
		assert.True(t, validate.IsValidEmail(e), "expected valid: %s", e)
	}

	invalid := []string{"", "notanemail", "@b.com", "a@", "a @b.com", "a@b"}
	for _, e := range invalid {
		assert.False(t, validate.IsValidEmail(e), "expected invalid: %s", e)
	}
}

func TestPassword(t *testing.T) {
	assert.NoError(t, validate.Password("secret123", 8, 72))
	assert.Error(t, validate.Password("short", 8, 72))
	assert.Error(t, validate.Password(string(make([]byte, 73)), 8, 72))
}
