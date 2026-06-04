package models

import "time"

// Verification is a generic key-value record with an expiry, keyed by Identifier.
// OAuth sign-in stores its state here; other flows (e.g. email verification) may reuse it.
type Verification struct {
	ID         string    `json:"id"         db:"id"`
	Identifier string    `json:"identifier" db:"identifier"`
	Value      string    `json:"value"      db:"value"`
	ExpiresAt  time.Time `json:"expiresAt"  db:"expires_at"`
	CreatedAt  time.Time `json:"createdAt"  db:"created_at"`
	UpdatedAt  time.Time `json:"updatedAt"  db:"updated_at"`
}
