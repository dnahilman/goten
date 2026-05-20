package models

import "time"

type Account struct {
	ID         string    `json:"id"                   db:"id"`
	UserID     string    `json:"userId"               db:"user_id"`
	AccountID  string    `json:"accountId"            db:"account_id"`
	ProviderID string    `json:"providerId"           db:"provider_id"`
	Password   *string   `json:"password,omitempty"   db:"password"`
	CreatedAt  time.Time `json:"createdAt"            db:"created_at"`
	UpdatedAt  time.Time `json:"updatedAt"            db:"updated_at"`
}
