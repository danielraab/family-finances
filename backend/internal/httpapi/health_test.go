package httpapi

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandlerHealthyDB(t *testing.T) {
	rec := httptest.NewRecorder()
	healthHandler(stubPinger{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, _ := io.ReadAll(rec.Result().Body)
	if string(body) != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
}

func TestHealthHandlerDBDown(t *testing.T) {
	rec := httptest.NewRecorder()
	h := healthHandler(stubPinger{err: errors.New("connection refused")})
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/healthz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	body, _ := io.ReadAll(rec.Result().Body)
	if string(body) == "ok" {
		t.Fatalf("body = %q, want a failure message", body)
	}
}

func TestHealthHandlerNilDB(t *testing.T) {
	rec := httptest.NewRecorder()
	healthHandler(nil).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/healthz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}
