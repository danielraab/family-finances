package account_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/auth"
)

// createAccount POSTs body and returns the decoded account, failing the
// test if the status is not 201.
func createAccount(t *testing.T, h http.Handler, user auth.User, body string) account.Account {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/accounts", strings.NewReader(body)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/accounts status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/accounts", rec)
	var acc account.Account
	if err := json.Unmarshal(rec.Body.Bytes(), &acc); err != nil {
		t.Fatal(err)
	}
	return acc
}

func TestHandlerCreateAccountWithIconAndColor(t *testing.T) {
	h, svc := newHandler(t)
	typ, _ := svc.CreateType(t.Context(), "u1", "Checking", "")
	user := auth.User{ID: "u1"}

	acc := createAccount(t, h, user, `{"title":"Main","type_id":"`+typ.ID+
		`","currency":"EUR","opening_date":"2024-01-01","icon":"wallet","color":"blue"}`)
	if acc.Icon != "wallet" || acc.Color != "blue" {
		t.Fatalf("icon/color = %q/%q, want wallet/blue", acc.Icon, acc.Color)
	}
}

func TestHandlerCreateAccountWithoutIconOrColor(t *testing.T) {
	h, svc := newHandler(t)
	typ, _ := svc.CreateType(t.Context(), "u1", "Checking", "")
	user := auth.User{ID: "u1"}

	acc := createAccount(t, h, user, `{"title":"Main","type_id":"`+typ.ID+
		`","currency":"EUR","opening_date":"2024-01-01"}`)
	if acc.Icon != "" || acc.Color != "" {
		t.Fatalf("icon/color = %q/%q, want both empty", acc.Icon, acc.Color)
	}
}

func TestHandlerCreateAccountMalformedIconRejected(t *testing.T) {
	h, svc := newHandler(t)
	typ, _ := svc.CreateType(t.Context(), "u1", "Checking", "")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/accounts", strings.NewReader(
		`{"title":"Main","type_id":"`+typ.ID+`","currency":"EUR","opening_date":"2024-01-01","icon":"Wallet Icon!"}`,
	)), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/accounts", rec)
}

func TestHandlerUpdateAccountMalformedColorRejected(t *testing.T) {
	h, svc := newHandler(t)
	typ, _ := svc.CreateType(t.Context(), "u1", "Checking", "")
	user := auth.User{ID: "u1"}
	acc := createAccount(t, h, user, `{"title":"Main","type_id":"`+typ.ID+
		`","currency":"EUR","opening_date":"2024-01-01"}`)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/accounts/"+acc.ID, strings.NewReader(
		`{"color":"not a token"}`,
	)), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "PATCH", "/api/accounts/"+acc.ID, rec)
}

func TestHandlerUpdateAccountClearsIconKeepsColor(t *testing.T) {
	h, svc := newHandler(t)
	typ, _ := svc.CreateType(t.Context(), "u1", "Checking", "")
	user := auth.User{ID: "u1"}
	acc := createAccount(t, h, user, `{"title":"Main","type_id":"`+typ.ID+
		`","currency":"EUR","opening_date":"2024-01-01","icon":"wallet","color":"blue"}`)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/accounts/"+acc.ID, strings.NewReader(
		`{"icon":""}`,
	)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "PATCH", "/api/accounts/"+acc.ID, rec)
	var got account.Account
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Icon != "" {
		t.Fatalf("icon = %q after clear, want empty", got.Icon)
	}
	if got.Color != "blue" {
		t.Fatalf("color = %q, want blue (unchanged)", got.Color)
	}
}

func TestHandlerUpdateAccountUnrelatedFieldPreservesIconAndColor(t *testing.T) {
	h, svc := newHandler(t)
	typ, _ := svc.CreateType(t.Context(), "u1", "Checking", "")
	user := auth.User{ID: "u1"}
	acc := createAccount(t, h, user, `{"title":"Main","type_id":"`+typ.ID+
		`","currency":"EUR","opening_date":"2024-01-01","icon":"wallet","color":"blue"}`)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/accounts/"+acc.ID, strings.NewReader(
		`{"title":"Renamed"}`,
	)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var got account.Account
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Icon != "wallet" || got.Color != "blue" {
		t.Fatalf("icon/color = %q/%q after unrelated update, want wallet/blue", got.Icon, got.Color)
	}
}
