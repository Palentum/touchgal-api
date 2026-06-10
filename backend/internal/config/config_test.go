package config

import "testing"

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

func validConfig() Config {
	return Config{
		DatabaseDSN:                   "postgres://example",
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
		MailDriver:                    MailDriverSMTP,
	}
}
