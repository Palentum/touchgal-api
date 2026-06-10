package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
