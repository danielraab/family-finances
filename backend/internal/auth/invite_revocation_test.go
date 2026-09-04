package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"at.draab/familyfinances/internal/auth"
)

// createInvite is a small test helper: goes through the service directly
// rather than the HTTP handler, mirroring TestHandlerInviteAccept.
func createInvite(t *testing.T, hr *harness, inviterID, email string) auth.Invite {
	t.Helper()
	inv, err := hr.svc.CreateInvite(context.Background(), inviterID, email)
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	return inv
}

func decodeInviteInfo(t *testing.T, rec *httptest.ResponseRecorder) auth.InviteInfo {
	t.Helper()
	var inv auth.InviteInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &inv); err != nil {
		t.Fatalf("decode InviteInfo: %v; body %s", err, rec.Body.String())
	}
	return inv
}

func TestHandlerRevokeInviteByInviter(t *testing.T) {
	hr := newHarness(t)
	_, _ = signInEmail(t, hr.svc, hr.mailer, "admin@example.com") // bootstrap admin, unused here
	inviter, _ := signInEmail(t, hr.svc, hr.mailer, "inviter@example.com")
	inv := createInvite(t, hr, inviter.ID, "guest@example.com")
	revokePath := "/api/auth/invites/" + inv.ID + "/revoke"

	rec := hr.do(t, httptest.NewRequest("POST", revokePath, nil), &inviter)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body.String())
	}
	conforms(t, "POST", revokePath, rec)

	got := decodeInviteInfo(t, rec)
	if got.RevokedAt == nil {
		t.Fatal("revoked_at not set")
	}
}

func TestHandlerRevokeInviteByAdminNotInviter(t *testing.T) {
	hr := newHarness(t)
	admin, _ := signInEmail(t, hr.svc, hr.mailer, "admin@example.com")
	inviter, _ := signInEmail(t, hr.svc, hr.mailer, "inviter@example.com")
	inv := createInvite(t, hr, inviter.ID, "guest@example.com")
	revokePath := "/api/auth/invites/" + inv.ID + "/revoke"

	rec := hr.do(t, httptest.NewRequest("POST", revokePath, nil), &admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body.String())
	}
	conforms(t, "POST", revokePath, rec)
}

func TestHandlerRevokeInviteForbiddenForThirdParty(t *testing.T) {
	hr := newHarness(t)
	_, _ = signInEmail(t, hr.svc, hr.mailer, "admin@example.com") // bootstrap admin, unused here
	inviter, _ := signInEmail(t, hr.svc, hr.mailer, "inviter@example.com")
	thirdParty, _ := signInEmail(t, hr.svc, hr.mailer, "thirdparty@example.com")
	inv := createInvite(t, hr, inviter.ID, "guest@example.com")
	revokePath := "/api/auth/invites/" + inv.ID + "/revoke"

	rec := hr.do(t, httptest.NewRequest("POST", revokePath, nil), &thirdParty)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	conforms(t, "POST", revokePath, rec)
}

