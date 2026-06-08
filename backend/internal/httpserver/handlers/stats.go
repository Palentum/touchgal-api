package handlers

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/httpserver/middleware"
	"github.com/touchgal/developer/backend/internal/services/stats"
)

type StatsHandler struct{ svc *stats.Service }

func NewStatsHandler(svc *stats.Service) *StatsHandler { return &StatsHandler{svc: svc} }

func (h *StatsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r)
	days, tokenID := statsQuery(r)
	data, err := h.svc.Summary(r.Context(), user.ID, days, tokenID)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, data)
}

func (h *StatsHandler) Trend(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r)
	days, tokenID := statsQuery(r)
	data, err := h.svc.Trend(r.Context(), user.ID, days, tokenID)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, data)
}

func (h *StatsHandler) Sources(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r)
	days, tokenID := statsQuery(r)
	data, err := h.svc.Sources(r.Context(), user.ID, days, tokenID)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, data)
}

func (h *StatsHandler) Endpoints(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r)
	days, tokenID := statsQuery(r)
	data, err := h.svc.Endpoints(r.Context(), user.ID, days, tokenID)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, data)
}

func statsQuery(r *http.Request) (int, *uuid.UUID) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if raw := r.URL.Query().Get("tokenId"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			return days, &id
		}
	}
	return days, nil
}
