package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	DisplayName string     `json:"displayName"`
	Status      string     `json:"status"`
	IsAdmin     bool       `json:"isAdmin"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type EmailVerificationCode struct {
	ID         uuid.UUID
	Email      string
	Purpose    string
	CodeHash   string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	Attempts   int
	IP         string
	CreatedAt  time.Time
}

type Session struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	SessionHash string
	UserAgent   string
	IP          string
	ExpiresAt   time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
	LastSeenAt  *time.Time
}
