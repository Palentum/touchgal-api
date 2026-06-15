package email

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
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

func TestVerificationHTMLBodyUsesDesignTokens(t *testing.T) {
	body := VerificationHTMLBody("register", "<123456>", 10)

	for _, want := range []string{
		"#faf9f5",
		"#cc785c",
		"#181715",
		"完成 TouchGal API 开发者门户注册",
		"10 分钟",
		"&lt;123456&gt;",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("verification HTML missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "<123456>") {
		t.Fatalf("verification HTML contains unescaped code:\n%s", body)
	}
}

func TestApplicationSubmittedHTMLBodyEscapesFields(t *testing.T) {
	body := ApplicationSubmittedHTMLBody(model.Application{
		ApplicantName: "<Kun>",
		ProjectName:   "<Bot>",
		ProjectURL:    "https://example.com/?q=<x>",
		UsageScenario: "line <one>",
	}, "https://portal.example.com/admin/applications")

	for _, want := range []string{
		"&lt;Kun&gt;",
		"&lt;Bot&gt;",
		"&lt;one&gt;",
		"#faf9f5",
		"#cc785c",
		"#181715",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("application submitted HTML missing %q:\n%s", want, body)
		}
	}
	for _, notWant := range []string{"<Kun>", "<Bot>"} {
		if strings.Contains(body, notWant) {
			t.Fatalf("application submitted HTML contains unescaped %q:\n%s", notWant, body)
		}
	}
}
