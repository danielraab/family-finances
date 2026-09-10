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

// mustCreate makes an account owned by ownerID with the given free-text type.
func mustCreate(t *testing.T, svc *account.Service, ownerID, title, typ string) account.Account {
	t.Helper()
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(t.Context(), ownerID, account.New{
		Title: title, Type: typ, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	return acc
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
	h, _ := newHandler(t)
	user := auth.User{ID: "u1"}

	body := `{"title":"Main","type":"Checking","currency":"EUR","opening_date":"2024-01-01"}`
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
	if created.Type != "Checking" {
		t.Fatalf("Type = %q, want Checking", created.Type)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/accounts/"+created.ID, nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d", rec.Code)
	}
	conforms(t, "GET", "/api/accounts/"+created.ID, rec)
}

func TestHandlerCreateTrimsType(t *testing.T) {
	h, _ := newHandler(t)
	user := auth.User{ID: "u1"}

	body := `{"title":"Main","type":"  Checking  ","currency":"EUR","opening_date":"2024-01-01"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/accounts", strings.NewReader(body)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body = %s", rec.Code, rec.Body)
	}
	var created account.Account
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Type != "Checking" {
		t.Fatalf("Type = %q, want the trimmed %q", created.Type, "Checking")
	}
}

func TestHandlerCreateBlankTypeRejected(t *testing.T) {
	h, _ := newHandler(t)
	user := auth.User{ID: "u1"}

	body := `{"title":"Main","type":"   ","currency":"EUR","opening_date":"2024-01-01"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/accounts", strings.NewReader(body)), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/accounts", rec)
}

func TestHandlerCrossOwnerGetIsNotFound(t *testing.T) {
	h, svc := newHandler(t)
	acc := mustCreate(t, svc, "u1", "X", "Checking")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/accounts/"+acc.ID, nil), auth.User{ID: "u2"}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	conforms(t, "GET", "/api/accounts/"+acc.ID, rec)
}

func TestHandlerDisableEnable(t *testing.T) {
	h, svc := newHandler(t)
	acc := mustCreate(t, svc, "u1", "X", "Checking")
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
	acc := mustCreate(t, svc, "u1", "X", "Checking")
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

func TestHandlerAccountTypesRequireAuth(t *testing.T) {
	h, _ := newHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/account-types", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandlerAccountTypesListDistinctInUseValues(t *testing.T) {
	h, svc := newHandler(t)
	mustCreate(t, svc, "u1", "A", "Savings")
	mustCreate(t, svc, "u1", "B", "Checking")
	mustCreate(t, svc, "u1", "C", "Checking") // duplicate collapses
	mustCreate(t, svc, "u2", "D", "Brokerage")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/account-types", nil), auth.User{ID: "u1"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	var types []string
	if err := json.Unmarshal(rec.Body.Bytes(), &types); err != nil {
		t.Fatal(err)
	}
	if want := []string{"Checking", "Savings"}; !equalStrings(types, want) {
		t.Fatalf("types = %v, want %v (sorted, deduped, caller-scoped)", types, want)
	}
	conforms(t, "GET", "/api/account-types", rec)
}

func TestHandlerAccountTypesEmptyForUserWithNoAccounts(t *testing.T) {
	h, _ := newHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/account-types", nil), auth.User{ID: "u1"}))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("list = %d %s, want 200 []", rec.Code, rec.Body.String())
	}
	conforms(t, "GET", "/api/account-types", rec)
}

func TestHandlerAccountTypeWritesAreGone(t *testing.T) {
	h, _ := newHandler(t)
	user := auth.User{ID: "u1"}
	for _, tc := range []struct{ method, target string }{
		{"POST", "/api/account-types"},
		{"PATCH", "/api/account-types/x"},
		{"DELETE", "/api/account-types/x"},
		{"POST", "/api/account-types/x/disable"},
		{"POST", "/api/account-types/x/enable"},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, withUser(httptest.NewRequest(tc.method, tc.target, strings.NewReader(`{}`)), user))
		if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: status = %d, want the route to no longer exist (404/405)", tc.method, tc.target, rec.Code)
		}
	}
}

func TestHandlerClosingBeforeOpeningRejected(t *testing.T) {
	h, _ := newHandler(t)
	user := auth.User{ID: "u1"}

	body := `{"title":"X","type":"Checking","currency":"EUR","opening_date":"2024-06-01","closing_date":"2024-01-01"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/accounts", strings.NewReader(body)), user))
	conforms(t, "POST", "/api/accounts", rec)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