func TestHandlerRevokeInviteUnauthenticated(t *testing.T) {
	hr := newHarness(t)
	inviter, _ := signInEmail(t, hr.svc, hr.mailer, "inviter@example.com")
	inv := createInvite(t, hr, inviter.ID, "guest@example.com")
	revokePath := "/api/auth/invites/" + inv.ID + "/revoke"

	rec := hr.do(t, httptest.NewRequest("POST", revokePath, nil), nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	conforms(t, "POST", revokePath, rec)
}

func TestHandlerRevokeInviteNotFound(t *testing.T) {
	hr := newHarness(t)
	admin, _ := signInEmail(t, hr.svc, hr.mailer, "admin@example.com")
	revokePath := "/api/auth/invites/nope/revoke"

	rec := hr.do(t, httptest.NewRequest("POST", revokePath, nil), &admin)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	conforms(t, "POST", revokePath, rec)
}

func TestHandlerRevokeInviteIsIdempotent(t *testing.T) {
	hr := newHarness(t)
	inviter, _ := signInEmail(t, hr.svc, hr.mailer, "inviter@example.com")
	inv := createInvite(t, hr, inviter.ID, "guest@example.com")
	revokePath := "/api/auth/invites/" + inv.ID + "/revoke"

	first := hr.do(t, httptest.NewRequest("POST", revokePath, nil), &inviter)
	if first.Code != http.StatusOK {
		t.Fatalf("first revoke status = %d, want 200", first.Code)
	}
	firstBody := decodeInviteInfo(t, first)

	second := hr.do(t, httptest.NewRequest("POST", revokePath, nil), &inviter)
	if second.Code != http.StatusOK {
		t.Fatalf("second revoke status = %d, want 200", second.Code)
	}
	conforms(t, "POST", revokePath, second)
	secondBody := decodeInviteInfo(t, second)

	if firstBody.RevokedAt == nil || secondBody.RevokedAt == nil {
		t.Fatal("revoked_at not set")
	}
	if !firstBody.RevokedAt.Equal(*secondBody.RevokedAt) {
		t.Fatalf("revoked_at changed on repeat revoke: %v -> %v", firstBody.RevokedAt, secondBody.RevokedAt)
	}
}

func TestHandlerRevokeAcceptedOrExpiredInviteStillSucceeds(t *testing.T) {
	hr := newHarness(t)
	admin, _ := signInEmail(t, hr.svc, hr.mailer, "admin@example.com")
	inviter, _ := signInEmail(t, hr.svc, hr.mailer, "inviter@example.com")
	inv := createInvite(t, hr, inviter.ID, "accepted-guest@example.com")
	revokePath := "/api/auth/invites/" + inv.ID + "/revoke"

	m, _ := hr.mailer.last()
	acceptReq := httptest.NewRequest("GET", "/api/auth/invites/accept?token="+tokenFromLink(t, m.link), nil)
	acceptReq.Header.Set("Accept", "application/json")
	acceptRec := hr.do(t, acceptReq, nil)
	if acceptRec.Code != http.StatusOK {
		t.Fatalf("accept status = %d, want 200; body %s", acceptRec.Code, acceptRec.Body.String())
	}

	rec := hr.do(t, httptest.NewRequest("POST", revokePath, nil), &admin)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke-after-accept status = %d, want 200; body %s", rec.Code, rec.Body.String())
	}
	conforms(t, "POST", revokePath, rec)

	// Still listed, not removed.
	listRec := hr.do(t, httptest.NewRequest("GET", "/api/auth/invites", nil), &admin)
	var invites []auth.InviteInfo
	if err := json.Unmarshal(listRec.Body.Bytes(), &invites); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, i := range invites {
		if i.ID == inv.ID {
			found = true
			if i.RevokedAt == nil || i.AcceptedAt == nil {
				t.Fatalf("expected both accepted_at and revoked_at set: %+v", i)
			}
		}
	}
	if !found {
		t.Fatal("revoked-but-accepted invite disappeared from the listing")
	}
}

func TestHandlerRevokedInviteCannotBeAccepted(t *testing.T) {
	hr := newHarness(t)
	inviter, _ := signInEmail(t, hr.svc, hr.mailer, "inviter@example.com")
	inv := createInvite(t, hr, inviter.ID, "guest@example.com")
	m, _ := hr.mailer.last()
	revokePath := "/api/auth/invites/" + inv.ID + "/revoke"

	rec := hr.do(t, httptest.NewRequest("POST", revokePath, nil), &inviter)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d, want 200", rec.Code)
	}

	acceptReq := httptest.NewRequest("GET", "/api/auth/invites/accept?token="+tokenFromLink(t, m.link), nil)
	acceptReq.Header.Set("Accept", "application/json")
	acceptRec := hr.do(t, acceptReq, nil)
	if acceptRec.Code == http.StatusOK {
		t.Fatal("revoked invite's link should not accept")
	}
}

