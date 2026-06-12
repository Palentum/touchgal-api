package turnstile

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/touchgal/developer/backend/internal/model"
)

func TestHTTPVerifierPostsSiteverifyForm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("Content-Type = %q, want application/x-www-form-urlencoded", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("secret"); got != "secret-key" {
			t.Fatalf("secret = %q, want secret-key", got)
		}
		if got := r.Form.Get("response"); got != "response-token" {
			t.Fatalf("response = %q, want response-token", got)
		}
		if got := r.Form.Get("remoteip"); got != "203.0.113.10" {
			t.Fatalf("remoteip = %q, want 203.0.113.10", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	verifier := NewHTTPVerifier("secret-key", server.Client(), server.URL)
	if err := verifier.Verify(context.Background(), "response-token", "203.0.113.10"); err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
}

func TestHTTPVerifierRejectsMissingOrLongTokenWithoutRequest(t *testing.T) {
	verifier := NewHTTPVerifier("secret-key", fatalHTTPClient{t: t}, "https://example.invalid")

	for _, token := range []string{"", strings.Repeat("a", maxResponseTokenLength+1)} {
		err := verifier.Verify(context.Background(), token, "")
		if !errors.Is(err, model.ErrTurnstileFailed) {
			t.Fatalf("Verify(%q) error = %v, want ErrTurnstileFailed", token, err)
		}
	}
}

func TestHTTPVerifierFailedOutcome(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"error-codes":["timeout-or-duplicate"]}`))
	}))
	defer server.Close()

	verifier := NewHTTPVerifier("secret-key", server.Client(), server.URL)
	err := verifier.Verify(context.Background(), "response-token", "")
	if !errors.Is(err, model.ErrTurnstileFailed) {
		t.Fatalf("Verify error = %v, want ErrTurnstileFailed", err)
	}
}

func TestHTTPVerifierUnavailableOnBadStatusOrMalformedJSON(t *testing.T) {
	tests := map[string]http.HandlerFunc{
		"bad status": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "upstream error", http.StatusInternalServerError)
		},
		"malformed json": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`not-json`))
		},
	}

	for name, handler := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(handler)
			defer server.Close()

			verifier := NewHTTPVerifier("secret-key", server.Client(), server.URL)
			err := verifier.Verify(context.Background(), "response-token", "")
			if !errors.Is(err, model.ErrTurnstileUnavailable) {
				t.Fatalf("Verify error = %v, want ErrTurnstileUnavailable", err)
			}
		})
	}
}

type fatalHTTPClient struct {
	t *testing.T
}

func (c fatalHTTPClient) Do(*http.Request) (*http.Response, error) {
	c.t.Fatal("unexpected HTTP request")
	return nil, nil
}
