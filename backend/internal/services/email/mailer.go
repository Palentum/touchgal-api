package email

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

const (
	verificationMIMEBoundary = "touchgal-api-verification-boundary"
	mimeBoundaryRandomBytes  = 16
)

type Mailer interface {
	SendVerificationCode(to, purpose, code string, ttlMinutes int) error
	SendApplicationSubmitted(to []string, app model.Application, reviewURL string) error
	SendApplicationApproved(to string, app model.Application, dashboardURL string) error
}

func NewMailer(cfg config.Config, logger zerolog.Logger) Mailer {
	revealLogCodes := cfg.IsDevelopment()
	switch cfg.MailDriver {
	case config.MailDriverPostal:
		return NewPostalMailer(cfg)
	case config.MailDriverLog:
		return LoggingMailer{Logger: logger, RevealCode: revealLogCodes}
	default:
		if cfg.SMTPHost == "" && revealLogCodes {
			return LoggingMailer{Logger: logger, RevealCode: true}
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
	subject := VerificationSubject(purpose)
	plainBody := VerificationBody(code, ttlMinutes)
	htmlBody := VerificationHTMLBody(purpose, code, ttlMinutes)
	return m.sendMultipartWithBoundary([]string{to}, subject, plainBody, htmlBody, verificationMIMEBoundary)
}

func (m *SMTPMailer) sendMultipart(to []string, subject, plainBody, htmlBody string) error {
	boundary, err := randomMIMEBoundary()
	if err != nil {
		return err
	}
	return m.sendMultipartWithBoundary(to, subject, plainBody, htmlBody, boundary)
}

func randomMIMEBoundary() (string, error) {
	var entropy [mimeBoundaryRandomBytes]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return "", err
	}
	return "touchgal-api-boundary-" + hex.EncodeToString(entropy[:]), nil
}

func (m *SMTPMailer) sendMultipartWithBoundary(to []string, subject, plainBody, htmlBody, boundary string) error {
	if m.cfg.SMTPHost == "" {
		return errors.New("SMTP_HOST is required")
	}
	if m.cfg.SMTPFrom == "" {
		return errors.New("SMTP_FROM is required")
	}
	if len(to) == 0 {
		return errors.New("email recipient is required")
	}
	fromLabel := formatFrom(m.cfg.SMTPFrom, m.cfg.SMTPFromName)
	msg := strings.Join([]string{
		"From: " + fromLabel,
		"To: " + strings.Join(to, ", "),
		"Subject: " + subject,
		"MIME-Version: 1.0",
		`Content-Type: multipart/alternative; boundary="` + boundary + `"`,
		"",
		"--" + boundary,
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		plainBody,
		"--" + boundary,
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		htmlBody,
		"--" + boundary + "--",
		"",
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", m.cfg.SMTPHost, m.cfg.SMTPPort)
	var auth smtp.Auth
	if m.cfg.SMTPUsername != "" || m.cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", m.cfg.SMTPUsername, m.cfg.SMTPPassword, m.cfg.SMTPHost)
	}
	return m.sendMail(addr, auth, to, []byte(msg))
}

func (m *SMTPMailer) SendApplicationSubmitted(to []string, app model.Application, reviewURL string) error {
	return m.sendMultipart(to, ApplicationSubmittedSubject(), ApplicationSubmittedBody(app, reviewURL), ApplicationSubmittedHTMLBody(app, reviewURL))
}

func (m *SMTPMailer) SendApplicationApproved(to string, app model.Application, dashboardURL string) error {
	return m.sendMultipart([]string{to}, ApplicationApprovedSubject(), ApplicationApprovedBody(app, dashboardURL), ApplicationApprovedHTMLBody(app, dashboardURL))
}

func (m *SMTPMailer) sendMail(addr string, auth smtp.Auth, to []string, msg []byte) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	timeout := m.cfg.MailSendTimeout()
	conn, err := (&net.Dialer{Timeout: timeout}).Dial("tcp", addr)
	if err != nil {
		return err
	}
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		_ = conn.Close()
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return err
		}
	}
	if auth != nil {
		if ok, _ := c.Extension("AUTH"); !ok {
			return errors.New("smtp: server doesn't support AUTH")
		}
		if err := c.Auth(auth); err != nil {
			return err
		}
	}
	if err := c.Mail(m.cfg.SMTPFrom); err != nil {
		return err
	}
	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

type LoggingMailer struct {
	Logger     zerolog.Logger
	RevealCode bool
}

func (m LoggingMailer) SendVerificationCode(to, purpose, code string, ttlMinutes int) error {
	event := m.Logger.Info().
		Str("email", to).
		Str("purpose", purpose).
		Int("ttl_minutes", ttlMinutes)
	if m.RevealCode {
		event.Str("code", code).Msg("development verification code")
		return nil
	}
	event.Msg("verification code generated by log mailer; code redacted outside development")
	return nil
}

func (m LoggingMailer) SendApplicationSubmitted(to []string, app model.Application, reviewURL string) error {
	m.Logger.Info().
		Int("recipient_count", len(to)).
		Str("application_id", app.ID.String()).
		Msg("application submitted admin notification generated by log mailer")
	return nil
}

func (m LoggingMailer) SendApplicationApproved(to string, app model.Application, dashboardURL string) error {
	m.Logger.Info().
		Str("email", to).
		Str("application_id", app.ID.String()).
		Str("dashboard_url", dashboardURL).
		Msg("application approved user notification generated by log mailer")
	return nil
}
