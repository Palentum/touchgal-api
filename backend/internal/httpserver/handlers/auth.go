package handlers

import (
	"net/http"
	"time"

	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/httpserver/middleware"
	"github.com/touchgal/developer/backend/internal/services/auth"
)

type AuthHandler struct {
	cfg config.Config
	svc *auth.Service
}

func NewAuthHandler(cfg config.Config, svc *auth.Service) *AuthHandler {
	return &AuthHandler{cfg: cfg, svc: svc}
}

type registerStartRequest struct {
	Email          string `json:"email"`
	DisplayName    string `json:"displayName"`
	TurnstileToken string `json:"turnstileToken"`
}
type emailRequest struct {
	Email          string `json:"email"`
	TurnstileToken string `json:"turnstileToken"`
}
type verifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}
type codeStartResponse struct {
	Sent                  bool `json:"sent"`
	ExpiresInSeconds      int  `json:"expiresInSeconds"`
	ResendCooldownSeconds int  `json:"resendCooldownSeconds"`
}

func (h *AuthHandler) RegisterStart(w http.ResponseWriter, r *http.Request) {
	var req registerStartRequest
	if err := DecodeJSON(w, r, &req, smallJSONBodyLimit); err != nil {
		respondDecodeJSONError(w, err)
		return
	}
	if err := h.svc.RequestRegisterCode(r.Context(), req.Email, req.DisplayName, middleware.ClientIP(r), req.TurnstileToken); err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, h.codeStartResponse())
}

func (h *AuthHandler) RegisterVerify(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := DecodeJSON(w, r, &req, smallJSONBodyLimit); err != nil {
		respondDecodeJSONError(w, err)
		return
	}
	result, err := h.svc.VerifyRegister(r.Context(), req.Email, req.Code, r.UserAgent(), middleware.ClientIP(r))
	if err != nil {
		Error(w, err)
		return
	}
	h.setSessionCookie(w, result.SessionToken, result.ExpiresAt)
	Success(w, http.StatusOK, result.User)
}

func (h *AuthHandler) LoginStart(w http.ResponseWriter, r *http.Request) {
	var req emailRequest
	if err := DecodeJSON(w, r, &req, smallJSONBodyLimit); err != nil {
		respondDecodeJSONError(w, err)
		return
	}
	if err := h.svc.RequestLoginCode(r.Context(), req.Email, middleware.ClientIP(r), req.TurnstileToken); err != nil {
		Error(w, err)
		return
	}
	Success(w, http.StatusOK, h.codeStartResponse())
}

func (h *AuthHandler) codeStartResponse() codeStartResponse {
	return codeStartResponse{
		Sent:                  true,
		ExpiresInSeconds:      int(h.cfg.EmailCodeTTL().Seconds()),
		ResendCooldownSeconds: int(h.cfg.EmailCooldown().Seconds()),
	}
}

func (h *AuthHandler) LoginVerify(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := DecodeJSON(w, r, &req, smallJSONBodyLimit); err != nil {
		respondDecodeJSONError(w, err)
		return
	}
	result, err := h.svc.VerifyLogin(r.Context(), req.Email, req.Code, r.UserAgent(), middleware.ClientIP(r))
	if err != nil {
		Error(w, err)
		return
	}
	h.setSessionCookie(w, result.SessionToken, result.ExpiresAt)
	Success(w, http.StatusOK, result.User)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(h.cfg.SessionCookieName); err == nil {
		_ = h.svc.RevokeSession(r.Context(), cookie.Value)
	}
	h.clearSessionCookie(w)
	Success(w, http.StatusOK, map[string]bool{"loggedOut": true})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r)
	if !ok {
		ErrorCode(w, http.StatusUnauthorized, "UNAUTHORIZED", "Login required")
		return
	}
	Success(w, http.StatusOK, user)
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: h.cfg.SessionCookieName, Value: token, Path: "/", Domain: h.cfg.SessionCookieDomain,
		Expires: expiresAt, MaxAge: int(time.Until(expiresAt).Seconds()), HttpOnly: true,
		Secure: h.cfg.SessionCookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: h.cfg.SessionCookieName, Value: "", Path: "/", Domain: h.cfg.SessionCookieDomain, MaxAge: -1, HttpOnly: true, Secure: h.cfg.SessionCookieSecure, SameSite: http.SameSiteLaxMode})
}
