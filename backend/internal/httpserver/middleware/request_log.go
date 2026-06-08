package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/model"
)

type RequestLogStore interface {
	InsertRequestLog(ctx context.Context, log model.RequestLog) error
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func APIRequestLog(store RequestLogStore, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			info, _ := CurrentToken(r)
			log := model.RequestLog{
				Method:     r.Method,
				Path:       r.URL.Path,
				Route:      chi.RouteContext(r.Context()).RoutePattern(),
				StatusCode: sw.status,
				LatencyMS:  int(time.Since(started).Milliseconds()),
				IP:         ClientIP(r),
				UserAgent:  r.UserAgent(),
				Origin:     r.Header.Get("Origin"),
				Referer:    r.Referer(),
			}
			if info != nil {
				log.TokenID = &info.Token.ID
				log.UserID = &info.Token.UserID
				log.ApplicationID = &info.Token.ApplicationID
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				if err := store.InsertRequestLog(ctx, log); err != nil {
					logger.Warn().Err(err).Msg("api request log write failed")
				}
			}()
		})
	}
}
