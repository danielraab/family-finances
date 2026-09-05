package settings_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/httpapi"
	"at.draab/familyfinances/internal/openapicheck"
	"at.draab/familyfinances/internal/settings"
	"at.draab/familyfinances/internal/storage/memory"
)

func conforms(t *testing.T, method, target string, rec *httptest.ResponseRecorder) {
	t.Helper()
	openapicheck.AssertResponse(t, method, target, rec.Code, rec.Header(), rec.Body.Bytes())
}

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	svc := settings.NewService(memory.NewSettingsStore())
	return settings.NewHandler(svc, settings.HandlerOptions{RenderError: httpapi.WriteError})
}

func withUser(req *http.Request, u auth.User) *http.Request {
	return req.WithContext(auth.WithUser(req.Context(), u))
}

func TestHandlerGetRequiresAuth(t *testing.T) {
	h := newHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/settings", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	conforms(t, "GET", "/api/settings", rec)
}

func TestHandlerGetReturnsDefaults(t *testing.T) {
	h := newHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/settings", nil), auth.User{ID: "u1"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	conforms(t, "GET", "/api/settings", rec)

	var body settings.Settings
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	want := settings.Settings{Language: "en", Timezone: "UTC", DefaultCurrency: "EUR", DisplayedDecimalPlaces: 2}
	if body != want {
		t.Fatalf("body = %+v, want %+v", body, want)
	}
}

func TestHandlerPutUpdatesOneField(t *testing.T) {
	h := newHandler(t)
	user := auth.User{ID: "u1"}

	putReq := withUser(httptest.NewRequest("PUT", "/api/settings", strings.NewReader(`{"language":"de"}`)), user)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, putReq)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", rec.Code)
	}
	conforms(t, "PUT", "/api/settings", rec)

	var body settings.Settings
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Language != "de" || body.Timezone != "UTC" || body.DefaultCurrency != "EUR" {
		t.Fatalf("body = %+v", body)
	}
}

func TestHandlerPutRejectsInvalidValue(t *testing.T) {
	h := newHandler(t)
	req := withUser(httptest.NewRequest("PUT", "/api/settings", strings.NewReader(`{"language":"fr"}`)), auth.User{ID: "u1"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	conforms(t, "PUT", "/api/settings", rec)
}

func TestHandlerPutRequiresAuth(t *testing.T) {
	h := newHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("PUT", "/api/settings", strings.NewReader(`{}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	conforms(t, "PUT", "/api/settings", rec)
}
