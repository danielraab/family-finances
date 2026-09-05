package category_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/httpapi"
	"at.draab/familyfinances/internal/openapicheck"
	"at.draab/familyfinances/internal/storage/memory"
)

func newHandler() (http.Handler, *category.Service) {
	svc := category.NewService(memory.NewCategoryStore())
	return category.NewHandler(svc, category.HandlerOptions{RenderError: httpapi.WriteError}), svc
}

func withUser(req *http.Request, u auth.User) *http.Request {
	return req.WithContext(auth.WithUser(req.Context(), u))
}

func conforms(t *testing.T, method, target string, rec *httptest.ResponseRecorder) {
	t.Helper()
	openapicheck.AssertResponse(t, method, target, rec.Code, rec.Header(), rec.Body.Bytes())
}

func TestHandlerListRequiresAuth(t *testing.T) {
	h, _ := newHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/categories", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerNonAdminCannotCreate(t *testing.T) {
	h, _ := newHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/categories", strings.NewReader(`{"name":"Groceries"}`)), auth.User{ID: "u1"}))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	conforms(t, "POST", "/api/categories", rec)
}

func TestHandlerAdminCreatesNonAdminLists(t *testing.T) {
	h, _ := newHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/categories", strings.NewReader(`{"name":"Groceries"}`)), auth.User{ID: "admin1", IsAdmin: true}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/categories", rec)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/categories", nil), auth.User{ID: "u1"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	conforms(t, "GET", "/api/categories", rec)
}

func TestHandlerDeleteWithChildrenConflict(t *testing.T) {
	h, svc := newHandler()
	parent, err := svc.Create(t.Context(), category.New{Name: "Expenses"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(t.Context(), category.New{ParentID: &parent.ID, Name: "Groceries"}); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/categories/"+parent.ID, nil), auth.User{ID: "admin1", IsAdmin: true}))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	conforms(t, "DELETE", "/api/categories/"+parent.ID, rec)
}
