package config

import (
	"testing"
	"time"
)

func TestLoadDefaultsSyncWorkerDisabled(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("ENABLE_SYNC_WORKER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected default config to load: %v", err)
	}
	if cfg.EnableSyncWorker {
		t.Fatal("expected sync worker to be disabled by default")
	}
}

func TestLoadHTTPServerDefaults(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "")
	t.Setenv("HTTP_READ_TIMEOUT", "")
	t.Setenv("HTTP_WRITE_TIMEOUT", "")
	t.Setenv("HTTP_IDLE_TIMEOUT", "")
	t.Setenv("HTTP_MAX_HEADER_BYTES", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected default config to load: %v", err)
	}
	if cfg.HTTPReadHeaderTimeout != 10*time.Second {
		t.Fatalf("unexpected read header timeout: %s", cfg.HTTPReadHeaderTimeout)
	}
	if cfg.HTTPReadTimeout != 15*time.Second {
		t.Fatalf("unexpected read timeout: %s", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPWriteTimeout != 60*time.Second {
		t.Fatalf("unexpected write timeout: %s", cfg.HTTPWriteTimeout)
	}
	if cfg.HTTPIdleTimeout != 120*time.Second {
		t.Fatalf("unexpected idle timeout: %s", cfg.HTTPIdleTimeout)
	}
	if cfg.HTTPMaxHeaderBytes != 1<<20 {
		t.Fatalf("unexpected max header bytes: %d", cfg.HTTPMaxHeaderBytes)
	}
}

func TestLoadHTTPServerEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "2s")
	t.Setenv("HTTP_READ_TIMEOUT", "3s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "40s")
	t.Setenv("HTTP_IDLE_TIMEOUT", "5s")
	t.Setenv("HTTP_MAX_HEADER_BYTES", "4096")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load: %v", err)
	}
	if cfg.HTTPReadHeaderTimeout != 2*time.Second || cfg.HTTPReadTimeout != 3*time.Second || cfg.HTTPWriteTimeout != 40*time.Second || cfg.HTTPIdleTimeout != 5*time.Second || cfg.HTTPMaxHeaderBytes != 4096 {
		t.Fatalf("unexpected HTTP server config: %+v", cfg)
	}
}

func TestLoadObservabilityDefaultsAndEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("ENABLE_PPROF", "")
	t.Setenv("ENABLE_METRICS", "")
	t.Setenv("OBSERVABILITY_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected default config to load: %v", err)
	}
	if cfg.EnablePprof || cfg.EnableMetrics || cfg.ObservabilityAddr != "127.0.0.1:6060" {
		t.Fatalf("unexpected observability defaults: %+v", cfg)
	}

	t.Setenv("ENABLE_PPROF", "true")
	t.Setenv("ENABLE_METRICS", "true")
	t.Setenv("OBSERVABILITY_ADDR", "127.0.0.1:7070")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("expected env config to load: %v", err)
	}
	if !cfg.EnablePprof || !cfg.EnableMetrics || cfg.ObservabilityAddr != "127.0.0.1:7070" {
		t.Fatalf("unexpected observability env config: %+v", cfg)
	}
}

func TestValidateHTTPServerSettings(t *testing.T) {
	cfg := validConfig()
	cfg.HTTPReadHeaderTimeout = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid HTTP_READ_HEADER_TIMEOUT error")
	}

	cfg = validConfig()
	cfg.HTTPReadTimeout = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid HTTP_READ_TIMEOUT error")
	}

	cfg = validConfig()
	cfg.HTTPWriteTimeout = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid HTTP_WRITE_TIMEOUT error")
	}

	cfg = validConfig()
	cfg.HTTPIdleTimeout = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid HTTP_IDLE_TIMEOUT error")
	}

	cfg = validConfig()
	cfg.HTTPMaxHeaderBytes = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid HTTP_MAX_HEADER_BYTES error")
	}
	cfg = validConfig()
	cfg.HTTPWriteTimeout = cfg.HTTPReadTimeout + cfg.DatabasePool.QueryTimeout
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected HTTP_WRITE_TIMEOUT not greater than HTTP_READ_TIMEOUT plus DB_QUERY_TIMEOUT error")
	}
}

