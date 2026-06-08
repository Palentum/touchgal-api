package email

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/config"
)

type Mailer interface {
	SendVerificationCode(to, purpose, code string, ttlMinutes int) error
}

func NewMailer(cfg config.Config, logger zerolog.Logger) Mailer {
	switch cfg.MailDriver {
	case config.MailDriverPostal:
		return NewPostalMailer(cfg)
	case config.MailDriverLog:
		return LoggingMailer{Logger: logger}
	default:
		if cfg.SMTPHost == "" {
			return LoggingMailer{Logger: logger}
		}
		return NewSMTPMailer(cfg)
	}
}

type SMTPMailer struct {
	cfg config.Config
}

func NewSMTPMailer(cfg config.Config) *SMTPMailer { return &SMTPMailer{cfg: cfg} }

func formatFrom(address, name string) string {
	if name == "" {
		return address
	}
	return fmt.Sprintf("%s <%s>", name, address)
}

func (m *SMTPMailer) SendVerificationCode(to, purpose, code string, ttlMinutes int) error {
	if m.cfg.SMTPHost == "" || m.cfg.SMTPFrom == "" {
		return nil
	}
	subject := VerificationSubject(purpose)
	body := VerificationBody(code, ttlMinutes)
	fromLabel := formatFrom(m.cfg.SMTPFrom, m.cfg.SMTPFromName)
	msg := strings.Join([]string{
		"From: " + fromLabel,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", m.cfg.SMTPHost, m.cfg.SMTPPort)
	var auth smtp.Auth
	if m.cfg.SMTPUsername != "" || m.cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", m.cfg.SMTPUsername, m.cfg.SMTPPassword, m.cfg.SMTPHost)
	}
	return smtp.SendMail(addr, auth, m.cfg.SMTPFrom, []string{to}, []byte(msg))
}

type LoggingMailer struct{ Logger zerolog.Logger }

func (m LoggingMailer) SendVerificationCode(to, purpose, code string, ttlMinutes int) error {
	m.Logger.Info().Str("email", to).Str("purpose", purpose).Int("ttl_minutes", ttlMinutes).Str("code", code).Msg("development verification code")
	return nil
}
