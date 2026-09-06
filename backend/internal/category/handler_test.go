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

func TestHandlerAnyAuthenticatedUserCreatesAndListsOwnCategories(t *testing.T) {
	h, _ := newHandler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/categories", strings.NewReader(`{"name":"Groceries"}`)), auth.User{ID: "u1"}))
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
	if !strings.Contains(rec.Body.String(), "Groceries") {
		t.Fatalf("list body = %s, want it to include the created category", rec.Body)
	}
}

func TestHandlerCrossOwnerAccessNotFound(t *testing.T) {
	h, svc := newHandler()
	c, err := svc.Create(t.Context(), "u1", category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}

	for _, req := range []*http.Request{
		httptest.NewRequest("PATCH", "/api/categories/"+c.ID, strings.NewReader(`{"name":"X"}`)),
		httptest.NewRequest("DELETE", "/api/categories/"+c.ID, nil),
		httptest.NewRequest("POST", "/api/categories/"+c.ID+"/disable", nil),
		httptest.NewRequest("POST", "/api/categories/"+c.ID+"/enable", nil),
		httptest.NewRequest("POST", "/api/categories/"+c.ID+"/move-up", nil),
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, withUser(req, auth.User{ID: "u2"}))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s %s status = %d, want 404", req.Method, req.URL.Path, rec.Code)
		}
	}
}

func TestHandlerDeleteWithChildrenConflict(t *testing.T) {
	h, svc := newHandler()
	parent, err := svc.Create(t.Context(), "u1", category.New{Name: "Expenses"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(t.Context(), "u1", category.New{ParentID: &parent.ID, Name: "Groceries"}); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/categories/"+parent.ID, nil), auth.User{ID: "u1"}))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	conforms(t, "DELETE", "/api/categories/"+parent.ID, rec)
}

func TestHandlerDeleteLeafSucceeds(t *testing.T) {
	h, svc := newHandler()
	c, err := svc.Create(t.Context(), "u1", category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/categories/"+c.ID, nil), auth.User{ID: "u1"}))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestHandlerDisableEnable(t *testing.T) {
	h, svc := newHandler()
	c, err := svc.Create(t.Context(), "u1", category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/categories/"+c.ID+"/disable", nil), auth.User{ID: "u1"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("disable status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/categories/{id}/disable", rec)
	if !strings.Contains(rec.Body.String(), `"disabled":true`) {
		t.Fatalf("disable body = %s, want disabled:true", rec.Body)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/categories/"+c.ID+"/enable", nil), auth.User{ID: "u1"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("enable status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/categories/{id}/enable", rec)
	if !strings.Contains(rec.Body.String(), `"disabled":false`) {
		t.Fatalf("enable body = %s, want disabled:false", rec.Body)
	}
}

func TestHandlerMoveUpMoveDown(t *testing.T) {
	h, svc := newHandler()
	a, err := svc.Create(t.Context(), "u1", category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(t.Context(), "u1", category.New{Name: "B"})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/categories/"+b.ID+"/move-up", nil), auth.User{ID: "u1"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("move-up status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/categories/{id}/move-up", rec)

	got, err := svc.Get(t.Context(), "u1", b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.SortOrder != a.SortOrder {
		t.Fatalf("b.SortOrder after move-up = %d, want %d", got.SortOrder, a.SortOrder)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/categories/"+b.ID+"/move-down", nil), auth.User{ID: "u1"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("move-down status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/categories/{id}/move-down", rec)
}
