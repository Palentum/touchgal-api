package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/touchgal/developer/backend/internal/httpserver/middleware"
	"github.com/touchgal/developer/backend/internal/services/publicapi"
)

type PublicAPIHandler struct{ svc *publicapi.Service }

func NewPublicAPIHandler(svc *publicapi.Service) *PublicAPIHandler {
	return &PublicAPIHandler{svc: svc}
}

func (h *PublicAPIHandler) Me(w http.ResponseWriter, r *http.Request) {
	info, _ := middleware.CurrentToken(r)
	Success(w, http.StatusOK, map[string]any{
		"tokenPrefix":       info.Token.TokenPrefix,
		"applicationId":     info.Token.ApplicationID,
		"applicationStatus": info.ApplicationStatus,
		"minuteLimit":       info.EffectiveMinuteLimit(),
		"dailyLimit":        info.EffectiveDailyLimit(),
	})
}

func (h *PublicAPIHandler) Search(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		keyword = r.URL.Query().Get("q")
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	allowNsfw, err := publicapi.ParseAllowNsfw(r.URL.Query().Get("allowNsfw"))
	if err != nil {
		Error(w, err)
		return
	}
	result, err := h.svc.Search(r.Context(), keyword, page, limit, allowNsfw)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, result)
}

func (h *PublicAPIHandler) Detail(w http.ResponseWriter, r *http.Request) {
	allowNsfw, err := publicapi.ParseAllowNsfw(r.URL.Query().Get("allowNsfw"))
	if err != nil {
		Error(w, err)
		return
	}
	detail, err := h.svc.Detail(r.Context(), chi.URLParam(r, "uniqueId"), allowNsfw)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, detail)
}

func (h *PublicAPIHandler) Resources(w http.ResponseWriter, r *http.Request) {
	allowNsfw, err := publicapi.ParseAllowNsfw(r.URL.Query().Get("allowNsfw"))
	if err != nil {
		Error(w, err)
		return
	}
	result, err := h.svc.Resources(r.Context(), chi.URLParam(r, "uniqueId"), allowNsfw)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, result)
}

func (h *PublicAPIHandler) Patches(w http.ResponseWriter, r *http.Request) {
	allowNsfw, err := publicapi.ParseAllowNsfw(r.URL.Query().Get("allowNsfw"))
	if err != nil {
		Error(w, err)
		return
	}
	result, err := h.svc.Patches(r.Context(), chi.URLParam(r, "uniqueId"), allowNsfw)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, result)
}
