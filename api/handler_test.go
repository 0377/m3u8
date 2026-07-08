package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0377/m3u8/api"
	"github.com/0377/m3u8/api/task"
)

func TestHealthHandler(t *testing.T) {
	h := api.NewHandler(&task.Manager{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	h.Health(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
}
