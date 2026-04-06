package entity

import "time"

// User entity
type User struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Password   string    `json:"password,omitempty"` // stored as hash
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	Phone      string    `json:"phone"`
	AvatarPath string    `json:"avatarPath"`
	Role       string    `json:"role"`
	LastOnline time.Time `json:"last_online"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
