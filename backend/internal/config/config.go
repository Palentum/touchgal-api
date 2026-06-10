package config

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/touchgal/developer/backend/internal/logging"
)

const (
	MailDriverSMTP   = "smtp"
	MailDriverPostal = "postal"
	MailDriverLog    = "log"
)

type PostgresConfig struct {
	PoolMaxConns                    int
	PoolMinConns                    int
	PoolMinIdleConns                int
	PoolMaxConnLifetime             time.Duration
	PoolMaxConnIdleTime             time.Duration
	PoolHealthCheckPeriod           time.Duration
	StatementTimeout                time.Duration
	IdleInTransactionSessionTimeout time.Duration
	QueryTimeout                    time.Duration
}

type Config struct {
	AppEnv     string
	LogLevel   string
	HTTPAddr   string
	PublicURL  string
	APIBaseURL string

	DatabaseDSN        string
	SourceDatabaseDSN  string
	DatabasePool       PostgresConfig
	SyncDatabasePool   PostgresConfig
	SourceDatabasePool PostgresConfig

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	SessionSecret       string
	SessionCookieName   string
	SessionCookieDomain string
	SessionCookieSecure bool
	SessionTTLHours     int

	MailDriver string

	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string

	PostalAPIURL string
	PostalAPIKey string

	EmailCodeTTLMinutes           int
	EmailCodeResendCooldownSecs   int
	EmailCodeMaxAttempts          int
	APITokenPepper                string
	APITokenPrefix                string
	DefaultTokenMinuteLimit       int
	DefaultTokenDailyLimit        int
	APIPreAuthIPMinuteLimit       int
	APIPreAuthIPDailyLimit        int
	APITokenAuthCacheTTLSeconds   int
	APILastUsedUpdateIntervalSecs int
	APIRequestLogQueueSize        int
	APIRequestLogBatchSize        int
	APIRequestLogFlushInterval    time.Duration
	APIRequestLogRetentionDays    int
	EnableSyncWorker              bool
	SyncIntervalMinutes           int
	SyncFullIntervalHours         int
	SyncIncrementalSafetyMinutes  int
	SyncDefaultContentPolicy      string
	TouchGalSiteURL               string
	TouchGalTechDocsURL           string
	APIDocsURL                    string
}

func defaultDatabasePostgresConfig() PostgresConfig {
	return PostgresConfig{
		PoolMaxConns:                    16,
		PoolMinConns:                    1,
		PoolMinIdleConns:                0,
		PoolMaxConnLifetime:             time.Hour,
		PoolMaxConnIdleTime:             15 * time.Minute,
		PoolHealthCheckPeriod:           time.Minute,
		StatementTimeout:                30 * time.Second,
		IdleInTransactionSessionTimeout: time.Minute,
		QueryTimeout:                    35 * time.Second,
	}
}

func defaultSyncPostgresConfig() PostgresConfig {
	return PostgresConfig{
		PoolMaxConns:                    4,
		PoolMinConns:                    0,
		PoolMinIdleConns:                0,
		PoolMaxConnLifetime:             time.Hour,
		PoolMaxConnIdleTime:             15 * time.Minute,
		PoolHealthCheckPeriod:           time.Minute,
		StatementTimeout:                15 * time.Minute,
		IdleInTransactionSessionTimeout: 5 * time.Minute,
		QueryTimeout:                    16 * time.Minute,
	}
}

func defaultSourcePostgresConfig() PostgresConfig {
	cfg := defaultSyncPostgresConfig()
	cfg.IdleInTransactionSessionTimeout = time.Minute
	return cfg
}

