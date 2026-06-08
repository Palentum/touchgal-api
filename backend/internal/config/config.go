package config

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv     string
	HTTPAddr   string
	PublicURL  string
	APIBaseURL string

	DatabaseDSN       string
	SourceDatabaseDSN string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	SessionSecret       string
	SessionCookieName   string
	SessionCookieDomain string
	SessionCookieSecure bool
	SessionTTLHours     int

	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string

	EmailCodeTTLMinutes          int
	EmailCodeResendCooldownSecs  int
	EmailCodeMaxAttempts         int
	APITokenPepper               string
	APITokenPrefix               string
	DefaultTokenMinuteLimit      int
	DefaultTokenDailyLimit       int
	EnableSyncWorker             bool
	SyncIntervalMinutes          int
	SyncFullIntervalHours        int
	SyncIncrementalSafetyMinutes int
	SyncDefaultContentPolicy     string
	TouchGalSiteURL              string
	TouchGalTechDocsURL          string
	APIDocsURL                   string
}

func Load() (Config, error) {
	loadEnvFile(".env")
	loadEnvFile("backend/.env")

	cfg := Config{
		AppEnv:     env("APP_ENV", "development"),
		HTTPAddr:   env("HTTP_ADDR", ":8080"),
		PublicURL:  env("PUBLIC_BASE_URL", "http://localhost:3000"),
		APIBaseURL: env("API_BASE_URL", "http://localhost:8080"),

		DatabaseDSN:       env("DATABASE_DSN", "postgres://touchgal_api:touchgal_api@localhost:5432/touchgal_api?sslmode=disable"),
		SourceDatabaseDSN: env("SOURCE_DATABASE_DSN", ""),

		RedisAddr:     env("REDIS_ADDR", "localhost:6379"),
		RedisPassword: env("REDIS_PASSWORD", ""),
		RedisDB:       envInt("REDIS_DB", 0),

		SessionSecret:       env("SESSION_SECRET", "please-change-this-64-byte-secret"),
		SessionCookieName:   env("SESSION_COOKIE_NAME", "tgal_dev_session"),
		SessionCookieDomain: env("SESSION_COOKIE_DOMAIN", ""),
		SessionCookieSecure: envBool("SESSION_COOKIE_SECURE", false),
		SessionTTLHours:     envInt("SESSION_TTL_HOURS", 720),

		SMTPHost:     env("SMTP_HOST", ""),
		SMTPPort:     envInt("SMTP_PORT", 587),
		SMTPUsername: env("SMTP_USERNAME", ""),
		SMTPPassword: env("SMTP_PASSWORD", ""),
		SMTPFrom:     env("SMTP_FROM", "no-reply@example.com"),
		SMTPFromName: env("SMTP_FROM_NAME", "TouchGal Developer"),

		EmailCodeTTLMinutes:          envInt("EMAIL_CODE_TTL_MINUTES", 10),
		EmailCodeResendCooldownSecs:  envInt("EMAIL_CODE_RESEND_COOLDOWN_SECONDS", 60),
		EmailCodeMaxAttempts:         envInt("EMAIL_CODE_MAX_ATTEMPTS", 5),
		APITokenPepper:               env("API_TOKEN_PEPPER", "please-change-this-long-random-secret"),
		APITokenPrefix:               env("API_TOKEN_PREFIX", "tgal_live"),
		DefaultTokenMinuteLimit:      envInt("DEFAULT_TOKEN_MINUTE_LIMIT", 60),
		DefaultTokenDailyLimit:       envInt("DEFAULT_TOKEN_DAILY_LIMIT", 5000),
		EnableSyncWorker:             envBool("ENABLE_SYNC_WORKER", true),
		SyncIntervalMinutes:          envInt("SYNC_INTERVAL_MINUTES", 30),
		SyncFullIntervalHours:        envInt("SYNC_FULL_INTERVAL_HOURS", 24),
		SyncIncrementalSafetyMinutes: envInt("SYNC_INCREMENTAL_SAFETY_MINUTES", 10),
		SyncDefaultContentPolicy:     env("SYNC_DEFAULT_CONTENT_POLICY", "all"),
		TouchGalSiteURL:              env("TOUCHGAL_SITE_URL", "https://www.touchgal.ink"),
		TouchGalTechDocsURL:          env("TOUCHGAL_TECH_DOCS_URL", "https://github.com/KUN1007/kun-touchgal-next"),
		APIDocsURL:                   env("API_DOCS_URL", "/docs/api"),
	}
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if c.DatabaseDSN == "" {
		return errors.New("DATABASE_DSN is required")
	}
	if c.SessionSecret == "" {
		return errors.New("SESSION_SECRET is required")
	}
	if c.APITokenPepper == "" {
		return errors.New("API_TOKEN_PEPPER is required")
	}
	if c.EmailCodeMaxAttempts <= 0 {
		return errors.New("EMAIL_CODE_MAX_ATTEMPTS must be positive")
	}
	if c.DefaultTokenMinuteLimit <= 0 || c.DefaultTokenDailyLimit <= 0 {
		return errors.New("token limits must be positive")
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
