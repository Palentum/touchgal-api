package handlers

import (
	"net/http"

	"github.com/touchgal/developer/backend/internal/httpserver/middleware"
	"github.com/touchgal/developer/backend/internal/model"
	"github.com/touchgal/developer/backend/internal/services/application"
)

type ApplicationHandler struct{ svc *application.Service }

func NewApplicationHandler(svc *application.Service) *ApplicationHandler {
	return &ApplicationHandler{svc: svc}
}

func (h *ApplicationHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r)
	apps, err := h.svc.ListMine(r.Context(), user.ID)
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, apps)
}

func (h *ApplicationHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r)
	var req struct {
		ApplicantName         string `json:"applicantName"`
		ProjectName           string `json:"projectName"`
		ProjectURL            string `json:"projectUrl"`
		ExpectedDailyRequests int    `json:"expectedDailyRequests"`
		UsageScenario         string `json:"usageScenario"`
		AgreeToTerms          bool   `json:"agreeToTerms"`
	}
	if err := DecodeJSON(r, &req); err != nil {
		ErrorCode(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body")
		return
	}
	app, err := h.svc.Create(r.Context(), user.ID, model.CreateApplicationInput{ApplicantName: req.ApplicantName, ProjectName: req.ProjectName, ProjectURL: req.ProjectURL, ExpectedDailyRequests: req.ExpectedDailyRequests, UsageScenario: req.UsageScenario, AgreeToTerms: req.AgreeToTerms})
	if err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusCreated, app)
}
