package turnstile

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/touchgal/developer/backend/internal/model"
)

const siteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

const maxResponseTokenLength = 2048

// Verifier validates a Cloudflare Turnstile response token before mail is sent.
type Verifier interface {
	Verify(ctx context.Context, token, remoteIP string) error
}

// HTTPClient is the subset of http.Client used by HTTPVerifier.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// NoopVerifier allows local and test environments to skip Turnstile.
type NoopVerifier struct{}

func (NoopVerifier) Verify(context.Context, string, string) error {
	return nil
}

// HTTPVerifier verifies Turnstile tokens through Cloudflare siteverify.
type HTTPVerifier struct {
	secret   string
	endpoint string
	client   HTTPClient
}

// New returns a no-op verifier when secret is empty, otherwise an HTTP verifier.
func New(secret string) Verifier {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return NoopVerifier{}
	}
	return &HTTPVerifier{
		secret:   secret,
		endpoint: siteverifyURL,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

// NewHTTPVerifier builds an HTTP verifier with test-overridable dependencies.
func NewHTTPVerifier(secret string, client HTTPClient, endpoint string) *HTTPVerifier {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	if endpoint == "" {
		endpoint = siteverifyURL
	}
	return &HTTPVerifier{
		secret:   strings.TrimSpace(secret),
		endpoint: endpoint,
		client:   client,
	}
}

func (v *HTTPVerifier) Verify(ctx context.Context, token, remoteIP string) error {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > maxResponseTokenLength {
		return model.ErrTurnstileFailed
	}

	form := url.Values{}
	form.Set("secret", v.secret)
	form.Set("response", token)
	if remoteIP = strings.TrimSpace(remoteIP); remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return model.ErrTurnstileUnavailable
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		return model.ErrTurnstileUnavailable
	}
	if resp.Body == nil {
		return model.ErrTurnstileUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return model.ErrTurnstileUnavailable
	}

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<10)).Decode(&result); err != nil {
		return model.ErrTurnstileUnavailable
	}
	if !result.Success {
		return model.ErrTurnstileFailed
	}
	return nil
}