func postgresConfigFromEnv(prefix string, fallback PostgresConfig) PostgresConfig {
	return PostgresConfig{
		PoolMaxConns:                    envInt(prefix+"_POOL_MAX_CONNS", fallback.PoolMaxConns),
		PoolMinConns:                    envInt(prefix+"_POOL_MIN_CONNS", fallback.PoolMinConns),
		PoolMinIdleConns:                envInt(prefix+"_POOL_MIN_IDLE_CONNS", fallback.PoolMinIdleConns),
		PoolMaxConnLifetime:             envDuration(prefix+"_POOL_MAX_CONN_LIFETIME", fallback.PoolMaxConnLifetime),
		PoolMaxConnIdleTime:             envDuration(prefix+"_POOL_MAX_CONN_IDLE_TIME", fallback.PoolMaxConnIdleTime),
		PoolHealthCheckPeriod:           envDuration(prefix+"_POOL_HEALTH_CHECK_PERIOD", fallback.PoolHealthCheckPeriod),
		StatementTimeout:                envDuration(prefix+"_STATEMENT_TIMEOUT", fallback.StatementTimeout),
		IdleInTransactionSessionTimeout: envDuration(prefix+"_IDLE_IN_TRANSACTION_SESSION_TIMEOUT", fallback.IdleInTransactionSessionTimeout),
		QueryTimeout:                    envDuration(prefix+"_QUERY_TIMEOUT", fallback.QueryTimeout),
	}
}

func Load() (Config, error) {
	loadEnvFile(".env")
	loadEnvFile("backend/.env")

	cfg := Config{
		AppEnv:     env("APP_ENV", "development"),
		LogLevel:   strings.ToLower(strings.TrimSpace(env("LOG_LEVEL", "info"))),
		HTTPAddr:   env("HTTP_ADDR", ":8080"),
		PublicURL:  env("PUBLIC_BASE_URL", "http://localhost:3000"),
		APIBaseURL: env("API_BASE_URL", "http://localhost:8080"),

		DatabaseDSN:        env("DATABASE_DSN", "postgres://touchgal_api:touchgal_api@localhost:5432/touchgal_api?sslmode=disable"),
		SourceDatabaseDSN:  env("SOURCE_DATABASE_DSN", ""),
		DatabasePool:       postgresConfigFromEnv("DB", defaultDatabasePostgresConfig()),
		SyncDatabasePool:   postgresConfigFromEnv("SYNC_DB", defaultSyncPostgresConfig()),
		SourceDatabasePool: postgresConfigFromEnv("SOURCE_DB", defaultSourcePostgresConfig()),

		RedisAddr:     env("REDIS_ADDR", "localhost:6379"),
		RedisPassword: env("REDIS_PASSWORD", ""),
		RedisDB:       envInt("REDIS_DB", 0),

		SessionSecret:       env("SESSION_SECRET", "please-change-this-64-byte-secret"),
		SessionCookieName:   env("SESSION_COOKIE_NAME", "tgal_dev_session"),
		SessionCookieDomain: env("SESSION_COOKIE_DOMAIN", ""),
		SessionCookieSecure: envBool("SESSION_COOKIE_SECURE", false),
		SessionTTLHours:     envInt("SESSION_TTL_HOURS", 720),

		MailDriver: strings.ToLower(strings.TrimSpace(env("MAIL_DRIVER", MailDriverSMTP))),

		SMTPHost:     env("SMTP_HOST", ""),
		SMTPPort:     envInt("SMTP_PORT", 587),
		SMTPUsername: env("SMTP_USERNAME", ""),
		SMTPPassword: env("SMTP_PASSWORD", ""),
		SMTPFrom:     env("SMTP_FROM", "no-reply@example.com"),
		SMTPFromName: env("SMTP_FROM_NAME", "TouchGal API"),

		PostalAPIURL: env("POSTAL_API_URL", ""),
		PostalAPIKey: env("POSTAL_API_KEY", ""),

		EmailCodeTTLMinutes:           envInt("EMAIL_CODE_TTL_MINUTES", 10),
		EmailCodeResendCooldownSecs:   envInt("EMAIL_CODE_RESEND_COOLDOWN_SECONDS", 60),
		EmailCodeMaxAttempts:          envInt("EMAIL_CODE_MAX_ATTEMPTS", 5),
		APITokenPepper:                env("API_TOKEN_PEPPER", "please-change-this-long-random-secret"),
		APITokenPrefix:                env("API_TOKEN_PREFIX", "tgal_live"),
		DefaultTokenMinuteLimit:       envInt("DEFAULT_TOKEN_MINUTE_LIMIT", 60),
		DefaultTokenDailyLimit:        envInt("DEFAULT_TOKEN_DAILY_LIMIT", 5000),
		APIPreAuthIPMinuteLimit:       envInt("API_PREAUTH_IP_MINUTE_LIMIT", 600),
		APIPreAuthIPDailyLimit:        envInt("API_PREAUTH_IP_DAILY_LIMIT", 20000),
		APITokenAuthCacheTTLSeconds:   envInt("API_TOKEN_AUTH_CACHE_TTL_SECONDS", 60),
		APILastUsedUpdateIntervalSecs: envInt("API_LAST_USED_UPDATE_INTERVAL_SECONDS", 300),
		APIRequestLogQueueSize:        envInt("API_REQUEST_LOG_QUEUE_SIZE", 16384),
		APIRequestLogBatchSize:        envInt("API_REQUEST_LOG_BATCH_SIZE", 500),
		APIRequestLogFlushInterval:    envDuration("API_REQUEST_LOG_FLUSH_INTERVAL", time.Second),
		APIRequestLogRetentionDays:    envInt("API_REQUEST_LOG_RETENTION_DAYS", 14),
		EnableSyncWorker:              envBool("ENABLE_SYNC_WORKER", false),
		SyncIntervalMinutes:           envInt("SYNC_INTERVAL_MINUTES", 30),
		SyncFullIntervalHours:         envInt("SYNC_FULL_INTERVAL_HOURS", 24),
		SyncIncrementalSafetyMinutes:  envInt("SYNC_INCREMENTAL_SAFETY_MINUTES", 10),
		SyncDefaultContentPolicy:      env("SYNC_DEFAULT_CONTENT_POLICY", "all"),
		TouchGalSiteURL:               env("TOUCHGAL_SITE_URL", "https://www.touchgal.ink"),
		TouchGalTechDocsURL:           env("TOUCHGAL_TECH_DOCS_URL", "https://github.com/KUN1007/kun-touchgal-next"),
		APIDocsURL:                    env("API_DOCS_URL", "/docs/api"),
	}
	return cfg, cfg.Validate()
}

