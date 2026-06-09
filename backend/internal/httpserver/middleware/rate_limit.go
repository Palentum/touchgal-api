package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func APIRateLimit(redisClient *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info, ok := CurrentToken(r)
			if !ok {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid API token")
				return
			}
			minuteCount, dayCount, err := incrementLimits(r.Context(), redisClient, info.Token.ID.String())
			if err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
				return
			}
			minuteLimit := info.EffectiveMinuteLimit()
			dayLimit := info.EffectiveDailyLimit()
			minuteRemaining := minuteLimit - minuteCount
			dayRemaining := dayLimit - dayCount
			if minuteRemaining < 0 {
				minuteRemaining = 0
			}
			if dayRemaining < 0 {
				dayRemaining = 0
			}
			w.Header().Set("X-RateLimit-Limit-Minute", strconv.Itoa(minuteLimit))
			w.Header().Set("X-RateLimit-Remaining-Minute", strconv.Itoa(minuteRemaining))
			w.Header().Set("X-RateLimit-Limit-Day", strconv.Itoa(dayLimit))
			w.Header().Set("X-RateLimit-Remaining-Day", strconv.Itoa(dayRemaining))
			if minuteCount > minuteLimit || dayCount > dayLimit {
				writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "API rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func incrementLimits(ctx context.Context, client *redis.Client, tokenID string) (int, int, error) {
	now := time.Now().UTC()
	minuteKey := fmt.Sprintf("ratelimit:token:%s:minute:%s", tokenID, now.Format("200601021504"))
	dayKey := fmt.Sprintf("ratelimit:token:%s:day:%s", tokenID, now.Format("20060102"))
	pipe := client.TxPipeline()
	minute := pipe.Incr(ctx, minuteKey)
	pipe.Expire(ctx, minuteKey, 2*time.Minute)
	day := pipe.Incr(ctx, dayKey)
	pipe.Expire(ctx, dayKey, 48*time.Hour)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, 0, err
	}
	return int(minute.Val()), int(day.Val()), nil
}