func TestValidateObservabilitySettings(t *testing.T) {
	cfg := validConfig()
	cfg.EnablePprof = true
	cfg.ObservabilityAddr = "127.0.0.1:6060"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected loopback observability address to be valid: %v", err)
	}

	cfg = validConfig()
	cfg.EnableMetrics = true
	cfg.ObservabilityAddr = "10.0.0.10:6060"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected private observability address to be valid: %v", err)
	}

	cfg = validConfig()
	cfg.EnablePprof = true
	cfg.ObservabilityAddr = ":6060"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected wildcard observability address error")
	}

	cfg = validConfig()
	cfg.EnableMetrics = true
	cfg.ObservabilityAddr = "8.8.8.8:6060"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected public observability address error")
	}
}

func TestValidateMailDriverPostalRequiresAPISettings(t *testing.T) {
	cfg := validConfig()
	cfg.MailDriver = MailDriverPostal
	cfg.PostalAPIURL = "https://postal.example.com"
	cfg.PostalAPIKey = "secret"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid Postal config: %v", err)
	}

	missingURL := cfg
	missingURL.PostalAPIURL = ""
	if err := missingURL.Validate(); err == nil {
		t.Fatal("expected missing POSTAL_API_URL error")
	}

	missingKey := cfg
	missingKey.PostalAPIKey = ""
	if err := missingKey.Validate(); err == nil {
		t.Fatal("expected missing POSTAL_API_KEY error")
	}

	plainHTTP := cfg
	plainHTTP.PostalAPIURL = "http://postal.example.com"
	if err := plainHTTP.Validate(); err == nil {
		t.Fatal("expected non-https POSTAL_API_URL error")
	}

	relativeURL := cfg
	relativeURL.PostalAPIURL = "/api/v1/send/message"
	if err := relativeURL.Validate(); err == nil {
		t.Fatal("expected absolute POSTAL_API_URL error")
	}
}

func TestValidateMailDriverRejectsUnknownDriver(t *testing.T) {
	cfg := validConfig()
	cfg.MailDriver = "sendmail"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid MAIL_DRIVER error")
	}
}

func TestLoadMailSendTimeoutDefaultAndEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("MAIL_SEND_TIMEOUT_SECONDS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected default config to load: %v", err)
	}
	if cfg.MailSendTimeoutSeconds != 10 || cfg.MailSendTimeout() != 10*time.Second {
		t.Fatalf("unexpected mail send timeout default: seconds=%d duration=%s", cfg.MailSendTimeoutSeconds, cfg.MailSendTimeout())
	}

	t.Setenv("MAIL_SEND_TIMEOUT_SECONDS", "3")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("expected env config to load: %v", err)
	}
	if cfg.MailSendTimeoutSeconds != 3 || cfg.MailSendTimeout() != 3*time.Second {
		t.Fatalf("unexpected mail send timeout from env: seconds=%d duration=%s", cfg.MailSendTimeoutSeconds, cfg.MailSendTimeout())
	}
}

func TestValidateMailSendTimeoutPositive(t *testing.T) {
	cfg := validConfig()
	cfg.MailSendTimeoutSeconds = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid MAIL_SEND_TIMEOUT_SECONDS error")
	}
}

func TestValidateRejectsUnknownLogLevel(t *testing.T) {
	cfg := validConfig()
	cfg.LogLevel = "verbose"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid LOG_LEVEL error")
	}
}

func TestLoadRedisDefaultsAndEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, key := range []string{
		"REDIS_ADDR",
		"REDIS_PASSWORD",
		"REDIS_DB",
		"REDIS_POOL_SIZE",
		"REDIS_MIN_IDLE_CONNS",
		"REDIS_DIAL_TIMEOUT",
		"REDIS_READ_TIMEOUT",
		"REDIS_WRITE_TIMEOUT",
		"REDIS_POOL_TIMEOUT",
	} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected default config to load: %v", err)
	}
	if cfg.RedisAddr != "localhost:6379" || cfg.RedisPassword != "" || cfg.RedisDB != 0 {
		t.Fatalf("unexpected Redis endpoint defaults: addr=%q password=%q db=%d", cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	}
	if cfg.RedisPoolSize != 0 || cfg.RedisMinIdleConns != 0 || cfg.RedisDialTimeout != 0 || cfg.RedisReadTimeout != 0 || cfg.RedisWriteTimeout != 0 || cfg.RedisPoolTimeout != 0 {
		t.Fatalf("expected Redis pool/timeout defaults to defer to go-redis, got %+v", cfg)
	}

	t.Setenv("REDIS_ADDR", "redis.example.com:6380")
	t.Setenv("REDIS_PASSWORD", "secret")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_POOL_SIZE", "128")
	t.Setenv("REDIS_MIN_IDLE_CONNS", "16")
	t.Setenv("REDIS_DIAL_TIMEOUT", "250ms")
	t.Setenv("REDIS_READ_TIMEOUT", "750ms")
	t.Setenv("REDIS_WRITE_TIMEOUT", "1s")
	t.Setenv("REDIS_POOL_TIMEOUT", "2s")

	cfg, err = Load()
	if err != nil {
		t.Fatalf("expected env config to load: %v", err)
	}
	if cfg.RedisAddr != "redis.example.com:6380" || cfg.RedisPassword != "secret" || cfg.RedisDB != 2 {
		t.Fatalf("unexpected Redis endpoint env config: addr=%q password=%q db=%d", cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	}
	if cfg.RedisPoolSize != 128 || cfg.RedisMinIdleConns != 16 || cfg.RedisDialTimeout != 250*time.Millisecond || cfg.RedisReadTimeout != 750*time.Millisecond || cfg.RedisWriteTimeout != time.Second || cfg.RedisPoolTimeout != 2*time.Second {
		t.Fatalf("unexpected Redis pool/timeout env config: %+v", cfg)
	}
}

func TestLoadTokenLimitDefaultsAndEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("MAX_ACTIVE_TOKENS_PER_USER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected default config to load: %v", err)
	}
	if cfg.MaxActiveTokensPerUser != DefaultMaxActiveTokensPerUser {
		t.Fatalf("unexpected max active token default: %d", cfg.MaxActiveTokensPerUser)
	}

	t.Setenv("MAX_ACTIVE_TOKENS_PER_USER", "3")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("expected env config to load: %v", err)
	}
	if cfg.MaxActiveTokensPerUser != 3 {
		t.Fatalf("unexpected max active token env: %d", cfg.MaxActiveTokensPerUser)
	}
}

func TestLoadTokenAuthCacheMaxEntries(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("API_TOKEN_AUTH_CACHE_MAX_ENTRIES", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected default config to load: %v", err)
	}
	if cfg.APITokenAuthCacheMaxEntries != DefaultAPITokenAuthCacheMaxEntries {
		t.Fatalf("unexpected token auth cache max entries default: %d", cfg.APITokenAuthCacheMaxEntries)
	}

	t.Setenv("API_TOKEN_AUTH_CACHE_MAX_ENTRIES", "128")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("expected env config to load: %v", err)
	}
	if cfg.APITokenAuthCacheMaxEntries != 128 {
		t.Fatalf("unexpected token auth cache max entries env: %d", cfg.APITokenAuthCacheMaxEntries)
	}
}

func TestLoadSyncRunFinishTimeout(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("SYNC_RUN_FINISH_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected default config to load: %v", err)
	}
	if cfg.SyncRunFinishTimeout != DefaultSyncRunFinishTimeout {
		t.Fatalf("unexpected sync finish timeout default: %s", cfg.SyncRunFinishTimeout)
	}

	t.Setenv("SYNC_RUN_FINISH_TIMEOUT", "20s")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("expected env config to load: %v", err)
	}
	if cfg.SyncRunFinishTimeout != 20*time.Second {
		t.Fatalf("unexpected sync finish timeout env: %s", cfg.SyncRunFinishTimeout)
	}
}

func TestValidateTokenLimitSettings(t *testing.T) {
	cfg := validConfig()
	cfg.MaxActiveTokensPerUser = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid MAX_ACTIVE_TOKENS_PER_USER error")
	}
	cfg = validConfig()
	cfg.APITokenAuthCacheMaxEntries = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid API_TOKEN_AUTH_CACHE_MAX_ENTRIES error")
	}
}

