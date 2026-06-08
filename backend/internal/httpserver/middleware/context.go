package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/touchgal/developer/backend/internal/model"
)

type contextKey string

const (
	userKey      contextKey = "current_user"
	tokenInfoKey contextKey = "current_token"
	requestIDKey contextKey = "request_id"
)

func WithUser(ctx context.Context, user *model.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}
func CurrentUser(r *http.Request) (*model.User, bool) {
	user, ok := r.Context().Value(userKey).(*model.User)
	return user, ok && user != nil
}
func WithTokenInfo(ctx context.Context, info *model.TokenAuthInfo) context.Context {
	return context.WithValue(ctx, tokenInfoKey, info)
}
func CurrentToken(r *http.Request) (*model.TokenAuthInfo, bool) {
	info, ok := r.Context().Value(tokenInfoKey).(*model.TokenAuthInfo)
	return info, ok && info != nil
}
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func ClientIP(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); value != "" {
		if first := strings.TrimSpace(strings.Split(value, ",")[0]); first != "" {
			return first
		}
	}
	if value := strings.TrimSpace(r.Header.Get("X-Real-IP")); value != "" {
		return value
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
