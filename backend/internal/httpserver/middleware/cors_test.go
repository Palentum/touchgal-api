package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/touchgal/developer/backend/internal/config"
)

func TestCORSPermitsTokenMutationMethods(t *testing.T) {
	handler := CORS(config.Config{PublicURL: "https://portal.example"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodOptions, "/tokens/id", nil)
	req.Header.Set("Origin", "https://portal.example")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	methods := res.Header().Get("Access-Control-Allow-Methods")
	for _, method := range []string{"PATCH", "DELETE"} {
		if !strings.Contains(methods, method) {
			t.Fatalf("expected %s in allowed methods, got %q", method, methods)
		}
	}
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected preflight 204, got %d", res.Code)
	}
}
