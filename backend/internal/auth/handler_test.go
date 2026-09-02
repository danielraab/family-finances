package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/httpapi"
	"at.draab/familyfinances/internal/storage/memory"
)

type harness struct {
	h      *auth.Handler
	svc    *auth.Service
	store  *memory.AuthStore
	mailer *stubMailer
}

func newHarness(t *testing.T, opts ...svcOpt) *harness {
	t.Helper()
	store := memory.NewAuthStore()
	mailer := &stubMailer{}
	cfg := svcConfig{clock: &clock{now: time.Now()}}
	for _, o := range opts {
		o(&cfg)
	}
	svc := auth.NewService(store, mailer, cfg.oidc, baseParams(), auth.WithClock(cfg.clock.Now))
	h := auth.NewHandler(svc, auth.HandlerOptions{RenderError: httpapi.WriteError, CookieSecure: true})
	return &harness{h: h, svc: svc, store: store, mailer: mailer}
}

func (hr *harness) do(t *testing.T, req *http.Request, user *auth.User) *httptest.ResponseRecorder {
	t.Helper()
	if user != nil {
		req = req.WithContext(auth.WithUser(req.Context(), *user))
	}
	rec := httptest.NewRecorder()
	hr.h.ServeHTTP(rec, req)
	return rec
}

