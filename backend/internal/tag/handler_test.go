package tag_test

import (
	"encoding/json"
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

func TestHandlerCreateReportsZeroEntryCount(t *testing.T) {
	h := newHandler()
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/tags", strings.NewReader(`{"name":"groceries"}`)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"entry_count":0`) {
		t.Fatalf("body = %s, want entry_count 0", rec.Body)
	}
	conforms(t, "POST", "/api/tags", rec)
}

func TestHandlerDisableAndEnable(t *testing.T) {
	h := newHandler()
	user := auth.User{ID: "u1"}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/tags", strings.NewReader(`{"name":"groceries"}`)), user))
	var created struct{ ID string }
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/tags/"+created.ID+"/disable", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("disable status = %d, body = %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"disabled":true`) {
		t.Fatalf("body = %s, want disabled true", rec.Body)
	}
	conforms(t, "POST", "/api/tags/{id}/disable", rec)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/tags/"+created.ID+"/enable", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("enable status = %d, body = %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"disabled":false`) {
		t.Fatalf("body = %s, want disabled false", rec.Body)
	}
	conforms(t, "POST", "/api/tags/{id}/enable", rec)
}

func TestHandlerDisableRequiresAuth(t *testing.T) {
	h := newHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/tags/tag1/disable", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerDisableNonexistentTagNotFound(t *testing.T) {
	h := newHandler()
	user := auth.User{ID: "u1"}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/tags/nope/disable", nil), user))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	conforms(t, "POST", "/api/tags/{id}/disable", rec)
}

func TestHandlerDisableAnotherUsersTagNotFound(t *testing.T) {
	h := newHandler()
	owner := auth.User{ID: "u1"}
	other := auth.User{ID: "u2"}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/tags", strings.NewReader(`{"name":"groceries"}`)), owner))
	var created struct{ ID string }
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/tags/"+created.ID+"/disable", nil), other))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
