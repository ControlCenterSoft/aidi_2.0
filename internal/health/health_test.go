package health

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ControlCenterSoft/aidi_2.0/internal/buildinfo"
)

func TestLive(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	Live(buildinfo.New("dev", "", "")).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected no-store cache policy")
	}
}

func TestReady(t *testing.T) {
	build := buildinfo.New("dev", "", "")

	ready := httptest.NewRecorder()
	Ready(build, func() bool { return true }).ServeHTTP(
		ready,
		httptest.NewRequest(http.MethodGet, "/ready", nil),
	)
	if ready.Code != http.StatusOK || !strings.Contains(ready.Body.String(), `"status":"ready"`) {
		t.Fatalf("unexpected ready response: code=%d body=%s", ready.Code, ready.Body.String())
	}

	notReady := httptest.NewRecorder()
	Ready(build, func() bool { return false }).ServeHTTP(
		notReady,
		httptest.NewRequest(http.MethodGet, "/ready", nil),
	)
	if notReady.Code != http.StatusServiceUnavailable || !strings.Contains(notReady.Body.String(), `"status":"not_ready"`) {
		t.Fatalf("unexpected not-ready response: code=%d body=%s", notReady.Code, notReady.Body.String())
	}
}
