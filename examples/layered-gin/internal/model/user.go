package model

import "time"

// UserProfile extends Goten's auth user (goten_users) with app-specific
// fields. UserID is a foreign key to goten_users.id.
type UserProfile struct {
	UserID    string    `gorm:"primaryKey"           json:"user_id"`
	FullName  string    `gorm:"not null"             json:"full_name"`
	Phone     string    `                            json:"phone,omitempty"`
	Role      string    `gorm:"default:customer"     json:"role"`
	CreatedAt time.Time `                            json:"created_at"`
	UpdatedAt time.Time `                            json:"updated_at"`
}

func (UserProfile) TableName() string { return "app_user_profiles" }
