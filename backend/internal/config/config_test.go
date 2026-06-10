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

func TestValidateRejectsUnknownLogLevel(t *testing.T) {
	cfg := validConfig()
	cfg.LogLevel = "verbose"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid LOG_LEVEL error")
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

func TestValidateTokenLimitSettings(t *testing.T) {
	cfg := validConfig()
	cfg.MaxActiveTokensPerUser = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid MAX_ACTIVE_TOKENS_PER_USER error")
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

func validConfig() Config {
	return Config{
		DatabaseDSN:                   "postgres://example",
		HTTPReadHeaderTimeout:         time.Second,
		HTTPReadTimeout:               time.Second,
		HTTPWriteTimeout:              40 * time.Second,
		HTTPIdleTimeout:               time.Second,
		HTTPMaxHeaderBytes:            1,
		DatabasePool:                  defaultDatabasePostgresConfig(),
		SyncDatabasePool:              defaultSyncPostgresConfig(),
		SourceDatabasePool:            defaultSourcePostgresConfig(),
		LogLevel:                      "info",
		SessionSecret:                 "secret",
		APITokenPepper:                "pepper",
		EmailCodeMaxAttempts:          1,
		DefaultTokenMinuteLimit:       1,
		DefaultTokenDailyLimit:        1,
		APIPreAuthIPMinuteLimit:       1,
		APIPreAuthIPDailyLimit:        1,
		APITokenAuthCacheTTLSeconds:   1,
		MaxActiveTokensPerUser:        1,
		APILastUsedUpdateIntervalSecs: 1,
		APIRequestLogQueueSize:        1,
		APIRequestLogBatchSize:        1,
		APIRequestLogFlushInterval:    time.Second,
		APIRequestLogRetentionDays:    1,
		MailDriver:                    MailDriverSMTP,
	}
}
