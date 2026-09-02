package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheckURL(t *testing.T) {
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
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
