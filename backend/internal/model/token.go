package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	TokenActive   = "active"
	TokenDisabled = "disabled"
	TokenRevoked  = "revoked"
)

type APIToken struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"userId"`
	ApplicationID uuid.UUID  `json:"applicationId"`
	Name          string     `json:"name"`
	TokenPrefix   string     `json:"tokenPrefix"`
	TokenHash     string     `json:"-"`
	Status        string     `json:"status"`
	MinuteLimit   int        `json:"minuteLimit"`
	DailyLimit    int        `json:"dailyLimit"`
	LastUsedAt    *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	RevokedAt     *time.Time `json:"revokedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type TokenAuthInfo struct {
	Token             APIToken
	ApplicationStatus string
	UserID            uuid.UUID
	ApplicationID     uuid.UUID
}
