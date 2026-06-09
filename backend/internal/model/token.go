package model

import (
	"time"

	"github.com/google/uuid"
)

const TokenActive = "active"

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
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type TokenAuthInfo struct {
	Token             APIToken
	ApplicationStatus string
	UserID            uuid.UUID
	UserMinuteLimit   int
	UserDailyLimit    int
	ApplicationID     uuid.UUID
}

func (i TokenAuthInfo) EffectiveMinuteLimit() int {
	if i.UserMinuteLimit > 0 && i.UserMinuteLimit < i.Token.MinuteLimit {
		return i.UserMinuteLimit
	}
	return i.Token.MinuteLimit
}

func (i TokenAuthInfo) EffectiveDailyLimit() int {
	if i.UserDailyLimit > 0 && i.UserDailyLimit < i.Token.DailyLimit {
		return i.UserDailyLimit
	}
	return i.Token.DailyLimit
}