func validatePostgresConfig(prefix string, cfg PostgresConfig) error {
	if cfg.PoolMaxConns <= 0 {
		return fmt.Errorf("%s_POOL_MAX_CONNS must be positive", prefix)
	}
	if cfg.PoolMinConns < 0 {
		return fmt.Errorf("%s_POOL_MIN_CONNS must be zero or positive", prefix)
	}
	if cfg.PoolMinIdleConns < 0 {
		return fmt.Errorf("%s_POOL_MIN_IDLE_CONNS must be zero or positive", prefix)
	}
	if cfg.PoolMinConns > cfg.PoolMaxConns {
		return fmt.Errorf("%s_POOL_MIN_CONNS must be less than or equal to %s_POOL_MAX_CONNS", prefix, prefix)
	}
	if cfg.PoolMinIdleConns > cfg.PoolMaxConns {
		return fmt.Errorf("%s_POOL_MIN_IDLE_CONNS must be less than or equal to %s_POOL_MAX_CONNS", prefix, prefix)
	}
	if cfg.PoolMaxConnLifetime <= 0 {
		return fmt.Errorf("%s_POOL_MAX_CONN_LIFETIME must be positive", prefix)
	}
	if cfg.PoolMaxConnIdleTime <= 0 {
		return fmt.Errorf("%s_POOL_MAX_CONN_IDLE_TIME must be positive", prefix)
	}
	if cfg.PoolHealthCheckPeriod <= 0 {
		return fmt.Errorf("%s_POOL_HEALTH_CHECK_PERIOD must be positive", prefix)
	}
	if cfg.StatementTimeout < 0 {
		return fmt.Errorf("%s_STATEMENT_TIMEOUT must be zero or positive", prefix)
	}
	if cfg.IdleInTransactionSessionTimeout < 0 {
		return fmt.Errorf("%s_IDLE_IN_TRANSACTION_SESSION_TIMEOUT must be zero or positive", prefix)
	}
	if cfg.QueryTimeout < 0 {
		return fmt.Errorf("%s_QUERY_TIMEOUT must be zero or positive", prefix)
	}
	return nil
}

