package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"at.draab/familyfinances/internal/openapicheck"
)

func TestVersionHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	versionHandler(rec, httptest.NewRequest(http.MethodGet, "/api/version", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body BuildInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// version/commit are whatever this test binary's own build reports
	// (unstamped, so buildinfo falls back to VCS info or empty) — the
	// contract is that the fields are always present strings, never null,
	// which openapicheck below verifies against the schema.

	openapicheck.AssertResponse(t, "GET", "/api/version", rec.Code, rec.Header(), rec.Body.Bytes())
}
