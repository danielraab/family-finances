package tag_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/httpapi"
	"at.draab/familyfinances/internal/openapicheck"
	"at.draab/familyfinances/internal/storage/memory"
	"at.draab/familyfinances/internal/tag"
)

func newHandler() http.Handler {
	svc := tag.NewService(memory.NewTagStore())
	return tag.NewHandler(svc, tag.HandlerOptions{RenderError: httpapi.WriteError})
}

func withUser(req *http.Request, u auth.User) *http.Request {
	return req.WithContext(auth.WithUser(req.Context(), u))
}

func conforms(t *testing.T, method, target string, rec *httptest.ResponseRecorder) {
	t.Helper()
	openapicheck.AssertResponse(t, method, target, rec.Code, rec.Header(), rec.Body.Bytes())
}

func TestHandlerListRequiresAuth(t *testing.T) {
	h := newHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/tags", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerCreateAndList(t *testing.T) {
	h := newHandler()
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/tags", strings.NewReader(`{"name":"groceries"}`)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/tags", rec)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/tags", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	conforms(t, "GET", "/api/tags", rec)
}

func TestHandlerDuplicateNameConflict(t *testing.T) {
	h := newHandler()
	user := auth.User{ID: "u1"}
	h.ServeHTTP(httptest.NewRecorder(), withUser(httptest.NewRequest("POST", "/api/tags", strings.NewReader(`{"name":"groceries"}`)), user))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/tags", strings.NewReader(`{"name":"groceries"}`)), user))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	conforms(t, "POST", "/api/tags", rec)
}
