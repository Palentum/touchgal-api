package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/httpserver/middleware"
	"github.com/touchgal/developer/backend/internal/services/token"
)

type TokenHandler struct{ svc *token.Service }

func NewTokenHandler(svc *token.Service) *TokenHandler { return &TokenHandler{svc: svc} }

func (h *TokenHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r)
	tokens, err := h.svc.ListMine(r.Context(), user.ID)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, tokens)
}

func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r)
	var req struct {
		Name string `json:"name"`
	}
	if err := DecodeJSON(r, &req); err != nil {
		ErrorCode(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}
	result, err := h.svc.Create(r.Context(), user.ID, user.IsAdmin, req.Name)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusCreated, map[string]any{
		"id":          result.Token.ID,
		"name":        result.Token.Name,
		"token":       result.PlainToken,
		"tokenPrefix": result.Token.TokenPrefix,
		"minuteLimit": result.Token.MinuteLimit,
		"dailyLimit":  result.Token.DailyLimit,
	})
}

func (h *TokenHandler) UpdateMine(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		ErrorCode(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid token id")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := DecodeJSON(r, &req); err != nil {
		ErrorCode(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}
	updated, err := h.svc.UpdateNameMine(r.Context(), id, user.ID, req.Name)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, updated)
}

func (h *TokenHandler) DeleteMine(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		ErrorCode(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid token id")
		return
	}
	if err := h.svc.DeleteMine(r.Context(), id, user.ID); err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, map[string]bool{"deleted": true})
}
