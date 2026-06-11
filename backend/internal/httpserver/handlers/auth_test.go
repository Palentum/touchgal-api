package handlers

import (
	"testing"

	"github.com/touchgal/developer/backend/internal/config"
)

func TestCodeStartResponseUsesConfiguredTiming(t *testing.T) {
	h := &AuthHandler{cfg: config.Config{EmailCodeTTLMinutes: 7, EmailCodeResendCooldownSecs: 45}}

	response := h.codeStartResponse()

	if !response.Sent {
		t.Fatal("expected sent flag")
	}
	if response.ExpiresInSeconds != 7*60 {
		t.Fatalf("expected 420 seconds ttl, got %d", response.ExpiresInSeconds)
	}
	if response.ResendCooldownSeconds != 45 {
		t.Fatalf("expected 45 seconds cooldown, got %d", response.ResendCooldownSeconds)
	}
}
