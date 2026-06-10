package middleware

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var benchmarkRateLimitSummary rateLimitSummary
var benchmarkRateLimitCounts [3]rateLimitCount

func BenchmarkSummarizeRateLimits(b *testing.B) {
	checks := []rateLimitCheck{
		{scope: "token", subject: uuid.New().String(), minuteLimit: 60, dayLimit: 5000},
		{scope: "user", subject: uuid.New().String(), minuteLimit: 120, dayLimit: 10000},
		{scope: "application", subject: uuid.New().String(), minuteLimit: 80, dayLimit: 8000},
	}
	counts := []rateLimitCount{
		{minuteCount: 7, dayCount: 700},
		{minuteCount: 8, dayCount: 701},
		{minuteCount: 9, dayCount: 702},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkRateLimitSummary = summarizeRateLimits(checks, counts)
	}
}

func BenchmarkIncrementLimitChecksRedis(b *testing.B) {
	addr := os.Getenv("REDIS_BENCH_ADDR")
	if addr == "" {
		b.Skip("set REDIS_BENCH_ADDR to run the Redis-backed rate-limit benchmark")
	}
	db, ok, err := redisBenchDB()
	if err != nil {
		b.Fatal(err)
	}
	if !ok {
		b.Skip("set REDIS_BENCH_DB to an isolated Redis database before running the Redis-backed benchmark")
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_BENCH_PASSWORD"),
		DB:       db,
	})
	defer client.Close()
	pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := client.Ping(pingCtx).Err(); err != nil {
		cancel()
		b.Fatalf("ping Redis benchmark instance: %v", err)
	}
	cancel()

	checks := []rateLimitCheck{
		{scope: "token", subject: uuid.New().String(), minuteLimit: 1 << 30, dayLimit: 1 << 30},
		{scope: "user", subject: uuid.New().String(), minuteLimit: 1 << 30, dayLimit: 1 << 30},
		{scope: "application", subject: uuid.New().String(), minuteLimit: 1 << 30, dayLimit: 1 << 30},
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counts, err := incrementLimitChecks(ctx, client, checks)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkRateLimitCounts = counts
	}
}

func redisBenchDB() (int, bool, error) {
	raw := os.Getenv("REDIS_BENCH_DB")
	if raw == "" {
		return 0, false, nil
	}
	db, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false, err
	}
	return db, true, nil
}
