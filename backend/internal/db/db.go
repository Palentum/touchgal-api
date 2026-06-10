package db

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/touchgal/developer/backend/internal/config"
)

const (
	postgresPingFallbackTimeout = 10 * time.Second
	redisDefaultDialTimeout     = 5 * time.Second
	redisDefaultReadTimeout     = 3 * time.Second
	redisDefaultWriteTimeout    = 3 * time.Second
	redisDefaultPoolTimeout     = 4 * time.Second
)

func OpenPostgres(ctx context.Context, dsn string, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	poolConfig, err := postgresPoolConfig(dsn, cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, postgresPingTimeout(cfg.QueryTimeout))
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func postgresPoolConfig(dsn string, cfg config.PostgresConfig) (*pgxpool.Config, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	poolConfig.MaxConns = int32(cfg.PoolMaxConns)
	poolConfig.MinConns = int32(cfg.PoolMinConns)
	poolConfig.MinIdleConns = int32(cfg.PoolMinIdleConns)
	poolConfig.MaxConnLifetime = cfg.PoolMaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.PoolMaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.PoolHealthCheckPeriod
	setRuntimeDuration(poolConfig.ConnConfig.Config.RuntimeParams, "statement_timeout", cfg.StatementTimeout)
	setRuntimeDuration(poolConfig.ConnConfig.Config.RuntimeParams, "idle_in_transaction_session_timeout", cfg.IdleInTransactionSessionTimeout)
	return poolConfig, nil
}

func postgresPingTimeout(queryTimeout time.Duration) time.Duration {
	if queryTimeout > 0 {
		return queryTimeout
	}
	return postgresPingFallbackTimeout
}

func contextWithOptionalTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func setRuntimeDuration(params map[string]string, key string, d time.Duration) {
	if d <= 0 {
		delete(params, key)
		return
	}
	milliseconds := d.Milliseconds()
	if milliseconds == 0 {
		milliseconds = 1
	}
	params[key] = strconv.FormatInt(milliseconds, 10)
}

func OpenOptionalPostgres(ctx context.Context, dsn string, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, nil
	}
	return OpenPostgres(ctx, dsn, cfg)
}

func redisTimeoutOrDefault(value, fallback time.Duration) time.Duration {
	if value == 0 {
		return fallback
	}
	return value
}

func redisOptionsFromConfig(cfg config.Config) *redis.Options {
	return &redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     cfg.RedisPoolSize,
		MinIdleConns: cfg.RedisMinIdleConns,
		DialTimeout:  redisTimeoutOrDefault(cfg.RedisDialTimeout, redisDefaultDialTimeout),
		ReadTimeout:  redisTimeoutOrDefault(cfg.RedisReadTimeout, redisDefaultReadTimeout),
		WriteTimeout: redisTimeoutOrDefault(cfg.RedisWriteTimeout, redisDefaultWriteTimeout),
		PoolTimeout:  redisTimeoutOrDefault(cfg.RedisPoolTimeout, redisDefaultPoolTimeout),
	}
}

func OpenRedis(ctx context.Context, cfg config.Config) (*redis.Client, error) {
	client := redis.NewClient(redisOptionsFromConfig(cfg))
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}
