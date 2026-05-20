package goten_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	goten "github.com/dnahilman/goten"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError_Error(t *testing.T) {
	err := goten.ErrUnauthorized
	assert.Equal(t, "unauthorized", err.Error())
}

func TestAPIError_WriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	goten.ErrEmailExists.WriteJSON(w)

	resp := w.Result()
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok, "response should have 'error' key")
	assert.Equal(t, "EMAIL_EXISTS", errObj["code"])
	assert.Equal(t, "email already exists", errObj["message"])
	assert.InDelta(t, 409, errObj["status"], 0.1)
}

func TestSentinelErrors_StatusCodes(t *testing.T) {
	cases := []struct {
		err    *goten.APIError
		status int
	}{
		{goten.ErrInvalidEmail, 400},
		{goten.ErrInvalidPassword, 400},
		{goten.ErrEmailExists, 409},
		{goten.ErrUserNotFound, 404},
		{goten.ErrSessionExpired, 401},
		{goten.ErrSessionNotFound, 401},
		{goten.ErrUnauthorized, 401},
		{goten.ErrInternal, 500},
	}
	for _, c := range cases {
		t.Run(c.err.Code, func(t *testing.T) {
			assert.Equal(t, c.status, c.err.Status)
			assert.NotEmpty(t, c.err.Code)
			assert.NotEmpty(t, c.err.Message)
		})
	}
}
