package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/touchgal/developer/backend/internal/model"
)

var rateLimitCounterScript = redis.NewScript(`
local checkCount = tonumber(ARGV[1])
local minuteTTL = tonumber(ARGV[2])
local dayTTL = tonumber(ARGV[3])
local counts = {}
for i = 1, checkCount do
	local minuteIndex = (i - 1) * 2 + 1
	local dayIndex = minuteIndex + 1
	local minuteCount = redis.call("INCR", KEYS[minuteIndex])
	if minuteCount == 1 then
		redis.call("EXPIRE", KEYS[minuteIndex], minuteTTL)
	end
	local dayCount = redis.call("INCR", KEYS[dayIndex])
	if dayCount == 1 then
		redis.call("EXPIRE", KEYS[dayIndex], dayTTL)
	end
	counts[minuteIndex] = minuteCount
	counts[dayIndex] = dayCount
end
return counts
`)

func APIPreAuthRateLimit(redisClient *redis.Client, minuteLimit, dailyLimit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ClientIP(r)
			if ip == "" {
				ip = "unknown"
			}
			minuteCount, dayCount, err := incrementLimits(r.Context(), redisClient, "ip", ip)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
				return
			}
			if minuteCount > minuteLimit || dayCount > dailyLimit {
				writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "API rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func APIRateLimit(redisClient *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info, ok := CurrentToken(r)
			if !ok {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid API token")
				return
			}
			checks := [3]rateLimitCheck{{
				scope:       "token",
				subject:     info.Token.ID.String(),
				minuteLimit: info.Token.MinuteLimit,
				dayLimit:    info.Token.DailyLimit,
			}}
			checkCount := 1
			if userID := tokenUserID(info); userID != uuid.Nil && (info.UserMinuteLimit > 0 || info.UserDailyLimit > 0) {
				checks[checkCount] = rateLimitCheck{
					scope:       "user",
					subject:     userID.String(),
					minuteLimit: info.UserMinuteLimit,
					dayLimit:    info.UserDailyLimit,
				}
				checkCount++
			}
			if applicationID := tokenApplicationID(info); applicationID != uuid.Nil && (info.ApplicationMinuteLimit > 0 || info.ApplicationDailyLimit > 0) {
				checks[checkCount] = rateLimitCheck{
					scope:       "application",
					subject:     applicationID.String(),
					minuteLimit: info.ApplicationMinuteLimit,
					dayLimit:    info.ApplicationDailyLimit,
				}
				checkCount++
			}
			activeChecks := checks[:checkCount]
			counts, err := incrementLimitChecks(r.Context(), redisClient, activeChecks)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
				return
			}
			summary := summarizeRateLimits(activeChecks, counts[:len(activeChecks)])
			writeRateLimitHeaders(w, summary.minuteLimit, summary.minuteRemaining, summary.dayLimit, summary.dayRemaining)
			if summary.exceeded {
				writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "API rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeRateLimitHeaders(w http.ResponseWriter, minuteLimit, minuteRemaining, dayLimit, dayRemaining int) {
	w.Header().Set("X-RateLimit-Limit-Minute", strconv.Itoa(minuteLimit))
	w.Header().Set("X-RateLimit-Remaining-Minute", strconv.Itoa(minuteRemaining))
	w.Header().Set("X-RateLimit-Limit-Day", strconv.Itoa(dayLimit))
	w.Header().Set("X-RateLimit-Remaining-Day", strconv.Itoa(dayRemaining))
}

type rateLimitCheck struct {
	scope       string
	subject     string
	minuteLimit int
	dayLimit    int
}

type rateLimitCount struct {
	minuteCount int
	dayCount    int
}

type rateLimitSummary struct {
	minuteLimit     int
	minuteRemaining int
	dayLimit        int
	dayRemaining    int
	exceeded        bool
}

