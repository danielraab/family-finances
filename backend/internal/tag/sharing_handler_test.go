package tag_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/httpapi"
	"at.draab/familyfinances/internal/storage/memory"
	"at.draab/familyfinances/internal/tag"
)

// newSharingHandler wires a handler over a service that has a UserLookup/
// Mailer configured, unlike newHandler's bare fixture — needed for any test
// exercising POST .../shares (email lookup).
func newSharingHandler(t *testing.T) (http.Handler, *tag.Service, *fakeUsers, *fakeMailer) {
	t.Helper()
	store := memory.NewTagStore()
	users := newFakeUsers()
	mailer := &fakeMailer{}
	svc := tag.NewService(store, tag.WithMailer(mailer), tag.WithBaseURL("https://app.example"))
	svc.SetUserLookup(users)
	h := tag.NewHandler(svc, tag.HandlerOptions{RenderError: httpapi.WriteError})
	return h, svc, users, mailer
}

func mustTag(t *testing.T, svc *tag.Service, ownerID string) tag.Tag {
	t.Helper()
	tg, err := svc.Create(t.Context(), ownerID, "groceries")
	if err != nil {
		t.Fatal(err)
	}
	return tg
}

func TestTagHandlerListSharesEmpty(t *testing.T) {
	h, svc, _, _ := newSharingHandler(t)
	tg := mustTag(t, svc, "u1")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/tags/"+tg.ID+"/shares", nil), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/tags/"+tg.ID+"/shares", rec)
	var shares []tag.TagShare
	if err := json.Unmarshal(rec.Body.Bytes(), &shares); err != nil {
		t.Fatal(err)
	}
	if len(shares) != 0 {
		t.Fatalf("shares = %+v, want empty", shares)
	}
}

func TestTagHandlerListSharesNoPermissionIsNotFound(t *testing.T) {
	h, svc, _, _ := newSharingHandler(t)
	tg := mustTag(t, svc, "u1")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/tags/"+tg.ID+"/shares", nil), auth.User{ID: "stranger"}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	conforms(t, "GET", "/api/tags/"+tg.ID+"/shares", rec)
}

func TestTagHandlerGetIncludesSharedTag(t *testing.T) {
	h, svc, users, _ := newSharingHandler(t)
	tg := mustTag(t, svc, "u1")
	users.add("member@example.com", "u2", "Member")
	if _, err := svc.InviteShare(t.Context(), "u1", "Owner", tg.ID, "member@example.com", tag.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/tags/"+tg.ID, nil), auth.User{ID: "u2"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/tags/"+tg.ID, rec)
	var got tag.Tag
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Permission != tag.PermissionAppend || !got.Shared {
		t.Fatalf("got = %+v", got)
	}
}

func TestTagHandlerInviteShareMatched(t *testing.T) {
	h, svc, users, mailer := newSharingHandler(t)
	tg := mustTag(t, svc, "u1")
	users.add("member@example.com", "u2", "Member")
	user := auth.User{ID: "u1", DisplayName: "Owner"}

	body := `{"email":"member@example.com","permission":"append"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/tags/"+tg.ID+"/shares", strings.NewReader(body)), user))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/tags/"+tg.ID+"/shares", rec)
	var result struct {
		Matched bool          `json:"matched"`
		Share   *tag.TagShare `json:"share"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Matched || result.Share == nil || result.Share.Permission != tag.PermissionAppend {
		t.Fatalf("result = %+v", result)
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("mailer.sent = %+v, want one notification", mailer.sent)
	}
}

func TestTagHandlerInviteShareUnmatched(t *testing.T) {
	h, svc, _, _ := newSharingHandler(t)
	tg := mustTag(t, svc, "u1")
	user := auth.User{ID: "u1"}

	body := `{"email":"nobody@example.com","permission":"view"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/tags/"+tg.ID+"/shares", strings.NewReader(body)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s, want 200 for an unmatched email", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/tags/"+tg.ID+"/shares", rec)
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

func TestTagHandlerInviteShareForbiddenForNonOwner(t *testing.T) {
	h, svc, users, _ := newSharingHandler(t)
	tg := mustTag(t, svc, "u1")
	users.add("appender@example.com", "u2", "Appender")
	users.add("target@example.com", "u3", "Target")

	if _, err := svc.InviteShare(t.Context(), "u1", "Owner", tg.ID, "appender@example.com", tag.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	body := `{"email":"target@example.com","permission":"view"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("POST", "/api/tags/"+tg.ID+"/shares", strings.NewReader(body)), auth.User{ID: "u2"}))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "POST", "/api/tags/"+tg.ID+"/shares", rec)
}

func TestTagHandlerUpdateAndRevokeShare(t *testing.T) {
	h, svc, users, _ := newSharingHandler(t)
	tg := mustTag(t, svc, "u1")
	users.add("member@example.com", "u2", "Member")
	user := auth.User{ID: "u1"}
	if _, err := svc.InviteShare(t.Context(), "u1", "Owner", tg.ID, "member@example.com", tag.PermissionView); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("PATCH", "/api/tags/"+tg.ID+"/shares/u2", strings.NewReader(`{"permission":"append"}`)), user))
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "PATCH", "/api/tags/"+tg.ID+"/shares/u2", rec)
	var updated tag.TagShare
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Permission != tag.PermissionAppend {
		t.Fatalf("updated = %+v", updated)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/tags/"+tg.ID+"/shares/u2", nil), user))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "DELETE", "/api/tags/"+tg.ID+"/shares/u2", rec)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("GET", "/api/tags/"+tg.ID, nil), auth.User{ID: "u2"}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status after revoke = %d, want 404", rec.Code)
	}
}

func TestTagHandlerSelfLeave(t *testing.T) {
	h, svc, users, _ := newSharingHandler(t)
	tg := mustTag(t, svc, "u1")
	users.add("member@example.com", "u2", "Member")
	if _, err := svc.InviteShare(t.Context(), "u1", "Owner", tg.ID, "member@example.com", tag.PermissionView); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/tags/"+tg.ID+"/shares/u2", nil), auth.User{ID: "u2"}))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("self-leave status = %d, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "DELETE", "/api/tags/"+tg.ID+"/shares/u2", rec)
}

func TestTagHandlerRevokeRealOwnerRejected(t *testing.T) {
	h, svc, _, _ := newSharingHandler(t)
	tg := mustTag(t, svc, "u1")
	user := auth.User{ID: "u1"}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/tags/"+tg.ID+"/shares/u1", nil), user))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "DELETE", "/api/tags/"+tg.ID+"/shares/u1", rec)
}

func TestTagHandlerDeleteBlockedWhileShared(t *testing.T) {
	h, svc, users, _ := newSharingHandler(t)
	tg := mustTag(t, svc, "u1")
	users.add("member@example.com", "u2", "Member")
	if _, err := svc.InviteShare(t.Context(), "u1", "Owner", tg.ID, "member@example.com", tag.PermissionView); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, withUser(httptest.NewRequest("DELETE", "/api/tags/"+tg.ID, nil), auth.User{ID: "u1"}))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409, body = %s", rec.Code, rec.Body)
	}
	conforms(t, "DELETE", "/api/tags/"+tg.ID, rec)
}
