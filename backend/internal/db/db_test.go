package db

import (
	"testing"
	"time"

	"github.com/touchgal/developer/backend/internal/config"
)

func TestPostgresPoolConfigAppliesPoolAndRuntimeSettings(t *testing.T) {
	cfg := config.PostgresConfig{
		PoolMaxConns:                    7,
		PoolMinConns:                    2,
		PoolMinIdleConns:                1,
		PoolMaxConnLifetime:             2 * time.Hour,
		PoolMaxConnIdleTime:             10 * time.Minute,
		PoolHealthCheckPeriod:           30 * time.Second,
		StatementTimeout:                30 * time.Second,
		IdleInTransactionSessionTimeout: 45 * time.Second,
	}

	poolConfig, err := postgresPoolConfig("postgres://user:pass@localhost:5432/app?sslmode=disable&application_name=touchgal-api&statement_timeout=999", cfg)
	if err != nil {
		t.Fatalf("postgresPoolConfig: %v", err)
	}

	if poolConfig.MaxConns != 7 || poolConfig.MinConns != 2 || poolConfig.MinIdleConns != 1 {
		t.Fatalf("unexpected pool sizes: max=%d min=%d minIdle=%d", poolConfig.MaxConns, poolConfig.MinConns, poolConfig.MinIdleConns)
	}
	if poolConfig.MaxConnLifetime != 2*time.Hour || poolConfig.MaxConnIdleTime != 10*time.Minute || poolConfig.HealthCheckPeriod != 30*time.Second {
		t.Fatalf("unexpected pool durations: lifetime=%s idle=%s health=%s", poolConfig.MaxConnLifetime, poolConfig.MaxConnIdleTime, poolConfig.HealthCheckPeriod)
	}
	runtimeParams := poolConfig.ConnConfig.Config.RuntimeParams
	if runtimeParams["application_name"] != "touchgal-api" {
		t.Fatalf("expected application_name runtime param to be preserved, got %q", runtimeParams["application_name"])
	}
	if runtimeParams["statement_timeout"] != "30000" {
		t.Fatalf("expected statement_timeout in milliseconds, got %q", runtimeParams["statement_timeout"])
	}
	if runtimeParams["idle_in_transaction_session_timeout"] != "45000" {
		t.Fatalf("expected idle_in_transaction_session_timeout in milliseconds, got %q", runtimeParams["idle_in_transaction_session_timeout"])
	}
}

func TestPostgresPoolConfigCanDisableRuntimeTimeouts(t *testing.T) {
	cfg := config.PostgresConfig{
		PoolMaxConns:                    1,
		PoolMaxConnLifetime:             time.Hour,
		PoolMaxConnIdleTime:             time.Minute,
		PoolHealthCheckPeriod:           time.Minute,
		StatementTimeout:                0,
		IdleInTransactionSessionTimeout: 0,
	}

	poolConfig, err := postgresPoolConfig("postgres://user:pass@localhost:5432/app?sslmode=disable&statement_timeout=999&idle_in_transaction_session_timeout=999", cfg)
	if err != nil {
		t.Fatalf("postgresPoolConfig: %v", err)
	}
	if _, ok := poolConfig.ConnConfig.Config.RuntimeParams["statement_timeout"]; ok {
		t.Fatal("expected statement_timeout to be removed when disabled")
	}
	if _, ok := poolConfig.ConnConfig.Config.RuntimeParams["idle_in_transaction_session_timeout"]; ok {
		t.Fatal("expected idle_in_transaction_session_timeout to be removed when disabled")
	}
}
func TestPostgresPoolConfigRoundsRuntimeTimeoutsUpToOneMillisecond(t *testing.T) {
	cfg := config.PostgresConfig{
		PoolMaxConns:                    1,
		PoolMaxConnLifetime:             time.Hour,
		PoolMaxConnIdleTime:             time.Minute,
		PoolHealthCheckPeriod:           time.Minute,
		StatementTimeout:                time.Nanosecond,
		IdleInTransactionSessionTimeout: time.Nanosecond,
	}

	poolConfig, err := postgresPoolConfig("postgres://user:pass@localhost:5432/app?sslmode=disable", cfg)
	if err != nil {
		t.Fatalf("postgresPoolConfig: %v", err)
	}
	if got := poolConfig.ConnConfig.Config.RuntimeParams["statement_timeout"]; got != "1" {
		t.Fatalf("expected sub-millisecond statement_timeout to round up to 1ms, got %q", got)
	}
	if got := poolConfig.ConnConfig.Config.RuntimeParams["idle_in_transaction_session_timeout"]; got != "1" {
		t.Fatalf("expected sub-millisecond idle timeout to round up to 1ms, got %q", got)
	}
}

func TestPostgresPingTimeoutUsesFallbackWhenQueryTimeoutDisabled(t *testing.T) {
	if got := postgresPingTimeout(0); got != postgresPingFallbackTimeout {
		t.Fatalf("expected fallback ping timeout, got %s", got)
	}
	if got := postgresPingTimeout(-time.Second); got != postgresPingFallbackTimeout {
		t.Fatalf("expected fallback ping timeout for negative duration, got %s", got)
	}
	if got := postgresPingTimeout(250 * time.Millisecond); got != 250*time.Millisecond {
		t.Fatalf("expected explicit query timeout, got %s", got)
	}
}

func TestRedisOptionsFromConfigAppliesSafeTimeoutDefaults(t *testing.T) {
	options := redisOptionsFromConfig(config.Config{})
	if options.DialTimeout != redisDefaultDialTimeout || options.ReadTimeout != redisDefaultReadTimeout || options.WriteTimeout != redisDefaultWriteTimeout || options.PoolTimeout != redisDefaultPoolTimeout {
		t.Fatalf("unexpected Redis timeout defaults: dial=%s read=%s write=%s pool=%s", options.DialTimeout, options.ReadTimeout, options.WriteTimeout, options.PoolTimeout)
	}
}

func TestRedisOptionsFromConfigAppliesPoolAndTimeoutSettings(t *testing.T) {
	cfg := config.Config{
		RedisAddr:         "redis.example.com:6380",
		RedisPassword:     "secret",
		RedisDB:           2,
		RedisPoolSize:     128,
		RedisMinIdleConns: 16,
		RedisDialTimeout:  250 * time.Millisecond,
		RedisReadTimeout:  750 * time.Millisecond,
		RedisWriteTimeout: time.Second,
		RedisPoolTimeout:  2 * time.Second,
	}

	options := redisOptionsFromConfig(cfg)
	if options.Addr != cfg.RedisAddr || options.Password != cfg.RedisPassword || options.DB != cfg.RedisDB {
		t.Fatalf("unexpected Redis endpoint options: %+v", options)
	}
	if options.PoolSize != cfg.RedisPoolSize || options.MinIdleConns != cfg.RedisMinIdleConns {
		t.Fatalf("unexpected Redis pool options: pool=%d minIdle=%d", options.PoolSize, options.MinIdleConns)
	}
	if options.DialTimeout != cfg.RedisDialTimeout || options.ReadTimeout != cfg.RedisReadTimeout || options.WriteTimeout != cfg.RedisWriteTimeout || options.PoolTimeout != cfg.RedisPoolTimeout {
		t.Fatalf("unexpected Redis timeout options: dial=%s read=%s write=%s pool=%s", options.DialTimeout, options.ReadTimeout, options.WriteTimeout, options.PoolTimeout)
	}
}
