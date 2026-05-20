package httputil_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dnahilman/goten/internal/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"hello": "world"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "world", body["hello"])
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.WriteError(w, http.StatusBadRequest, "INVALID_BODY", "bad request")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "INVALID_BODY", body["code"])
	assert.Equal(t, "bad request", body["message"])
}

func TestDecodeJSON(t *testing.T) {
	body := `{"email":"a@b.com","password":"secret"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var out struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	require.NoError(t, httputil.DecodeJSON(r, &out))
	assert.Equal(t, "a@b.com", out.Email)
	assert.Equal(t, "secret", out.Password)
}

func TestDecodeJSON_Invalid(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not json"))
	var out map[string]any
	assert.Error(t, httputil.DecodeJSON(r, &out))
}
