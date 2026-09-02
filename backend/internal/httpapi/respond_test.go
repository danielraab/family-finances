package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusCreated, map[string]int{"n": 1})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", ct)
	}
	var got map[string]int
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["n"] != 1 {
		t.Fatalf("body = %v", got)
	}
}

func TestWriteErrorUnknownIs500(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, httptest.NewRequest(http.MethodGet, "/", nil), errors.New("boom"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	var got map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["error"] != "internal server error" {
		t.Fatalf("error = %q, want %q", got["error"], "internal server error")
	}
	if _, ok := got["request_id"]; !ok {
		t.Fatalf("body missing request_id key: %v", got)
	}
}

func TestWriteErrorMappedSentinel(t *testing.T) {
	sentinel := errors.New("not found")
	registerErrStatus(sentinel, http.StatusNotFound)
	t.Cleanup(func() { delete(errStatus, sentinel) })

	rec := httptest.NewRecorder()
	writeError(rec, httptest.NewRequest(http.MethodGet, "/", nil), sentinel)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
