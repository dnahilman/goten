package goten

import (
	"encoding/json"
	"net/http"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e *APIError) Error() string { return e.Message }

func (e *APIError) WriteJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": e})
}

var (
	ErrInvalidToken    = &APIError{Code: "INVALID_TOKEN", Message: "invalid session token", Status: 401}
	ErrInvalidEmail    = &APIError{Code: "INVALID_EMAIL", Message: "invalid email", Status: 400}
	ErrInvalidPassword = &APIError{Code: "INVALID_CREDENTIALS", Message: "invalid email or password", Status: 400}
	ErrEmailExists     = &APIError{Code: "EMAIL_EXISTS", Message: "email already exists", Status: 409}
	ErrUserNotFound    = &APIError{Code: "USER_NOT_FOUND", Message: "user not found", Status: 404}
	ErrSessionExpired  = &APIError{Code: "SESSION_EXPIRED", Message: "session expired", Status: 401}
	ErrSessionNotFound = &APIError{Code: "SESSION_NOT_FOUND", Message: "session not found", Status: 401}
	ErrUnauthorized    = &APIError{Code: "UNAUTHORIZED", Message: "unauthorized", Status: 401}
	ErrInternal        = &APIError{Code: "INTERNAL", Message: "internal server error", Status: 500}
)
