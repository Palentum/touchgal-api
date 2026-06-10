package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type LastUsedService interface {
	MarkLastUsed(ctx context.Context, id uuid.UUID) error
}

func APILastUsed(service LastUsedService, redisClient *redis.Client, logger zerolog.Logger, interval time.Duration) func(http.Handler) http.Handler {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if info, ok := CurrentToken(r); ok {
				tokenID := info.Token.ID
				if shouldUpdateLastUsed(r.Context(), redisClient, tokenID, interval, logger) {
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
						defer cancel()
						if err := service.MarkLastUsed(ctx, tokenID); err != nil {
							logger.Warn().Err(err).Str("token_id", tokenID.String()).Msg("api token last_used_at update failed")
						}
					}()
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func shouldUpdateLastUsed(ctx context.Context, redisClient *redis.Client, tokenID uuid.UUID, interval time.Duration, logger zerolog.Logger) bool {
	ok, err := redisClient.SetNX(ctx, "api:token:last_used:"+tokenID.String(), "1", interval).Result()
	if err != nil {
		logger.Warn().Err(err).Str("token_id", tokenID.String()).Msg("api token last_used_at throttle failed")
		return false
	}
	return ok
}
