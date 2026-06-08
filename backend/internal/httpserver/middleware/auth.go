package middleware

import (
	"context"
	"net/http"

	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/model"
)

type SessionAuthService interface {
	CurrentUser(ctx context.Context, rawSessionToken string) (*model.User, error)
}

func SessionAuth(cfg config.Config, service SessionAuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cfg.SessionCookieName)
			if err == nil && cookie.Value != "" {
				if user, err := service.CurrentUser(r.Context(), cookie.Value); err == nil {
					r = r.WithContext(WithUser(r.Context(), user))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := CurrentUser(r); !ok {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Login required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := CurrentUser(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Login required")
			return
		}
		if !user.IsAdmin {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin permission required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
