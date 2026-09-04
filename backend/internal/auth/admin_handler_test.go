package auth_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
)

func TestHandlerMeIncludesLanguage(t *testing.T) {
	hr := newHarness(t)
	admin, _ := signInEmail(t, hr.svc, hr.mailer, "admin@example.com")

	rec := hr.do(t, httptest.NewRequest("GET", "/api/auth/me", nil), &admin)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	conforms(t, "GET", "/api/auth/me", rec)

	var body struct {
		Language *string `json:"language"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Language != nil {
		t.Fatalf("language = %v, want nil (no LanguageLookup wired, no preference set)", *body.Language)
	}
}

func TestHandlerListUsersRequiresAdmin(t *testing.T) {
	hr := newHarness(t)
	admin, _ := signInEmail(t, hr.svc, hr.mailer, "admin@example.com")
	member, _ := signInEmail(t, hr.svc, hr.mailer, "member@example.com")

	// Anonymous -> 401.
	rec := hr.do(t, httptest.NewRequest("GET", "/api/auth/users", nil), nil)
	if rec.Code != 401 {
		t.Fatalf("anonymous status = %d, want 401", rec.Code)
	}
	conforms(t, "GET", "/api/auth/users", rec)

	// Non-admin -> 403.
	rec = hr.do(t, httptest.NewRequest("GET", "/api/auth/users", nil), &member)
	if rec.Code != 403 {
		t.Fatalf("non-admin status = %d, want 403", rec.Code)
	}
	conforms(t, "GET", "/api/auth/users", rec)

	// Admin -> 200 with both users.
	rec = hr.do(t, httptest.NewRequest("GET", "/api/auth/users", nil), &admin)
	if rec.Code != 200 {
		t.Fatalf("admin status = %d, want 200", rec.Code)
	}
	conforms(t, "GET", "/api/auth/users", rec)

	var users []auth.AdminUser
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
}

func TestHandlerDisableEnableDeleteUser(t *testing.T) {
	hr := newHarness(t)
	admin, _ := signInEmail(t, hr.svc, hr.mailer, "admin@example.com")
	member, memberTok := signInEmail(t, hr.svc, hr.mailer, "member@example.com")

	// Disable.
	rec := hr.do(t, httptest.NewRequest("POST", "/api/auth/users/"+member.ID+"/disable", nil), &admin)
	if rec.Code != 200 {
		t.Fatalf("disable status = %d, want 200", rec.Code)
	}
	conforms(t, "POST", "/api/auth/users/"+member.ID+"/disable", rec)

	// The disabled member's own session is now dead.
	meReq := httptest.NewRequest("GET", "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+memberTok)
	if _, err := hr.svc.Authenticate(meReq.Context(), memberTok); err == nil {
		t.Fatal("disabled member's session should no longer authenticate")
	}

	// Enable.
	rec = hr.do(t, httptest.NewRequest("POST", "/api/auth/users/"+member.ID+"/enable", nil), &admin)
	if rec.Code != 200 {
		t.Fatalf("enable status = %d, want 200", rec.Code)
	}
	conforms(t, "POST", "/api/auth/users/"+member.ID+"/enable", rec)

	// Delete.
	rec = hr.do(t, httptest.NewRequest("DELETE", "/api/auth/users/"+member.ID, nil), &admin)
	if rec.Code != 204 {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	conforms(t, "DELETE", "/api/auth/users/"+member.ID, rec)

	rec = hr.do(t, httptest.NewRequest("GET", "/api/auth/users", nil), &admin)
	var users []auth.AdminUser
	_ = json.Unmarshal(rec.Body.Bytes(), &users)
	for _, u := range users {
		if u.ID == member.ID {
			t.Fatal("soft-deleted member still listed after delete")
		}
	}
}

func TestHandlerAdminMayDisableThemselvesAsLastAdmin(t *testing.T) {
	hr := newHarness(t)
	admin, _ := signInEmail(t, hr.svc, hr.mailer, "admin@example.com")

	rec := hr.do(t, httptest.NewRequest("POST", "/api/auth/users/"+admin.ID+"/disable", nil), &admin)
	if rec.Code != 200 {
		t.Fatalf("self-disable status = %d, want 200 (no server-side guard)", rec.Code)
	}
	conforms(t, "POST", "/api/auth/users/"+admin.ID+"/disable", rec)
}

func TestHandlerListInvitesRequiresAdmin(t *testing.T) {
	hr := newHarness(t)
	admin, _ := signInEmail(t, hr.svc, hr.mailer, "admin@example.com")
	member, _ := signInEmail(t, hr.svc, hr.mailer, "member@example.com")

	if _, err := hr.svc.CreateInvite(httptest.NewRequest("GET", "/", nil).Context(), admin.ID, "invitee@example.com"); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	rec := hr.do(t, httptest.NewRequest("GET", "/api/auth/invites", nil), &member)
	if rec.Code != 403 {
		t.Fatalf("non-admin status = %d, want 403", rec.Code)
	}
	conforms(t, "GET", "/api/auth/invites", rec)

	rec = hr.do(t, httptest.NewRequest("GET", "/api/auth/invites", nil), &admin)
	if rec.Code != 200 {
		t.Fatalf("admin status = %d, want 200", rec.Code)
	}
	conforms(t, "GET", "/api/auth/invites", rec)

	var invites []auth.InviteInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &invites); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(invites) != 1 || invites[0].InvitedBy.Email != admin.Email {
		t.Fatalf("invites = %+v", invites)
	}
}

func TestHandlerCreateInviteResponseIncludesInviter(t *testing.T) {
	hr := newHarness(t)
	admin, _ := signInEmail(t, hr.svc, hr.mailer, "admin@example.com")

	req := httptest.NewRequest("POST", "/api/auth/invites", strings.NewReader(`{"email":"invitee@example.com"}`))
	rec := hr.do(t, req, &admin)
	if rec.Code != 201 {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	conforms(t, "POST", "/api/auth/invites", rec)

	var inv auth.InviteInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &inv); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if inv.InvitedBy.Email != admin.Email {
		t.Fatalf("InvitedBy.Email = %q, want %q", inv.InvitedBy.Email, admin.Email)
	}
}