func TestHandlerEmailStartAlways200(t *testing.T) {
	hr := newHarness(t)

	req := httptest.NewRequest("POST", "/api/auth/email/start", strings.NewReader(`{"email":"x@nowhere.test"}`))
	rec := hr.do(t, req, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	// Malformed body is the one case that is not 200.
	bad := httptest.NewRequest("POST", "/api/auth/email/start", strings.NewReader(`not json`))
	if rec := hr.do(t, bad, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad body status = %d, want 400", rec.Code)
	}
}

func TestHandlerEmailCallbackBrowserSetsCookieAndRedirects(t *testing.T) {
	hr := newHarness(t)
	if err := hr.svc.StartEmailLogin(context.Background(), "browser@example.com"); err != nil {
		t.Fatal(err)
	}
	m, _ := hr.mailer.last()
	tok := tokenFromLink(t, m.link)

	req := httptest.NewRequest("GET", "/api/auth/email/callback?token="+tok, nil)
	rec := hr.do(t, req, nil)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	cookie := rec.Result().Cookies()
	if len(cookie) == 0 || cookie[0].Name != auth.CookieName {
		t.Fatalf("no %s cookie set: %+v", auth.CookieName, cookie)
	}
	c := cookie[0]
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || !c.Secure || c.Path != "/" {
		t.Fatalf("cookie attrs wrong: %+v", c)
	}
}

func TestHandlerEmailCallbackJSONClientGetsToken(t *testing.T) {
	hr := newHarness(t)
	if err := hr.svc.StartEmailLogin(context.Background(), "api@example.com"); err != nil {
		t.Fatal(err)
	}
	m, _ := hr.mailer.last()

	req := httptest.NewRequest("GET", "/api/auth/email/callback?token="+tokenFromLink(t, m.link), nil)
	req.Header.Set("Accept", "application/json")
	rec := hr.do(t, req, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("JSON client should not get a cookie")
	}
	var body struct {
		SessionToken string    `json:"session_token"`
		User         auth.User `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.SessionToken == "" || body.User.Email != "api@example.com" {
		t.Fatalf("body = %+v", body)
	}
}

func TestHandlerEmailCallbackBadToken(t *testing.T) {
	hr := newHarness(t)
	req := httptest.NewRequest("GET", "/api/auth/email/callback?token=nope", nil)
	req.Header.Set("Accept", "application/json")
	rec := hr.do(t, req, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (ErrTokenInvalid)", rec.Code)
	}
}

func TestHandlerMe(t *testing.T) {
	hr := newHarness(t)

	if rec := hr.do(t, httptest.NewRequest("GET", "/api/auth/me", nil), nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anon status = %d, want 401", rec.Code)
	}

	user := auth.User{ID: "u1", Email: "me@example.com"}
	rec := hr.do(t, httptest.NewRequest("GET", "/api/auth/me", nil), &user)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got auth.User
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Email != "me@example.com" {
		t.Fatalf("body = %s err %v", rec.Body.String(), err)
	}
}

func TestHandlerLogout(t *testing.T) {
	hr := newHarness(t)
	user, tok := signInEmail(t, hr.svc, hr.mailer, "logmeout@example.com")

	if rec := hr.do(t, httptest.NewRequest("POST", "/api/auth/logout", nil), nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anon logout = %d, want 401", rec.Code)
	}

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := hr.do(t, req, &user)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if _, err := hr.svc.Authenticate(context.Background(), tok); err == nil {
		t.Fatal("session still valid after logout")
	}
}

func TestHandlerCreateInvite(t *testing.T) {
	hr := newHarness(t)
	inviter, _ := signInEmail(t, hr.svc, hr.mailer, "host@example.com")

	if rec := hr.do(t, httptest.NewRequest("POST", "/api/auth/invites", strings.NewReader(`{"email":"g@example.com"}`)), nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anon invite = %d, want 401", rec.Code)
	}

	req := httptest.NewRequest("POST", "/api/auth/invites", strings.NewReader(`{"email":"guest@example.com"}`))
	rec := hr.do(t, req, &inviter)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(strings.ToLower(rec.Body.String()), "token") {
		t.Fatalf("invite response leaked a token: %s", rec.Body.String())
	}
	if m, ok := hr.mailer.last(); !ok || m.kind != "invite" || m.addr != "guest@example.com" {
		t.Fatalf("invite mail not sent: %+v", m)
	}
}

func TestHandlerOIDCStartNotConfigured(t *testing.T) {
	hr := newHarness(t) // no OIDC stub
	rec := hr.do(t, httptest.NewRequest("GET", "/api/auth/oidc/start", nil), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (ErrOIDCNotConfigured)", rec.Code)
	}
}

func TestHandlerOIDCFlow(t *testing.T) {
	oidc := &stubOIDC{claims: auth.OIDCClaims{Issuer: "https://idp.example", Subject: "s-1", Email: "oidc@example.com", EmailVerified: true}}
	hr := newHarness(t, withOIDC(oidc))

	start := hr.do(t, httptest.NewRequest("GET", "/api/auth/oidc/start", nil), nil)
	if start.Code != http.StatusFound {
		t.Fatalf("start status = %d, want 302", start.Code)
	}
	loc := start.Header().Get("Location")
	state := loc[strings.Index(loc, "state=")+len("state="):]

	cb := httptest.NewRequest("GET", "/api/auth/oidc/callback?state="+state+"&code=abc", nil)
	cb.Header.Set("Accept", "application/json")
	rec := hr.do(t, cb, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("callback status = %d, want 200; body %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerInviteAccept(t *testing.T) {
	hr := newHarness(t)
	inviter, _ := signInEmail(t, hr.svc, hr.mailer, "host@example.com")
	if _, err := hr.svc.CreateInvite(context.Background(), inviter.ID, "invited@example.com"); err != nil {
		t.Fatal(err)
	}
	m, _ := hr.mailer.last()

	req := httptest.NewRequest("GET", "/api/auth/invites/accept?token="+tokenFromLink(t, m.link), nil)
	req.Header.Set("Accept", "application/json")
	rec := hr.do(t, req, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body.String())
	}

	// Second use of the same link is rejected.
	req2 := httptest.NewRequest("GET", "/api/auth/invites/accept?token="+tokenFromLink(t, m.link), nil)
	req2.Header.Set("Accept", "application/json")
	if rec := hr.do(t, req2, nil); rec.Code == http.StatusOK {
		t.Fatal("invite link reused successfully; want rejection")
	}
}
