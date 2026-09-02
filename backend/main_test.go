package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealthz(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", handleHealthz)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, _ := io.ReadAll(rec.Result().Body)
	if string(body) != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
}

func TestHealthcheckURL(t *testing.T) {
	okSrv := httptest.NewServer(http.HandlerFunc(handleHealthz))
	defer okSrv.Close()
	if code := healthcheckURL(okSrv.URL); code != 0 {
		t.Fatalf("healthy server: exit code = %d, want 0", code)
	}

	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer badSrv.Close()
	if code := healthcheckURL(badSrv.URL); code != 1 {
		t.Fatalf("unhealthy server: exit code = %d, want 1", code)
	}

	if code := healthcheckURL("http://127.0.0.1:0/api/healthz"); code != 1 {
		t.Fatalf("unreachable server: exit code = %d, want 1", code)
	}
}