func tokenUserID(info *model.TokenAuthInfo) uuid.UUID {
	if info.UserID != uuid.Nil {
		return info.UserID
	}
	return info.Token.UserID
}

func tokenApplicationID(info *model.TokenAuthInfo) uuid.UUID {
	if info.ApplicationID != uuid.Nil {
		return info.ApplicationID
	}
	return info.Token.ApplicationID
}

func summarizeRateLimits(checks []rateLimitCheck, counts []rateLimitCount) rateLimitSummary {
	var summary rateLimitSummary
	minuteSet := false
	daySet := false
	for i := range checks {
		check := checks[i]
		count := counts[i]
		if check.minuteLimit > 0 {
			remaining := check.minuteLimit - count.minuteCount
			if remaining < 0 {
				remaining = 0
			}
			if !minuteSet || check.minuteLimit < summary.minuteLimit {
				summary.minuteLimit = check.minuteLimit
			}
			if !minuteSet || remaining < summary.minuteRemaining {
				summary.minuteRemaining = remaining
			}
			if count.minuteCount > check.minuteLimit {
				summary.exceeded = true
			}
			minuteSet = true
		}
		if check.dayLimit > 0 {
			remaining := check.dayLimit - count.dayCount
			if remaining < 0 {
				remaining = 0
			}
			if !daySet || check.dayLimit < summary.dayLimit {
				summary.dayLimit = check.dayLimit
			}
			if !daySet || remaining < summary.dayRemaining {
				summary.dayRemaining = remaining
			}
			if count.dayCount > check.dayLimit {
				summary.exceeded = true
			}
			daySet = true
		}
	}
	return summary
}

func incrementLimits(ctx context.Context, client *redis.Client, scope, subject string) (int, int, error) {
	checks := [1]rateLimitCheck{{scope: scope, subject: subject}}
	counts, err := incrementLimitChecks(ctx, client, checks[:])
	if err != nil {
		return 0, 0, err
	}
	return counts[0].minuteCount, counts[0].dayCount, nil
}

func incrementLimitChecks(ctx context.Context, client *redis.Client, checks []rateLimitCheck) ([3]rateLimitCount, error) {
	now := time.Now().UTC()
	minuteBucket := now.Format("200601021504")
	dayBucket := now.Format("20060102")
	keys := make([]string, 0, len(checks)*2)
	for i := range checks {
		keys = append(keys,
			"ratelimit:"+checks[i].scope+":"+checks[i].subject+":minute:"+minuteBucket,
			"ratelimit:"+checks[i].scope+":"+checks[i].subject+":day:"+dayBucket,
		)
	}
	values, err := rateLimitCounterScript.Run(ctx, client, keys, len(checks), int((2 * time.Minute).Seconds()), int((48 * time.Hour).Seconds())).Result()
	if err != nil {
		return [3]rateLimitCount{}, err
	}
	rawCounts, ok := values.([]interface{})
	if !ok {
		return [3]rateLimitCount{}, fmt.Errorf("unexpected rate limit counter result %T", values)
	}
	if len(rawCounts) != len(checks)*2 {
		return [3]rateLimitCount{}, fmt.Errorf("unexpected rate limit counter result length %d", len(rawCounts))
	}
	var counts [3]rateLimitCount
	for i := range checks {
		minuteCount, err := redisInt(rawCounts[i*2])
		if err != nil {
			return [3]rateLimitCount{}, err
		}
		dayCount, err := redisInt(rawCounts[i*2+1])
		if err != nil {
			return [3]rateLimitCount{}, err
		}
		counts[i] = rateLimitCount{minuteCount: minuteCount, dayCount: dayCount}
	}
	return counts, nil
}

func redisInt(value any) (int, error) {
	switch typed := value.(type) {
	case int64:
		return int(typed), nil
	case int:
		return typed, nil
	case string:
		parsed, err := strconv.Atoi(typed)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unexpected Redis integer type %T", value)
	}
}
