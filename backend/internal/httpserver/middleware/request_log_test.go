package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/model"
)

type recordingRequestLogSink struct {
	logs []model.RequestLog
}

func (s *recordingRequestLogSink) EnqueueRequestLog(log model.RequestLog) bool {
	s.logs = append(s.logs, log)
	return true
}

func TestAPIRequestLogEnqueuesRouteStatusAndTokenContext(t *testing.T) {
	sink := &recordingRequestLogSink{}
	tokenID := uuid.New()
	userID := uuid.New()
	applicationID := uuid.New()

	router := chi.NewRouter()
	router.Use(APIRequestLog(sink))
	router.Get("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req = req.WithContext(WithTokenInfo(req.Context(), &model.TokenAuthInfo{
		Token: model.APIToken{ID: tokenID, UserID: userID, ApplicationID: applicationID},
	}))
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("response status got %d", res.Code)
	}
	if len(sink.logs) != 1 {
		t.Fatalf("enqueued logs got %d", len(sink.logs))
	}
	log := sink.logs[0]
	if log.Route != "/v1/me" || log.StatusCode != http.StatusNoContent || log.Method != http.MethodGet || log.Path != "/v1/me" {
		t.Fatalf("unexpected request log: %#v", log)
	}
	if log.TokenID == nil || *log.TokenID != tokenID {
		t.Fatalf("token id not captured: %#v", log.TokenID)
	}
	if log.UserID == nil || *log.UserID != userID {
		t.Fatalf("user id not captured: %#v", log.UserID)
	}
	if log.ApplicationID == nil || *log.ApplicationID != applicationID {
		t.Fatalf("application id not captured: %#v", log.ApplicationID)
	}
}

func TestAPIRequestLogTruncatesDatabaseBoundedFields(t *testing.T) {
	sink := &recordingRequestLogSink{}
	router := chi.NewRouter()
	router.Use(APIRequestLog(sink))
	router.Get("/v1/me", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("User-Agent", strings.Repeat("a", 1100))
	req.Header.Set("Origin", strings.Repeat("b", 1100))
	req.Header.Set("Referer", strings.Repeat("界", 1100))
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if len(sink.logs) != 1 {
		t.Fatalf("enqueued logs got %d", len(sink.logs))
	}
	log := sink.logs[0]
	if len(log.UserAgent) != 1000 {
		t.Fatalf("user agent length got %d", len(log.UserAgent))
	}
	if len(log.Origin) != 1000 {
		t.Fatalf("origin length got %d", len(log.Origin))
	}
	if got := len([]rune(log.Referer)); got != 1000 {
		t.Fatalf("referer runes got %d", got)
	}
}

func TestAPIRequestLogAllowsNilSink(t *testing.T) {
	handler := APIRequestLog(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/me", nil))
	if res.Code != http.StatusNoContent {
		t.Fatalf("response status got %d", res.Code)
	}
}
