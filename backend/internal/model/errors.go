package model

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrInvalidInput    = errors.New("invalid input")
	ErrRateLimited     = errors.New("rate limited")
	ErrConflict        = errors.New("conflict")
	ErrTooManyPending  = errors.New("too many pending applications")
	ErrCodeCooldown    = errors.New("verification code cooldown")
	ErrInvalidCode     = errors.New("invalid verification code")
	ErrExpiredCode     = errors.New("expired verification code")
	ErrApplicationOpen = errors.New("application is not approved")
)
