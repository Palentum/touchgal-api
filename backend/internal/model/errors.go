package model

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidInput       = errors.New("invalid input")
	ErrRateLimited        = errors.New("rate limited")
	ErrConflict           = errors.New("conflict")
	ErrApplicationExists  = errors.New("application already submitted")
	ErrCodeCooldown       = errors.New("verification code cooldown")
	ErrInvalidCode        = errors.New("invalid verification code")
	ErrExpiredCode        = errors.New("expired verification code")
	ErrApplicationOpen    = errors.New("application is not approved")
	ErrAccountDisabled    = errors.New("account disabled")
	ErrSyncRunning        = errors.New("sync is already running")
	ErrSyncDisabled       = errors.New("sync is disabled")
	ErrTokenLimitExceeded = errors.New("api token limit exceeded")
)
