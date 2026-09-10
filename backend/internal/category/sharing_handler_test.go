package category_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/httpapi"
	"at.draab/familyfinances/internal/storage/memory"
)

// newSharingHandler wires a handler over a service that has a UserLookup/
// Mailer configured, unlike newHandler's bare fixture — needed for any test
// exercising POST .../shares (email lookup).
func newSharingHandler(t *testing.T) (http.Handler, *category.Service, *fakeUsers, *fakeMailer) {
	t.Helper()
	store := memory.NewCategoryStore()
	users := newFakeUsers()
	mailer := &fakeMailer{}
	svc := category.NewService(store, category.WithMailer(mailer), category.WithBaseURL("https://app.example"))
	svc.SetUserLookup(users)
	h := category.NewHandler(svc, category.HandlerOptions{RenderError: httpapi.WriteError})
	return h, svc, users, mailer
}

func mustCategory(t *testing.T, svc *category.Service, ownerID string) category.Category {
	t.Helper()
	cat, err := svc.Create(t.Context(), ownerID, category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func TestCategoryHandlerListSharesEmpty(t *testing.T) {
	h, svc, _, _ := newSharingHandler(t)
	cat := mustCategory(t, svc, "u1")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/categories/"+cat.ID+"/shares", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/categories/"+cat.ID+"/shares", rec)
	var shares []category.CategoryShare
	if err := json.Unmarshal(rec.Body.Bytes(), &shares); err != nil {
		t.Fatal(err)
	}
	if len(shares) != 0 {
		t.Fatalf("shares = %+v, want empty", shares)
	}
}

func TestCategoryHandlerListSharesNoPermissionIsNotFound(t *testing.T) {
	h, svc, _, _ := newSharingHandler(t)
	cat := mustCategory(t, svc, "u1")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/categories/"+cat.ID+"/shares", nil), auth.User{ID: "stranger"}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	conforms(t, "GET", "/api/categories/"+cat.ID+"/shares", rec)
}

func TestCategoryHandlerGetIncludesSharedCategory(t *testing.T) {
	h, svc, users, _ := newSharingHandler(t)
	cat := mustCategory(t, svc, "u1")
	users.add("member@example.com", "u2", "Member")
	if _, err := svc.InviteShare(t.Context(), "u1", "Owner", cat.ID, "member@example.com", category.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/categories/"+cat.ID, nil), auth.User{ID: "u2"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/categories/"+cat.ID, rec)
	var got category.Category
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Permission != category.PermissionAppend || !got.Shared {
		t.Fatalf("got = %+v", got)
	}
}

func TestCategoryHandlerInviteShareMatched(t *testing.T) {
	h, svc, users, mailer := newSharingHandler(t)
	cat := mustCategory(t, svc, "u1")
	users.add("member@example.com", "u2", "Member")
	user := auth.User{ID: "u1", DisplayName: "Owner"}

	body := `{"email":"member@example.com","permission":"append"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/categories/"+cat.ID+"/shares", strings.NewReader(body)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/categories/"+cat.ID+"/shares", rec)
	var result struct {
		Matched bool                    `json:"matched"`
		Share   *category.CategoryShare `json:"share"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Matched || result.Share == nil || result.Share.Permission != category.PermissionAppend {
		t.Fatalf("result = %+v", result)
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("mailer.sent = %+v, want one notification", mailer.sent)
	}
}

func TestCategoryHandlerInviteShareUnmatched(t *testing.T) {
	h, svc, _, _ := newSharingHandler(t)
	cat := mustCategory(t, svc, "u1")
	user := auth.User{ID: "u1"}

	body := `{"email":"nobody@example.com","permission":"view"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/categories/"+cat.ID+"/shares", strings.NewReader(body)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200 for an unmatched email", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/categories/"+cat.ID+"/shares", rec)
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

func TestCategoryHandlerInviteShareForbiddenForNonOwner(t *testing.T) {
	h, svc, users, _ := newSharingHandler(t)
	cat := mustCategory(t, svc, "u1")
	users.add("appender@example.com", "u2", "Appender")
	users.add("target@example.com", "u3", "Target")

	if _, err := svc.InviteShare(t.Context(), "u1", "Owner", cat.ID, "appender@example.com", category.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	body := `{"email":"target@example.com","permission":"view"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/categories/"+cat.ID+"/shares", strings.NewReader(body)), auth.User{ID: "u2"}))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/categories/"+cat.ID+"/shares", rec)
}

func TestCategoryHandlerUpdateAndRevokeShare(t *testing.T) {
	h, svc, users, _ := newSharingHandler(t)
	cat := mustCategory(t, svc, "u1")
	users.add("member@example.com", "u2", "Member")
	user := auth.User{ID: "u1"}
	if _, err := svc.InviteShare(t.Context(), "u1", "Owner", cat.ID, "member@example.com", category.PermissionView); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/categories/"+cat.ID+"/shares/u2", strings.NewReader(`{"permission":"append"}`)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "PATCH", "/api/categories/"+cat.ID+"/shares/u2", rec)
	var updated category.CategoryShare
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Permission != category.PermissionAppend {
		t.Fatalf("updated = %+v", updated)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/categories/"+cat.ID+"/shares/u2", nil), user))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "DELETE", "/api/categories/"+cat.ID+"/shares/u2", rec)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/categories/"+cat.ID, nil), auth.User{ID: "u2"}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status after revoke = %d, want 404", rec.Code)
	}
}

func TestCategoryHandlerSelfLeave(t *testing.T) {
	h, svc, users, _ := newSharingHandler(t)
	cat := mustCategory(t, svc, "u1")
	users.add("member@example.com", "u2", "Member")
	if _, err := svc.InviteShare(t.Context(), "u1", "Owner", cat.ID, "member@example.com", category.PermissionView); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/categories/"+cat.ID+"/shares/u2", nil), auth.User{ID: "u2"}))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("self-leave status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "DELETE", "/api/categories/"+cat.ID+"/shares/u2", rec)
}

func TestCategoryHandlerRevokeRealOwnerRejected(t *testing.T) {
	h, svc, _, _ := newSharingHandler(t)
	cat := mustCategory(t, svc, "u1")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/categories/"+cat.ID+"/shares/u1", nil), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "DELETE", "/api/categories/"+cat.ID+"/shares/u1", rec)
}
