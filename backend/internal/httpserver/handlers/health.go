package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/touchgal/developer/backend/internal/repository"
)

const readinessCheckTimeout = 2 * time.Second

type readinessCheck struct {
	check func(context.Context) error
}

type HealthHandler struct {
	readinessChecks []readinessCheck
}

func NewHealthHandler(db repository.Queryer, redisClient *redis.Client) HealthHandler {
	checks := make([]readinessCheck, 0, 2)
	if db != nil {
		checks = append(checks, readinessCheck{
			check: func(ctx context.Context) error {
				_, err := db.Exec(ctx, "select 1")
				return err
			},
		})
	}
	if redisClient != nil {
		checks = append(checks, readinessCheck{
			check: func(ctx context.Context) error { return redisClient.Ping(ctx).Err() },
		})
	}
	return newHealthHandlerWithChecks(checks...)
}

func newHealthHandlerWithChecks(checks ...readinessCheck) HealthHandler {
	return HealthHandler{readinessChecks: checks}
}

func (HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	Success(w, http.StatusOK, map[string]string{"status": "ok", "version": "v1"})
}

func (h HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if len(h.readinessChecks) == 0 {
		ErrorCode(w, http.StatusServiceUnavailable, "NOT_READY", "Readiness checks are unavailable")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), readinessCheckTimeout)
	defer cancel()

	for _, check := range h.readinessChecks {
		if err := check.check(ctx); err != nil {
			ErrorCode(w, http.StatusServiceUnavailable, "NOT_READY", "Service dependencies are not ready")
			return
		}
	}
	Success(w, http.StatusOK, map[string]string{"status": "ready", "version": "v1"})
}
