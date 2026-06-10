package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/touchgal/developer/backend/internal/model"
)

type RequestLogSink interface {
	EnqueueRequestLog(log model.RequestLog) bool
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func APIRequestLog(sink RequestLogSink) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			info, _ := CurrentToken(r)
			log := model.RequestLog{
				Method:     truncateLogField(r.Method, 20),
				Path:       truncateLogField(r.URL.Path, 500),
				Route:      truncateLogField(chi.RouteContext(r.Context()).RoutePattern(), 300),
				StatusCode: sw.status,
				LatencyMS:  int(time.Since(started).Milliseconds()),
				IP:         truncateLogField(ClientIP(r), 100),
				UserAgent:  truncateLogField(r.UserAgent(), 1000),
				Origin:     truncateLogField(r.Header.Get("Origin"), 1000),
				Referer:    truncateLogField(r.Referer(), 1000),
			}
			if info != nil {
				log.TokenID = &info.Token.ID
				log.UserID = &info.Token.UserID
				log.ApplicationID = &info.Token.ApplicationID
			}
			if sink != nil {
				sink.EnqueueRequestLog(log)
			}
		})
	}
}

func truncateLogField(value string, maxRunes int) string {
	if maxRunes <= 0 || len(value) <= maxRunes {
		return value
	}
	runes := 0
	for idx := range value {
		if runes == maxRunes {
			return value[:idx]
		}
		runes++
	}
	return value
}