func (c Config) Validate() error {
	if c.DatabaseDSN == "" {
		return errors.New("DATABASE_DSN is required")
	}
	if err := validatePostgresConfig("DB", c.DatabasePool); err != nil {
		return err
	}
	if err := validatePostgresConfig("SYNC_DB", c.SyncDatabasePool); err != nil {
		return err
	}
	if err := validatePostgresConfig("SOURCE_DB", c.SourceDatabasePool); err != nil {
		return err
	}
	if c.SessionSecret == "" {
		return errors.New("SESSION_SECRET is required")
	}
	if c.APITokenPepper == "" {
		return errors.New("API_TOKEN_PEPPER is required")
	}
	if _, err := logging.ParseLevel(c.LogLevel); err != nil {
		return err
	}
	switch c.MailDriver {
	case MailDriverSMTP, MailDriverPostal, MailDriverLog:
	default:
		return errors.New("MAIL_DRIVER must be one of: smtp, postal, log")
	}
	if c.MailDriver == MailDriverPostal {
		if c.PostalAPIURL == "" {
			return errors.New("POSTAL_API_URL is required when MAIL_DRIVER=postal")
		}
		postalURL, err := url.Parse(c.PostalAPIURL)
		if err != nil || postalURL.Scheme == "" || postalURL.Host == "" {
			return errors.New("POSTAL_API_URL must be a valid absolute URL when MAIL_DRIVER=postal")
		}
		if postalURL.Scheme != "https" {
			return errors.New("POSTAL_API_URL must use https when MAIL_DRIVER=postal")
		}
		if c.PostalAPIKey == "" {
			return errors.New("POSTAL_API_KEY is required when MAIL_DRIVER=postal")
		}
	}
	if c.EmailCodeMaxAttempts <= 0 {
		return errors.New("EMAIL_CODE_MAX_ATTEMPTS must be positive")
	}
	if c.DefaultTokenMinuteLimit <= 0 || c.DefaultTokenDailyLimit <= 0 {
		return errors.New("token limits must be positive")
	}
	if c.APIPreAuthIPMinuteLimit <= 0 || c.APIPreAuthIPDailyLimit <= 0 {
		return errors.New("pre-auth IP limits must be positive")
	}
	if c.APITokenAuthCacheTTLSeconds <= 0 {
		return errors.New("API_TOKEN_AUTH_CACHE_TTL_SECONDS must be positive")
	}
	if c.APILastUsedUpdateIntervalSecs <= 0 {
		return errors.New("API_LAST_USED_UPDATE_INTERVAL_SECONDS must be positive")
	}
	if c.APIRequestLogQueueSize <= 0 {
		return errors.New("API_REQUEST_LOG_QUEUE_SIZE must be positive")
	}
	if c.APIRequestLogBatchSize <= 0 || c.APIRequestLogBatchSize > 5000 {
		return errors.New("API_REQUEST_LOG_BATCH_SIZE must be between 1 and 5000")
	}
	if c.APIRequestLogFlushInterval <= 0 {
		return errors.New("API_REQUEST_LOG_FLUSH_INTERVAL must be positive")
	}
	if c.APIRequestLogRetentionDays <= 0 {
		return errors.New("API_REQUEST_LOG_RETENTION_DAYS must be positive")
	}
	return nil
}

func (c Config) SessionTTL() time.Duration {
	return time.Duration(c.SessionTTLHours) * time.Hour
}

func (c Config) EmailCodeTTL() time.Duration {
	return time.Duration(c.EmailCodeTTLMinutes) * time.Minute
}

func (c Config) EmailCooldown() time.Duration {
	return time.Duration(c.EmailCodeResendCooldownSecs) * time.Second
}

func (c Config) APITokenAuthCacheTTL() time.Duration {
	return time.Duration(c.APITokenAuthCacheTTLSeconds) * time.Second
}

func (c Config) APILastUsedUpdateInterval() time.Duration {
	return time.Duration(c.APILastUsedUpdateIntervalSecs) * time.Second
}

func (c Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func env(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func envInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return parsed
}
func envDuration(key string, fallback time.Duration) time.Duration {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if val == "" {
		return fallback
	}
	return val == "true" || val == "1" || val == "yes"
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		_ = os.Setenv(key, value)
	}
}