func TestHandlerDeleteInviteRequiresRevokedFirst(t *testing.T) {
	hr := newHarness(t)
	admin, _ := signInEmail(t, hr.svc, hr.mailer, "admin@example.com")
	inv := createInvite(t, hr, admin.ID, "guest@example.com")
	revokePath := "/api/auth/invites/" + inv.ID + "/revoke"
	deletePath := "/api/auth/invites/" + inv.ID

	// Not revoked yet -> 409.
	rec := hr.do(t, httptest.NewRequest("DELETE", deletePath, nil), &admin)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	conforms(t, "DELETE", deletePath, rec)

	// Revoke, then delete succeeds.
	revokeRec := hr.do(t, httptest.NewRequest("POST", revokePath, nil), &admin)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d, want 200", revokeRec.Code)
	}
	delRec := hr.do(t, httptest.NewRequest("DELETE", deletePath, nil), &admin)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body %s", delRec.Code, delRec.Body.String())
	}
	conforms(t, "DELETE", deletePath, delRec)

	// Excluded from both listings afterward.
	adminList := hr.do(t, httptest.NewRequest("GET", "/api/auth/invites", nil), &admin)
	var invites []auth.InviteInfo
	_ = json.Unmarshal(adminList.Body.Bytes(), &invites)
	for _, i := range invites {
		if i.ID == inv.ID {
			t.Fatal("soft-deleted invite still in GET /api/auth/invites")
		}
	}

	mineList := hr.do(t, httptest.NewRequest("GET", "/api/auth/invites/mine", nil), &admin)
	var mine []auth.InviteInfo
	_ = json.Unmarshal(mineList.Body.Bytes(), &mine)
	for _, i := range mine {
		if i.ID == inv.ID {
			t.Fatal("soft-deleted invite still in GET /api/auth/invites/mine")
		}
	}

	// A second delete of the now-gone invite is a 404, not another 409/204.
	redelRec := hr.do(t, httptest.NewRequest("DELETE", deletePath, nil), &admin)
	if redelRec.Code != http.StatusNotFound {
		t.Fatalf("re-delete status = %d, want 404", redelRec.Code)
	}
	conforms(t, "DELETE", deletePath, redelRec)
}

func TestHandlerDeleteInviteRequiresAdmin(t *testing.T) {
	hr := newHarness(t)
	_, _ = signInEmail(t, hr.svc, hr.mailer, "admin@example.com") // bootstrap admin, unused here
	inviter, _ := signInEmail(t, hr.svc, hr.mailer, "inviter@example.com")
	inv := createInvite(t, hr, inviter.ID, "guest@example.com")
	revokePath := "/api/auth/invites/" + inv.ID + "/revoke"
	deletePath := "/api/auth/invites/" + inv.ID

	revokeRec := hr.do(t, httptest.NewRequest("POST", revokePath, nil), &inviter)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d, want 200", revokeRec.Code)
	}

	rec := hr.do(t, httptest.NewRequest("DELETE", deletePath, nil), &inviter)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 even for the invite's own (non-admin) inviter", rec.Code)
	}
	conforms(t, "DELETE", deletePath, rec)
}

func TestHandlerListMyInvites(t *testing.T) {
	hr := newHarness(t)
	admin, _ := signInEmail(t, hr.svc, hr.mailer, "admin@example.com")
	inviterA, _ := signInEmail(t, hr.svc, hr.mailer, "invitera@example.com")
	inviterB, _ := signInEmail(t, hr.svc, hr.mailer, "inviterb@example.com")
	createInvite(t, hr, inviterA.ID, "a-guest@example.com")
	createInvite(t, hr, inviterB.ID, "b-guest@example.com")

	rec := hr.do(t, httptest.NewRequest("GET", "/api/auth/invites/mine", nil), &inviterA)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	conforms(t, "GET", "/api/auth/invites/mine", rec)

	var mine []auth.InviteInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &mine); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(mine) != 1 || mine[0].Email != "a-guest@example.com" {
		t.Fatalf("mine = %+v, want just inviterA's own invite", mine)
	}

	// An admin with no invites of their own gets an empty (not null) list.
	adminMine := hr.do(t, httptest.NewRequest("GET", "/api/auth/invites/mine", nil), &admin)
	if adminMine.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", adminMine.Code)
	}
	conforms(t, "GET", "/api/auth/invites/mine", adminMine)
	if body := adminMine.Body.String(); body != "[]\n" && body != "[]" {
		t.Fatalf("empty mine list body = %q, want []", body)
	}

	anon := hr.do(t, httptest.NewRequest("GET", "/api/auth/invites/mine", nil), nil)
	if anon.Code != http.StatusUnauthorized {
		t.Fatalf("anon status = %d, want 401", anon.Code)
	}
	conforms(t, "GET", "/api/auth/invites/mine", anon)
}
