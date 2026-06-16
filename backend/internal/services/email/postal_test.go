package email

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

func TestNewMailerSelectsPostalDriver(t *testing.T) {
	mailer := NewMailer(config.Config{MailDriver: config.MailDriverPostal}, zerolog.Nop())
	if _, ok := mailer.(*PostalMailer); !ok {
		t.Fatalf("expected PostalMailer, got %T", mailer)
	}
}

func TestPostalMailerSendsVerificationCode(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != postalSendPath {
			t.Errorf("expected %s, got %s", postalSendPath, r.URL.Path)
		}
		if r.Header.Get("X-Server-API-Key") != "secret" {
			t.Errorf("missing Postal API key header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected JSON content type, got %q", r.Header.Get("Content-Type"))
		}

		var payload postalSendRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if payload.From != "TouchGal API <no-reply@example.com>" {
			t.Errorf("unexpected from: %q", payload.From)
		}
		if len(payload.To) != 1 || payload.To[0] != "user@example.com" {
			t.Errorf("unexpected to: %#v", payload.To)
		}
		if payload.Subject != "TouchGal API 登录验证码" {
			t.Errorf("unexpected subject: %q", payload.Subject)
		}
		if !strings.Contains(payload.PlainBody, "123456") || !strings.Contains(payload.PlainBody, "10 分钟") {
			t.Errorf("verification body missing code or ttl: %q", payload.PlainBody)
		}
		if !strings.Contains(payload.HTMLBody, "123456") || !strings.Contains(payload.HTMLBody, "#cc785c") || !strings.Contains(payload.HTMLBody, "#181715") {
			t.Errorf("verification HTML body missing code or design tokens: %q", payload.HTMLBody)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"message_id":"msg"}}`))
	}))
	defer server.Close()

	mailer := NewPostalMailer(config.Config{
		PostalAPIURL: server.URL,
		PostalAPIKey: "secret",
		SMTPFrom:     "no-reply@example.com",
		SMTPFromName: "TouchGal API",
	})
	if err := mailer.SendVerificationCode("user@example.com", "login", "123456", 10); err != nil {
		t.Fatalf("send verification code: %v", err)
	}
	if !called {
		t.Fatal("Postal API was not called")
	}
}

func TestPostalMailerSendsApplicationSubmittedNotification(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		var payload postalSendRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if len(payload.To) != 2 || payload.To[0] != "admin@example.com" || payload.To[1] != "ops@example.com" {
			t.Errorf("unexpected to: %#v", payload.To)
		}
		if payload.Subject != "TouchGal API 新应用申请待审核" {
			t.Errorf("unexpected subject: %q", payload.Subject)
		}
		if !strings.Contains(payload.PlainBody, "https://portal.example.com/admin/applications") {
			t.Errorf("plain body missing review URL: %q", payload.PlainBody)
		}
		if !strings.Contains(payload.HTMLBody, "https://portal.example.com/admin/applications") {
			t.Errorf("HTML body missing review URL: %q", payload.HTMLBody)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"message_id":"msg"}}`))
	}))
	defer server.Close()

	mailer := NewPostalMailer(config.Config{
		PostalAPIURL: server.URL,
		PostalAPIKey: "secret",
		SMTPFrom:     "no-reply@example.com",
	})
	err := mailer.SendApplicationSubmitted(
		[]string{"admin@example.com", "ops@example.com"},
		model.Application{ApplicantName: "Kun", ProjectName: "Docs Bot", ProjectURL: "https://example.com", ExpectedDailyRequests: 100, UsageScenario: "展示条目信息"},
		"https://portal.example.com/admin/applications",
	)
	if err != nil {
		t.Fatalf("send application submitted notification: %v", err)
	}
	if !called {
		t.Fatal("Postal API was not called")
	}
}

func TestPostalMailerSendsApplicationApprovedNotification(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		var payload postalSendRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if len(payload.To) != 1 || payload.To[0] != "dev@example.com" {
			t.Errorf("unexpected to: %#v", payload.To)
		}
		if payload.Subject != "TouchGal API 应用申请已通过" {
			t.Errorf("unexpected subject: %q", payload.Subject)
		}
		if !strings.Contains(payload.PlainBody, "https://portal.example.com/dashboard/tokens") {
			t.Errorf("plain body missing dashboard URL: %q", payload.PlainBody)
		}
		if !strings.Contains(payload.HTMLBody, "https://portal.example.com/dashboard/tokens") {
			t.Errorf("HTML body missing dashboard URL: %q", payload.HTMLBody)
		}
		if !strings.Contains(payload.PlainBody, "10") || !strings.Contains(payload.PlainBody, "100") {
			t.Errorf("plain body missing limits: %q", payload.PlainBody)
		}
		if !strings.Contains(payload.HTMLBody, "10") || !strings.Contains(payload.HTMLBody, "100") {
			t.Errorf("HTML body missing limits: %q", payload.HTMLBody)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"message_id":"msg"}}`))
	}))
	defer server.Close()

	mailer := NewPostalMailer(config.Config{
		PostalAPIURL: server.URL,
		PostalAPIKey: "secret",
		SMTPFrom:     "no-reply@example.com",
	})
	err := mailer.SendApplicationApproved(
		"dev@example.com",
		model.Application{ApplicantName: "Kun", ProjectName: "Docs Bot", ProjectURL: "https://example.com", DefaultMinuteLimit: 10, DefaultDailyLimit: 100},
		"https://portal.example.com/dashboard/tokens",
	)
	if err != nil {
		t.Fatalf("send application approved notification: %v", err)
	}
	if !called {
		t.Fatal("Postal API was not called")
	}
}

func TestPostalMailerReportsAPIErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"error","data":{"message":"bad key"}}`))
	}))
	defer server.Close()

	mailer := NewPostalMailer(config.Config{
		PostalAPIURL: server.URL,
		PostalAPIKey: "secret",
		SMTPFrom:     "no-reply@example.com",
	})
	err := mailer.SendVerificationCode("user@example.com", "login", "123456", 10)
	if err == nil {
		t.Fatal("expected Postal API error")
	}
	if !strings.Contains(err.Error(), `status "error"`) || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostalMailerReportsMissingStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"message_id":"msg"}}`))
	}))
	defer server.Close()

	mailer := NewPostalMailer(config.Config{
		PostalAPIURL: server.URL,
		PostalAPIKey: "secret",
		SMTPFrom:     "no-reply@example.com",
	})
	err := mailer.SendVerificationCode("user@example.com", "login", "123456", 10)
	if err == nil {
		t.Fatal("expected missing status error")
	}
	if !strings.Contains(err.Error(), "missing status") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostalSendURLAcceptsRootOrEndpoint(t *testing.T) {
	tests := map[string]string{
		"https://postal.example.com":                          "https://postal.example.com/api/v1/send/message",
		"https://postal.example.com/":                         "https://postal.example.com/api/v1/send/message",
		"https://postal.example.com/api/v1/send":              "https://postal.example.com/api/v1/send/message",
		"https://postal.example.com/api/v1/send/message":      "https://postal.example.com/api/v1/send/message",
		"https://postal.example.com/postal":                   "https://postal.example.com/postal/api/v1/send/message",
		"https://postal.example.com/api/v1/send/message?x=1":  "https://postal.example.com/api/v1/send/message",
		"https://postal.example.com/api/v1/send/message#frag": "https://postal.example.com/api/v1/send/message",
	}

	for input, expected := range tests {
		if got := postalSendURL(input); got != expected {
			t.Fatalf("postalSendURL(%q) = %q, want %q", input, got, expected)
		}
	}
}
