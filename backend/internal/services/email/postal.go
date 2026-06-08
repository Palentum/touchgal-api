package email

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/touchgal/developer/backend/internal/config"
)

const postalSendPath = "/api/v1/send/message"

type PostalMailer struct {
	cfg    config.Config
	client *http.Client
}

func NewPostalMailer(cfg config.Config) *PostalMailer {
	return &PostalMailer{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type postalSendRequest struct {
	From      string   `json:"from"`
	To        []string `json:"to"`
	Subject   string   `json:"subject"`
	PlainBody string   `json:"plain_body"`
}

type postalAPIResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

func (m *PostalMailer) SendVerificationCode(to, purpose, code string, ttlMinutes int) error {
	if m.cfg.PostalAPIURL == "" {
		return errors.New("POSTAL_API_URL is required")
	}
	if m.cfg.PostalAPIKey == "" {
		return errors.New("POSTAL_API_KEY is required")
	}
	if m.cfg.SMTPFrom == "" {
		return errors.New("SMTP_FROM is required")
	}

	payload := postalSendRequest{
		From:      formatFrom(m.cfg.SMTPFrom, m.cfg.SMTPFromName),
		To:        []string{to},
		Subject:   VerificationSubject(purpose),
		PlainBody: VerificationBody(code, ttlMinutes),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, postalSendURL(m.cfg.PostalAPIURL), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Server-API-Key", m.cfg.PostalAPIKey)

	if m.client == nil {
		return errors.New("postal HTTP client is not configured")
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	return handlePostalResponse(resp)
}

func postalSendURL(baseURL string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return strings.TrimRight(baseURL, "/") + postalSendPath
	}

	trimmedPath := strings.TrimRight(parsed.Path, "/")
	switch {
	case trimmedPath == "":
		parsed.Path = postalSendPath
	case trimmedPath == postalSendPath || strings.HasPrefix(postalSendPath, trimmedPath+"/"):
		parsed.Path = postalSendPath
	default:
		parsed.Path = trimmedPath + postalSendPath
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func handlePostalResponse(resp *http.Response) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return err
	}
	bodyText := strings.TrimSpace(string(body))

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("postal send failed: HTTP %d: %s", resp.StatusCode, bodyText)
	}
	if bodyText == "" {
		return errors.New("postal send failed: empty response")
	}

	var apiResp postalAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("decode postal response: %w", err)
	}
	if apiResp.Status == "success" {
		return nil
	}
	if apiResp.Status == "" {
		return errors.New("postal send failed: missing status")
	}
	if len(apiResp.Data) == 0 {
		return fmt.Errorf("postal send failed: status %q", apiResp.Status)
	}
	return fmt.Errorf("postal send failed: status %q: %s", apiResp.Status, strings.TrimSpace(string(apiResp.Data)))
}
