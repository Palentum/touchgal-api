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
