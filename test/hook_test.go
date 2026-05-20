package goten_test

import (
	"errors"
	"testing"

	goten "github.com/dnahilman/goten"
	"github.com/stretchr/testify/assert"
)

func TestErrHookHandled_Sentinel(t *testing.T) {
	assert.True(t, errors.Is(goten.ErrHookHandled, goten.ErrHookHandled))
}

func TestErrHookHandled_WrappedDetected(t *testing.T) {
	wrapped := errors.Join(errors.New("outer"), goten.ErrHookHandled)
	assert.True(t, errors.Is(wrapped, goten.ErrHookHandled))
}

func TestErrHookHandled_OtherErrorNotDetected(t *testing.T) {
	other := errors.New("something else")
	assert.False(t, errors.Is(other, goten.ErrHookHandled))
}
