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
	Token                  APIToken
	ApplicationStatus      string
	UserID                 uuid.UUID
	UserMinuteLimit        int
	UserDailyLimit         int
	ApplicationID          uuid.UUID
	ApplicationMinuteLimit int
	ApplicationDailyLimit  int
}

func (i TokenAuthInfo) EffectiveMinuteLimit() int {
	return minPositiveLimit(i.Token.MinuteLimit, i.UserMinuteLimit, i.ApplicationMinuteLimit)
}

func (i TokenAuthInfo) EffectiveDailyLimit() int {
	return minPositiveLimit(i.Token.DailyLimit, i.UserDailyLimit, i.ApplicationDailyLimit)
}

func minPositiveLimit(base, first, second int) int {
	limit := base
	if first > 0 && (limit <= 0 || first < limit) {
		limit = first
	}
	if second > 0 && (limit <= 0 || second < limit) {
		limit = second
	}
	return limit
}
