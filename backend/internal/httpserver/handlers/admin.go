package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/httpserver/middleware"
	"github.com/touchgal/developer/backend/internal/model"
	"github.com/touchgal/developer/backend/internal/services/application"
	syncsvc "github.com/touchgal/developer/backend/internal/services/sync"
	"github.com/touchgal/developer/backend/internal/services/token"
	usersvc "github.com/touchgal/developer/backend/internal/services/users"
)

type AdminHandler struct {
	applications *application.Service
	tokens       *token.Service
	users        *usersvc.Service
	syncService  *syncsvc.Service
	syncStore    syncRunStore
}

type syncRunStore interface {
	ListRuns(ctx context.Context, limit int) ([]model.SyncRun, error)
}

func NewAdminHandler(apps *application.Service, tokens *token.Service, users *usersvc.Service, syncService *syncsvc.Service, syncStore syncRunStore) *AdminHandler {
	return &AdminHandler{applications: apps, tokens: tokens, users: users, syncService: syncService, syncStore: syncStore}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, limit := pageLimit(r)
	users, err := h.users.ListAdmin(r.Context(), r.URL.Query().Get("status"), r.URL.Query().Get("q"), page, limit)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, users)
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	admin, _ := middleware.CurrentUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		ErrorCode(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid user id")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := DecodeJSON(r, &req); err != nil {
		ErrorCode(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}
	user, err := h.users.UpdateStatus(r.Context(), admin.ID, id, req.Status)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, user)
}

func (h *AdminHandler) ListApplications(w http.ResponseWriter, r *http.Request) {
	page, limit := pageLimit(r)
	apps, err := h.applications.ListAdmin(r.Context(), r.URL.Query().Get("status"), page, limit)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, apps)
}

func (h *AdminHandler) ApproveApplication(w http.ResponseWriter, r *http.Request) {
	h.reviewApplication(w, r, model.ApplicationApproved)
}
func (h *AdminHandler) RejectApplication(w http.ResponseWriter, r *http.Request) {
	h.reviewApplication(w, r, model.ApplicationRejected)
}
func (h *AdminHandler) RevokeApplication(w http.ResponseWriter, r *http.Request) {
	h.reviewApplication(w, r, model.ApplicationRevoked)
}

func (h *AdminHandler) reviewApplication(w http.ResponseWriter, r *http.Request, status string) {
	admin, _ := middleware.CurrentUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		ErrorCode(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid application id")
		return
	}
	var req struct {
		MinuteLimit int    `json:"minuteLimit"`
		DailyLimit  int    `json:"dailyLimit"`
		ReviewNote  string `json:"reviewNote"`
	}
	if err := DecodeJSON(r, &req); err != nil {
		ErrorCode(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}
	app, err := h.applications.Review(r.Context(), id, admin.ID, status, req.ReviewNote, req.MinuteLimit, req.DailyLimit)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, app)
}

func (h *AdminHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	page, limit := pageLimit(r)
	tokens, err := h.tokens.ListAdmin(r.Context(), r.URL.Query().Get("status"), page, limit)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, tokens)
}

func (h *AdminHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		ErrorCode(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid token id")
		return
	}
	if err := h.tokens.DeleteByAdmin(r.Context(), id); err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *AdminHandler) SyncRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := h.syncStore.ListRuns(r.Context(), 50)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, runs)
}

func (h *AdminHandler) RunSync(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode string `json:"mode"`
	}
	if err := DecodeJSON(r, &req); err != nil {
		ErrorCode(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}
	run, err := h.syncService.Run(r.Context(), req.Mode)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, run)
}

func pageLimit(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
