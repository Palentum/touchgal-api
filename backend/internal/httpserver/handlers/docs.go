package handlers

import (
	"net/http"
	"os"
)

type DocsHandler struct{}

func (DocsHandler) OpenAPI(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("internal/openapi/openapi.yaml")
	if err != nil {
		data, err = os.ReadFile("backend/internal/openapi/openapi.yaml")
	}
	if err != nil {
		ErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "OpenAPI document is not available")
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (DocsHandler) Swagger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>TouchGal Developer API Docs</title><script type="module" src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script></head><body><scalar-api-reference configuration='{"url":"/openapi.yaml"}'></scalar-api-reference></body></html>`))
}
