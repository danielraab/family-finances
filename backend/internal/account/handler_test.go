package account_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/httpapi"
	"at.draab/familyfinances/internal/openapicheck"
	"at.draab/familyfinances/internal/storage/memory"
)

func newHandler(t *testing.T) (http.Handler, *account.Service) {
	t.Helper()
	svc := account.NewService(memory.NewAccountStore())
	return account.NewHandler(svc, account.HandlerOptions{RenderError: httpapi.WriteError}), svc
}

func withUser(req *http.Request, u auth.User) *http.Request {
	return req.WithContext(auth.WithUser(req.Context(), u))
}

func conforms(t *testing.T, method, target string, rec *httptest.ResponseRecorder) {
	t.Helper()
	openapicheck.AssertResponse(t, method, target, rec.Code, rec.Header(), rec.Body.Bytes())
}

func TestHandlerListRequiresAuth(t *testing.T) {
	h, _ := newHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/accounts", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerCreateAndGet(t *testing.T) {
	h, svc := newHandler(t)
	typ, err := svc.CreateType(t.Context(), "Checking")
	if err != nil {
		t.Fatal(err)
	}
	user := auth.User{ID: "u1"}

	body := `{"title":"Main","type_id":"` + typ.ID + `","currency":"EUR","opening_date":"2024-01-01"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/accounts", strings.NewReader(body)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/accounts", rec)
	var created account.Account
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/accounts/"+created.ID, nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d", rec.Code)
	}
	conforms(t, "GET", "/api/accounts/"+created.ID, rec)
}

func TestHandlerCrossOwnerGetIsNotFound(t *testing.T) {
	h, svc := newHandler(t)
	typ, _ := svc.CreateType(t.Context(), "Checking")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(t.Context(), "u1", account.New{Title: "X", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/accounts/"+acc.ID, nil), auth.User{ID: "u2"}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	conforms(t, "GET", "/api/accounts/"+acc.ID, rec)
}

func TestHandlerDisableEnable(t *testing.T) {
	h, svc := newHandler(t)
	typ, _ := svc.CreateType(t.Context(), "Checking")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(t.Context(), "u1", account.New{Title: "X", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/accounts/"+acc.ID+"/disable", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("disable status = %d", rec.Code)
	}
	var got account.Account
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Disabled {
		t.Fatalf("Disabled = false after disable")
	}
	conforms(t, "POST", "/api/accounts/"+acc.ID+"/disable", rec)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/accounts/"+acc.ID+"/enable", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("enable status = %d", rec.Code)
	}
	conforms(t, "POST", "/api/accounts/"+acc.ID+"/enable", rec)
}

func TestHandlerSoftDelete(t *testing.T) {
	h, svc := newHandler(t)
	typ, _ := svc.CreateType(t.Context(), "Checking")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(t.Context(), "u1", account.New{Title: "X", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/accounts/"+acc.ID, nil), user))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", rec.Code)
	}
	conforms(t, "DELETE", "/api/accounts/"+acc.ID, rec)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/accounts", nil), user))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("list after delete = %d %s", rec.Code, rec.Body.String())
	}
	conforms(t, "GET", "/api/accounts", rec)
}

func TestHandlerAccountTypesNonAdminForbidden(t *testing.T) {
	h, _ := newHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/account-types", strings.NewReader(`{"name":"Savings"}`)), auth.User{ID: "u1", IsAdmin: false}))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	conforms(t, "POST", "/api/account-types", rec)
}

func TestHandlerAccountTypesAdminCanCreateAndAnyoneCanList(t *testing.T) {
	h, _ := newHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/account-types", strings.NewReader(`{"name":"Savings"}`)), auth.User{ID: "admin1", IsAdmin: true}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/account-types", rec)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/account-types", nil), auth.User{ID: "u1"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	conforms(t, "GET", "/api/account-types", rec)
}

func TestHandlerDeleteInUseAccountTypeConflict(t *testing.T) {
	h, svc := newHandler(t)
	typ, _ := svc.CreateType(t.Context(), "Checking")
	opening, _ := account.ParseDate("2024-01-01")
	if _, err := svc.Create(t.Context(), "u1", account.New{Title: "X", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening}); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/account-types/"+typ.ID, nil), auth.User{ID: "admin1", IsAdmin: true}))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	conforms(t, "DELETE", "/api/account-types/"+typ.ID, rec)
}

func TestHandlerClosingBeforeOpeningRejected(t *testing.T) {
	h, svc := newHandler(t)
	typ, _ := svc.CreateType(t.Context(), "Checking")
	user := auth.User{ID: "u1"}

	body := `{"title":"X","type_id":"` + typ.ID + `","currency":"EUR","opening_date":"2024-06-01","closing_date":"2024-01-01"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/accounts", strings.NewReader(body)), user))
	conforms(t, "POST", "/api/accounts", rec)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
}
