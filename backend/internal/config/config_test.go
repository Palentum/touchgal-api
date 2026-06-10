package config

import (
	"testing"
	"time"
)

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
		APILastUsedUpdateIntervalSecs: 1,
		APIRequestLogQueueSize:        1,
		APIRequestLogBatchSize:        1,
		APIRequestLogFlushInterval:    time.Second,
		APIRequestLogRetentionDays:    1,
		MailDriver:                    MailDriverSMTP,
	}
}
