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
	"at.draab/familyfinances/internal/storage/memory"
)

// newSharingHandler wires a handler over a service that has a UserLookup/
// Mailer configured, unlike newHandler's bare fixture — needed for any test
// exercising POST .../shares (email lookup).
func newSharingHandler(t *testing.T) (http.Handler, *account.Service, *fakeUsers, *fakeMailer) {
	t.Helper()
	store := memory.NewAccountStore()
	users := newFakeUsers()
	mailer := &fakeMailer{}
	svc := account.NewService(store, account.WithMailer(mailer), account.WithBaseURL("https://app.example"))
	svc.SetUserLookup(users)
	h := account.NewHandler(svc, account.HandlerOptions{RenderError: httpapi.WriteError})
	return h, svc, users, mailer
}

func mustAccount(t *testing.T, svc *account.Service, ownerID string) account.Account {
	t.Helper()
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(t.Context(), ownerID, account.New{Title: "Joint", Type: "Checking", Currency: "EUR", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}
	return acc
}

func TestHandlerListSharesEmpty(t *testing.T) {
	h, svc, _, _ := newSharingHandler(t)
	acc := mustAccount(t, svc, "u1")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/accounts/"+acc.ID+"/shares", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/accounts/"+acc.ID+"/shares", rec)
	var shares []account.AccountShare
	if err := json.Unmarshal(rec.Body.Bytes(), &shares); err != nil {
		t.Fatal(err)
	}
	if len(shares) != 0 {
		t.Fatalf("shares = %+v, want empty", shares)
	}
}

func TestHandlerListSharesNoPermissionIsNotFound(t *testing.T) {
	h, svc, _, _ := newSharingHandler(t)
	acc := mustAccount(t, svc, "u1")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/accounts/"+acc.ID+"/shares", nil), auth.User{ID: "stranger"}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	conforms(t, "GET", "/api/accounts/"+acc.ID+"/shares", rec)
}

func TestHandlerInviteShareMatched(t *testing.T) {
	h, svc, users, mailer := newSharingHandler(t)
	acc := mustAccount(t, svc, "u1")
	users.add("member@example.com", "u2", "Member")
	user := auth.User{ID: "u1", DisplayName: "Owner"}

	body := `{"email":"member@example.com","permission":"append"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/accounts/"+acc.ID+"/shares", strings.NewReader(body)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/accounts/"+acc.ID+"/shares", rec)
	var result struct {
		Matched bool                  `json:"matched"`
		Share   *account.AccountShare `json:"share"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Matched || result.Share == nil || result.Share.Permission != account.PermissionAppend {
		t.Fatalf("result = %+v", result)
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("mailer.sent = %+v, want one notification", mailer.sent)
	}
}

func TestHandlerInviteShareUnmatched(t *testing.T) {
	h, svc, _, _ := newSharingHandler(t)
	acc := mustAccount(t, svc, "u1")
	user := auth.User{ID: "u1"}

	body := `{"email":"nobody@example.com","permission":"view"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/accounts/"+acc.ID+"/shares", strings.NewReader(body)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200 for an unmatched email", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/accounts/"+acc.ID+"/shares", rec)
	var result struct {
		Matched bool `json:"matched"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Matched {
		t.Fatalf("result = %+v, want matched=false", result)
	}
}

func TestHandlerInviteShareForbiddenBelowOwnerTier(t *testing.T) {
	h, svc, users, _ := newSharingHandler(t)
	acc := mustAccount(t, svc, "u1")
	users.add("appender@example.com", "u2", "Appender")
	users.add("target@example.com", "u3", "Target")

	if _, err := svc.InviteShare(t.Context(), "u1", "Owner", acc.ID, "appender@example.com", account.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	body := `{"email":"target@example.com","permission":"view"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/accounts/"+acc.ID+"/shares", strings.NewReader(body)), auth.User{ID: "u2"}))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/accounts/"+acc.ID+"/shares", rec)
}

func TestHandlerUpdateAndRevokeShare(t *testing.T) {
	h, svc, users, _ := newSharingHandler(t)
	acc := mustAccount(t, svc, "u1")
	users.add("member@example.com", "u2", "Member")
	user := auth.User{ID: "u1"}
	if _, err := svc.InviteShare(t.Context(), "u1", "Owner", acc.ID, "member@example.com", account.PermissionView); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/accounts/"+acc.ID+"/shares/u2", strings.NewReader(`{"permission":"entry_admin"}`)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "PATCH", "/api/accounts/"+acc.ID+"/shares/u2", rec)
	var updated account.AccountShare
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Permission != account.PermissionEntryAdmin {
		t.Fatalf("updated = %+v", updated)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/accounts/"+acc.ID+"/shares/u2", nil), user))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "DELETE", "/api/accounts/"+acc.ID+"/shares/u2", rec)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/accounts/"+acc.ID, nil), auth.User{ID: "u2"}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status after revoke = %d, want 404", rec.Code)
	}
}

func TestHandlerSelfLeave(t *testing.T) {
	h, svc, users, _ := newSharingHandler(t)
	acc := mustAccount(t, svc, "u1")
	users.add("member@example.com", "u2", "Member")
	if _, err := svc.InviteShare(t.Context(), "u1", "Owner", acc.ID, "member@example.com", account.PermissionView); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/accounts/"+acc.ID+"/shares/u2", nil), auth.User{ID: "u2"}))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("self-leave status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "DELETE", "/api/accounts/"+acc.ID+"/shares/u2", rec)
}

func TestHandlerRevokeRealOwnerRejected(t *testing.T) {
	h, svc, _, _ := newSharingHandler(t)
	acc := mustAccount(t, svc, "u1")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/accounts/"+acc.ID+"/shares/u1", nil), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "DELETE", "/api/accounts/"+acc.ID+"/shares/u1", rec)
}
