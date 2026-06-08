package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/touchgal/developer/backend/internal/model"
)

type APITokenService interface {
	Authenticate(ctx context.Context, raw string) (*model.TokenAuthInfo, error)
}

func APITokenAuth(service APITokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := ExtractAPIToken(r)
			info, err := service.Authenticate(r.Context(), raw)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid API token")
				return
			}
			next.ServeHTTP(w, r.WithContext(WithTokenInfo(r.Context(), info)))
		})
	}
}

func ExtractAPIToken(r *http.Request) string {
	if header := r.Header.Get("Authorization"); header != "" {
		prefix := "Bearer "
		if strings.HasPrefix(header, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(header, prefix))
		}
	}
	return strings.TrimSpace(r.Header.Get("X-API-Token"))
}
