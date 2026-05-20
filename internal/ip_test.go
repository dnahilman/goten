package internal_test

import (
	"net/http/httptest"
	"testing"

	"github.com/dnahilman/goten/internal"
	"github.com/stretchr/testify/assert"
)

func TestGetClientIP_XForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	assert.Equal(t, "1.2.3.4", internal.GetClientIP(r, ""))
}

func TestGetClientIP_CustomHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Custom-IP", "9.9.9.9")
	assert.Equal(t, "9.9.9.9", internal.GetClientIP(r, "X-Custom-IP"))
}

func TestGetClientIP_RemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.1:54321"
	assert.Equal(t, "192.168.1.1", internal.GetClientIP(r, ""))
}
