package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/touchgal/developer/backend/internal/model"
)

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"token","extra":true}`))
	rec := httptest.NewRecorder()

	err := DecodeJSON(rec, req, &dst, smallJSONBodyLimit)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	if isRequestBodyTooLarge(err) {
		t.Fatalf("expected decode error, got body limit error: %v", err)
	}
}

func TestDecodeJSONRejectsBodyOverLimit(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"`+strings.Repeat("x", smallJSONBodyLimit)+`"}`))
	rec := httptest.NewRecorder()

	err := DecodeJSON(rec, req, &dst, smallJSONBodyLimit)
	if err == nil {
		t.Fatal("expected body limit error")
	}
	if !isRequestBodyTooLarge(err) {
		t.Fatalf("expected body limit error, got %v", err)
	}
}

func TestDecodeJSONRejectsTrailingBodyOverLimit(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"token"}`+strings.Repeat(" ", smallJSONBodyLimit)))
	rec := httptest.NewRecorder()

	err := DecodeJSON(rec, req, &dst, smallJSONBodyLimit)
	if err == nil {
		t.Fatal("expected body limit error")
	}
	if !isRequestBodyTooLarge(err) {
		t.Fatalf("expected body limit error, got %v", err)
	}
}

func TestDecodeJSONRejectsMultipleJSONValues(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"token"}{}`))
	rec := httptest.NewRecorder()

	err := DecodeJSON(rec, req, &dst, smallJSONBodyLimit)
	if err == nil {
		t.Fatal("expected extra JSON value error")
	}
	if isRequestBodyTooLarge(err) {
		t.Fatalf("expected decode error, got body limit error: %v", err)
	}
}

func TestRespondDecodeJSONErrorMapsLargeBody(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"`+strings.Repeat("x", smallJSONBodyLimit)+`"}`))
	decodeRec := httptest.NewRecorder()
	err := DecodeJSON(decodeRec, req, &dst, smallJSONBodyLimit)
	if err == nil {
		t.Fatal("expected body limit error")
	}

	responseRec := httptest.NewRecorder()
	respondDecodeJSONError(responseRec, err)

	if responseRec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, responseRec.Code)
	}
	if !strings.Contains(responseRec.Body.String(), `"code":"REQUEST_BODY_TOO_LARGE"`) {
		t.Fatalf("expected REQUEST_BODY_TOO_LARGE response, got %s", responseRec.Body.String())
	}
}

func TestErrorMapsVerificationCodeErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "cooldown", err: model.ErrCodeCooldown, status: http.StatusTooManyRequests, code: "CODE_COOLDOWN"},
		{name: "invalid", err: model.ErrInvalidCode, status: http.StatusBadRequest, code: "INVALID_CODE"},
		{name: "expired", err: model.ErrExpiredCode, status: http.StatusBadRequest, code: "EXPIRED_CODE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Error(rec, tt.err)

			if rec.Code != tt.status {
				t.Fatalf("expected status %d, got %d", tt.status, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), `"code":"`+tt.code+`"`) {
				t.Fatalf("expected %s response, got %s", tt.code, rec.Body.String())
			}
		})
	}
}
