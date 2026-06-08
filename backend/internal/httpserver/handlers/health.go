package handlers

import "net/http"

type HealthHandler struct{}

func (HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	Success(w, http.StatusOK, map[string]string{"status": "ok", "version": "v1"})
}
