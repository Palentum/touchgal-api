package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/touchgal/developer/backend/internal/model"
)

type successResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

type errorResponse struct {
	Success bool          `json:"success"`
	Error   apiErrorShape `json:"error"`
}

type apiErrorShape struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Success(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(successResponse{Success: true, Data: data})
}

func Error(w http.ResponseWriter, err error) {
	status, code, message := classify(err)
	ErrorCode(w, status, code, message)
}

func ErrorCode(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Success: false, Error: apiErrorShape{Code: code, Message: message}})
}

func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func classify(err error) (int, string, string) {
	switch {
	case errors.Is(err, model.ErrInvalidInput):
		return http.StatusBadRequest, "BAD_REQUEST", "Invalid request parameters"
	case errors.Is(err, model.ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid credentials"
	case errors.Is(err, model.ErrAccountDisabled):
		return http.StatusForbidden, "ACCOUNT_DISABLED", "Account is disabled"
	case errors.Is(err, model.ErrForbidden):
		return http.StatusForbidden, "FORBIDDEN", "Permission denied"
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND", "Resource not found"
	case errors.Is(err, model.ErrRateLimited):
		return http.StatusTooManyRequests, "RATE_LIMITED", "API rate limit exceeded"
	case errors.Is(err, model.ErrApplicationExists):
		return http.StatusConflict, "CONFLICT", "Application already submitted"
	case errors.Is(err, model.ErrCodeCooldown):
		return http.StatusTooManyRequests, "RATE_LIMITED", "Please wait before requesting another code"
	case errors.Is(err, model.ErrInvalidCode), errors.Is(err, model.ErrExpiredCode):
		return http.StatusBadRequest, "BAD_REQUEST", "Invalid or expired verification code"
	case errors.Is(err, model.ErrConflict):
		return http.StatusConflict, "CONFLICT", "Resource already exists"
	case errors.Is(err, model.ErrApplicationOpen):
		return http.StatusForbidden, "FORBIDDEN", "Application is not approved"
	case errors.Is(err, model.ErrSyncRunning):
		return http.StatusConflict, "SYNC_RUNNING", "Sync is already running"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error"
	}
}
