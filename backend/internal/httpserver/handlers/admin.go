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
	"github.com/touchgal/developer/backend/internal/services/token"
	usersvc "github.com/touchgal/developer/backend/internal/services/users"
)

type AdminHandler struct {
	applications *application.Service
	tokens       *token.Service
	users        *usersvc.Service
	syncService  syncRunStarter
	syncEnabled  bool
	syncStore    syncRunStore
}

type syncRunStarter interface {
	Start(ctx context.Context, mode string) (*model.SyncRun, error)
}

type syncRunStore interface {
	ListRuns(ctx context.Context, limit int) ([]model.SyncRun, error)
}

func NewAdminHandler(apps *application.Service, tokens *token.Service, users *usersvc.Service, syncService syncRunStarter, syncStore syncRunStore, syncEnabled bool) *AdminHandler {
	return &AdminHandler{applications: apps, tokens: tokens, users: users, syncService: syncService, syncStore: syncStore, syncEnabled: syncEnabled}
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
		Email       *string `json:"email"`
		DisplayName *string `json:"displayName"`
		Status      *string `json:"status"`
		MinuteLimit *int    `json:"minuteLimit"`
		DailyLimit  *int    `json:"dailyLimit"`
	}
	if err := DecodeJSON(w, r, &req, smallJSONBodyLimit); err != nil {
		respondDecodeJSONError(w, err)
		return
	}
	user, err := h.users.UpdateAdmin(r.Context(), admin.ID, id, usersvc.AdminUpdate{
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Status:      req.Status,
		MinuteLimit: req.MinuteLimit,
		DailyLimit:  req.DailyLimit,
	})
	if err != nil {
		Error(w, err)
		return
	}
	h.tokens.InvalidateUser(id)
	Success(w, http.StatusOK, user)
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	admin, _ := middleware.CurrentUser(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		ErrorCode(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid user id")
		return
	}
	if err := h.users.DeleteAdmin(r.Context(), admin.ID, id); err != nil {
		Error(w, err)
		return
	}
	h.tokens.InvalidateUser(id)
	Success(w, http.StatusOK, map[string]bool{"deleted": true})
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
	if err := DecodeJSON(w, r, &req, applicationJSONBodyLimit); err != nil {
		respondDecodeJSONError(w, err)
		return
	}
	app, err := h.applications.Review(r.Context(), id, admin.ID, status, req.ReviewNote, req.MinuteLimit, req.DailyLimit)
	if err != nil {
		Error(w, err)
		return
	}
	h.tokens.InvalidateApplication(app.ID)
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
	if err := DecodeJSON(w, r, &req, smallJSONBodyLimit); err != nil {
		respondDecodeJSONError(w, err)
		return
	}
	if !h.syncEnabled || h.syncService == nil {
		Error(w, model.ErrSyncDisabled)
		return
	}
	run, err := h.syncService.Start(r.Context(), req.Mode)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusAccepted, run)
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