func TestValidateAPIRequestLogSettings(t *testing.T) {
	cfg := validConfig()
	cfg.APIRequestLogQueueSize = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid API_REQUEST_LOG_QUEUE_SIZE error")
	}

	cfg = validConfig()
	cfg.APIRequestLogBatchSize = 5001
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid API_REQUEST_LOG_BATCH_SIZE error")
	}

	cfg = validConfig()
	cfg.APIRequestLogFlushInterval = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid API_REQUEST_LOG_FLUSH_INTERVAL error")
	}

	cfg = validConfig()
	cfg.APIRequestLogRetentionDays = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid API_REQUEST_LOG_RETENTION_DAYS error")
	}
}

func TestValidateSyncSettings(t *testing.T) {
	cfg := validConfig()
	cfg.SyncRunFinishTimeout = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid SYNC_RUN_FINISH_TIMEOUT error")
	}
}

func TestValidatePostgresSettings(t *testing.T) {
	cfg := validConfig()
	cfg.DatabasePool.PoolMaxConns = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid DB_POOL_MAX_CONNS error")
	}

	cfg = validConfig()
	cfg.DatabasePool.PoolMinConns = 2
	cfg.DatabasePool.PoolMaxConns = 1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected DB pool min greater than max error")
	}

	cfg = validConfig()
	cfg.SyncDatabasePool.PoolHealthCheckPeriod = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid SYNC_DB_POOL_HEALTH_CHECK_PERIOD error")
	}

	cfg = validConfig()
	cfg.SourceDatabasePool.QueryTimeout = -time.Second
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid SOURCE_DB_QUERY_TIMEOUT error")
	}
}

func TestValidateRedisSettings(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "negative pool size", mutate: func(cfg *Config) { cfg.RedisPoolSize = -1 }},
		{name: "negative min idle", mutate: func(cfg *Config) { cfg.RedisMinIdleConns = -1 }},
		{name: "min idle greater than explicit pool", mutate: func(cfg *Config) {
			cfg.RedisPoolSize = 1
			cfg.RedisMinIdleConns = 2
		}},
		{name: "negative dial timeout", mutate: func(cfg *Config) { cfg.RedisDialTimeout = -time.Second }},
		{name: "negative read timeout", mutate: func(cfg *Config) { cfg.RedisReadTimeout = -time.Second }},
		{name: "negative write timeout", mutate: func(cfg *Config) { cfg.RedisWriteTimeout = -time.Second }},
		{name: "negative pool timeout", mutate: func(cfg *Config) { cfg.RedisPoolTimeout = -time.Second }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validConfig()
			tc.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected invalid Redis config error")
			}
		})
	}
}

func validConfig() Config {
	return Config{
		DatabaseDSN:                       "postgres://example",
		HTTPReadHeaderTimeout:             time.Second,
		HTTPReadTimeout:                   time.Second,
		HTTPWriteTimeout:                  40 * time.Second,
		HTTPIdleTimeout:                   time.Second,
		HTTPMaxHeaderBytes:                1,
		ObservabilityAddr:                 "127.0.0.1:6060",
		DatabasePool:                      defaultDatabasePostgresConfig(),
		SyncDatabasePool:                  defaultSyncPostgresConfig(),
		SourceDatabasePool:                defaultSourcePostgresConfig(),
		LogLevel:                          "info",
		SessionSecret:                     "secret",
		APITokenPepper:                    "pepper",
		EmailCodeMaxAttempts:              1,
		DefaultTokenMinuteLimit:           1,
		DefaultTokenDailyLimit:            1,
		APIPreAuthIPMinuteLimit:           1,
		APIPreAuthIPDailyLimit:            1,
		SessionAuthCacheTTLSeconds:        1,
		SessionLastSeenUpdateIntervalSecs: 1,
		APITokenAuthCacheTTLSeconds:       1,
		APITokenAuthCacheMaxEntries:       DefaultAPITokenAuthCacheMaxEntries,
		MaxActiveTokensPerUser:            1,
		APILastUsedUpdateIntervalSecs:     1,
		APIRequestLogQueueSize:            1,
		APIRequestLogBatchSize:            1,
		APIRequestLogFlushInterval:        time.Second,
		APIRequestLogRetentionDays:        1,
		SyncRunFinishTimeout:              DefaultSyncRunFinishTimeout,
		MailDriver:                        MailDriverSMTP,
		MailSendTimeoutSeconds:            10,
	}
}
