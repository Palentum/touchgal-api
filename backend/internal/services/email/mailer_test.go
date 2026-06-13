package email

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/config"
)

func TestNewMailerFallsBackToLogOnlyInDevelopment(t *testing.T) {
	devMailer := NewMailer(config.Config{AppEnv: "development", MailDriver: config.MailDriverSMTP}, zerolog.Nop())
	logMailer, ok := devMailer.(LoggingMailer)
	if !ok {
		t.Fatalf("expected development SMTP without host to use LoggingMailer, got %T", devMailer)
	}
	if !logMailer.RevealCode {
		t.Fatal("expected development log mailer to reveal verification codes")
	}

	prodMailer := NewMailer(config.Config{AppEnv: "production", MailDriver: config.MailDriverSMTP}, zerolog.Nop())
	if _, ok := prodMailer.(*SMTPMailer); !ok {
		t.Fatalf("expected production SMTP without host to stay on SMTPMailer, got %T", prodMailer)
	}
}

func TestLoggingMailerRedactsCodeOutsideDevelopment(t *testing.T) {
	const code = "123456"
	var buf bytes.Buffer
	mailer := LoggingMailer{Logger: zerolog.New(&buf)}

	if err := mailer.SendVerificationCode("user@example.com", "login", code, 10); err != nil {
		t.Fatalf("send verification code: %v", err)
	}
	if strings.Contains(buf.String(), code) {
		t.Fatalf("log output contains verification code: %s", buf.String())
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse log entry: %v", err)
	}
	if _, ok := entry["code"]; ok {
		t.Fatalf("log output contains code field: %s", buf.String())
	}
}

func TestLoggingMailerRevealsCodeWhenEnabled(t *testing.T) {
	const code = "123456"
	var buf bytes.Buffer
	mailer := LoggingMailer{Logger: zerolog.New(&buf), RevealCode: true}

	if err := mailer.SendVerificationCode("user@example.com", "login", code, 10); err != nil {
		t.Fatalf("send verification code: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse log entry: %v", err)
	}
	if entry["code"] != code {
		t.Fatalf("code field = %v, want %s", entry["code"], code)
	}
}

func TestSMTPMailerRejectsMissingConfig(t *testing.T) {
	missingHost := NewSMTPMailer(config.Config{SMTPFrom: "no-reply@example.com"})
	if err := missingHost.SendVerificationCode("user@example.com", "login", "123456", 10); err == nil || !strings.Contains(err.Error(), "SMTP_HOST") {
		t.Fatalf("expected SMTP_HOST error, got %v", err)
	}

	missingFrom := NewSMTPMailer(config.Config{SMTPHost: "smtp.example.com"})
	if err := missingFrom.SendVerificationCode("user@example.com", "login", "123456", 10); err == nil || !strings.Contains(err.Error(), "SMTP_FROM") {
		t.Fatalf("expected SMTP_FROM error, got %v", err)
	}
}
