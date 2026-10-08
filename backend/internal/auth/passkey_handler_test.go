package auth_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/passkeyauth/passkeytest"
)

// passkeyOrigin is the browser origin matching baseParams().BaseURL.
const passkeyOrigin = "https://ff.example"

// signIn completes a magic-link sign-in for email and returns the new session
// token — a cookie (web) session, or a bearer (api) session when api is set.
func (hr *harness) signIn(t *testing.T, email string, api bool) string {
	t.Helper()
	if err := hr.svc.StartEmailLogin(context.Background(), email); err != nil {
		t.Fatal(err)
	}
	m, _ := hr.mailer.last()
	req := httptest.NewRequest("GET", "/api/auth/email/callback?token="+tokenFromLink(t, m.link), nil)
	if api {
		req.Header.Set("Accept", "application/json")
	}
	rec := hr.do(t, req, nil)
	if api {
		var body struct {
			SessionToken string `json:"session_token"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.SessionToken == "" {
			t.Fatalf("api sign-in: %d %s", rec.Code, rec.Body)
		}
		return body.SessionToken
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			return c.Value
		}
	}
	t.Fatalf("web sign-in set no cookie: %d %s", rec.Code, rec.Body)
	return ""
}

// doToken runs req the way internal/httpapi's middleware would for a request
// carrying token: as a cookie (or a bearer header when bearer is set),
// resolved to its user and session. An unknown token stays anonymous.
func (hr *harness) doToken(t *testing.T, req *http.Request, token string, bearer bool) *httptest.ResponseRecorder {
	t.Helper()
	if bearer {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	}
	if u, sess, err := hr.svc.AuthenticateSession(req.Context(), token); err == nil {
		req = req.WithContext(auth.WithSession(auth.WithUser(req.Context(), u), sess))
	}
	rec := httptest.NewRecorder()
	hr.h.ServeHTTP(rec, req)
	return rec
}

func jsonBody(v any) *strings.Reader {
	b, _ := json.Marshal(v)
	return strings.NewReader(string(b))
}

type ceremony struct {
	CeremonyID string          `json:"ceremony_id"`
	Options    json.RawMessage `json:"options"`
}

func decodeCeremony(t *testing.T, rec *httptest.ResponseRecorder) ceremony {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("start status = %d, body %s", rec.Code, rec.Body)
	}
	var c ceremony
	if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil || c.CeremonyID == "" || len(c.Options) == 0 {
		t.Fatalf("bad ceremony %s: %v", rec.Body, err)
	}
	return c
}

func (hr *harness) registerStart(t *testing.T, token string) ceremony {
	t.Helper()
	rec := hr.doToken(t, httptest.NewRequest("POST", "/api/auth/passkeys/register/start", nil), token, false)
	conforms(t, "POST", "/api/auth/passkeys/register/start", rec)
	return decodeCeremony(t, rec)
}

func (hr *harness) registerFinish(t *testing.T, token string, c ceremony, name string, cred json.RawMessage) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]any{"ceremony_id": c.CeremonyID, "credential": cred}
	if name != "" {
		body["name"] = name
	}
	rec := hr.doToken(t, httptest.NewRequest("POST", "/api/auth/passkeys/register/finish", jsonBody(body)), token, false)
	conforms(t, "POST", "/api/auth/passkeys/register/finish", rec)
	return rec
}

// registerPasskey runs a whole registration on token's session and returns
// the authenticator credential and the stored passkey id.
func (hr *harness) registerPasskey(t *testing.T, a *passkeytest.Authenticator, token, name string) (*passkeytest.Credential, string) {
	t.Helper()
	c := hr.registerStart(t, token)
	resp, cred, err := a.Register(c.Options, 1)
	if err != nil {
		t.Fatal(err)
	}
	rec := hr.registerFinish(t, token, c, name, resp)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register finish status = %d, body %s", rec.Code, rec.Body)
	}
	var info auth.PasskeyInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	return cred, info.ID
}

func (hr *harness) loginStart(t *testing.T) ceremony {
	t.Helper()
	rec := hr.do(t, httptest.NewRequest("POST", "/api/auth/passkeys/login/start", nil), nil)
	conforms(t, "POST", "/api/auth/passkeys/login/start", rec)
	return decodeCeremony(t, rec)
}

func (hr *harness) loginFinish(t *testing.T, c ceremony, resp json.RawMessage, cookie string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/auth/passkeys/login/finish",
		jsonBody(map[string]any{"ceremony_id": c.CeremonyID, "credential": resp}))
	var rec *httptest.ResponseRecorder
	if cookie != "" {
		rec = hr.doToken(t, req, cookie, false)
	} else {
		rec = hr.do(t, req, nil)
	}
	conforms(t, "POST", "/api/auth/passkeys/login/finish", rec)
	return rec
}

// passkeyLogin signs in with cred and returns the new session token.
func (hr *harness) passkeyLogin(t *testing.T, a *passkeytest.Authenticator, cred *passkeytest.Credential) string {
	t.Helper()
	c := hr.loginStart(t)
	resp, err := a.Login(c.Options, cred)
	if err != nil {
		t.Fatal(err)
	}
	rec := hr.loginFinish(t, c, resp, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login finish status = %d, body %s", rec.Code, rec.Body)
	}
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == auth.CookieName && ck.Value != "" {
			return ck.Value
		}
	}
	t.Fatal("passkey login set no session cookie")
	return ""
}

func (hr *harness) listPasskeys(t *testing.T, token string) []auth.PasskeyInfo {
	t.Helper()
	rec := hr.doToken(t, httptest.NewRequest("GET", "/api/auth/passkeys", nil), token, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body %s", rec.Code, rec.Body)
	}
	conforms(t, "GET", "/api/auth/passkeys", rec)
	var list []auth.PasskeyInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	return list
}

func (hr *harness) authenticates(token string) bool {
	_, err := hr.svc.Authenticate(context.Background(), token)
	return err == nil
}

func wantError(t *testing.T, rec *httptest.ResponseRecorder, status int, sentinel error) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, status, rec.Body)
	}
	if sentinel != nil && !strings.Contains(rec.Body.String(), sentinel.Error()) {
		t.Fatalf("body %s does not carry %q", rec.Body, sentinel)
	}
}

// --- registration --------------------------------------------------------

func TestPasskeyRegisterOnFreshWebSession(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	hr.clock.advance(2 * time.Minute)

	a := passkeytest.New(passkeyOrigin)
	c := hr.registerStart(t, tok)
	resp, _, err := a.Register(c.Options, 0)
	if err != nil {
		t.Fatal(err)
	}
	rec := hr.registerFinish(t, tok, c, "  Work laptop  ", resp)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var info auth.PasskeyInfo
	_ = json.Unmarshal(rec.Body.Bytes(), &info)
	if info.Name != "Work laptop" || info.LastUsedAt != nil || info.Current {
		t.Fatalf("created passkey = %+v", info)
	}
	if list := hr.listPasskeys(t, tok); len(list) != 1 || list[0].ID != info.ID {
		t.Fatalf("list = %+v", list)
	}
}

func TestPasskeyRegisterStaleSessionRefused(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	hr.clock.advance(20 * time.Minute)

	rec := hr.doToken(t, httptest.NewRequest("POST", "/api/auth/passkeys/register/start", nil), tok, false)
	wantError(t, rec, http.StatusForbidden, auth.ErrReauthRequired)
	conforms(t, "POST", "/api/auth/passkeys/register/start", rec)
}

func TestPasskeyRegisterGoesStaleMidCeremony(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	hr.clock.advance(4 * time.Minute)
	c := hr.registerStart(t, tok)
	resp, _, _ := passkeytest.New(passkeyOrigin).Register(c.Options, 0)

	hr.clock.advance(2 * time.Minute) // 6 minutes into the session
	wantError(t, hr.registerFinish(t, tok, c, "", resp), http.StatusForbidden, auth.ErrReauthRequired)
	hr.clock.advance(-5 * time.Minute)
	if list := hr.listPasskeys(t, tok); len(list) != 0 {
		t.Fatalf("passkey stored despite stale finish: %+v", list)
	}
}

func TestPasskeyRegisterActivityDoesNotRefreshFreshness(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	for i := 0; i < 12; i++ {
		hr.clock.advance(5 * time.Minute)
		hr.doToken(t, httptest.NewRequest("GET", "/api/auth/me", nil), tok, false)
	}
	rec := hr.doToken(t, httptest.NewRequest("POST", "/api/auth/passkeys/register/start", nil), tok, false)
	wantError(t, rec, http.StatusForbidden, auth.ErrReauthRequired)
}

func TestPasskeyRegisterBearerSessionRefused(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", true)

	rec := hr.doToken(t, httptest.NewRequest("POST", "/api/auth/passkeys/register/start", nil), tok, true)
	wantError(t, rec, http.StatusForbidden, auth.ErrPasskeyWebOnly)

	finish := httptest.NewRequest("POST", "/api/auth/passkeys/register/finish",
		jsonBody(map[string]any{"ceremony_id": "x", "credential": map[string]any{}}))
	wantError(t, hr.doToken(t, finish, tok, true), http.StatusForbidden, auth.ErrPasskeyWebOnly)
}

func TestPasskeyRegisterUnauthenticated(t *testing.T) {
	hr := newHarness(t)
	for _, path := range []string{"/api/auth/passkeys/register/start", "/api/auth/passkeys/register/finish"} {
		rec := hr.do(t, httptest.NewRequest("POST", path, strings.NewReader(`{}`)), nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s status = %d, want 401", path, rec.Code)
		}
	}
	for _, r := range []*http.Request{
		httptest.NewRequest("GET", "/api/auth/passkeys", nil),
		httptest.NewRequest("DELETE", "/api/auth/passkeys/pk1", nil),
	} {
		if rec := hr.do(t, r, nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", r.Method, r.URL.Path, rec.Code)
		}
	}
}

func TestPasskeyRegisterNames(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)

	_, id := hr.registerPasskey(t, a, tok, "   ")
	list := hr.listPasskeys(t, tok)
	if len(list) != 1 || list[0].ID != id || list[0].Name != "Passkey" {
		t.Fatalf("default name list = %+v", list)
	}

	c := hr.registerStart(t, tok)
	resp, _, _ := a.Register(c.Options, 0)
	wantError(t, hr.registerFinish(t, tok, c, strings.Repeat("x", 101), resp), http.StatusBadRequest, auth.ErrInvalidPasskeyName)
	if n := len(hr.listPasskeys(t, tok)); n != 1 {
		t.Fatalf("over-long name stored a passkey: %d passkeys", n)
	}
}

func TestPasskeyRegisterCeremonyBoundToSession(t *testing.T) {
	hr := newHarness(t)
	tok1 := hr.signIn(t, "alice@example.com", false)
	tok2 := hr.signIn(t, "alice@example.com", false)
	c := hr.registerStart(t, tok1)
	resp, _, _ := passkeytest.New(passkeyOrigin).Register(c.Options, 0)

	wantError(t, hr.registerFinish(t, tok2, c, "", resp), http.StatusBadRequest, auth.ErrCeremonyInvalid)
	if n := len(hr.listPasskeys(t, tok1)); n != 0 {
		t.Fatalf("passkey stored from a foreign session's ceremony: %d", n)
	}
}

func TestPasskeyRegisterCeremonySingleUseAndShortLived(t *testing.T) {
	hr := newHarness(t, withReauthWindow(time.Hour))
	tok := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)

	c := hr.registerStart(t, tok)
	resp, _, _ := a.Register(c.Options, 0)
	if rec := hr.registerFinish(t, tok, c, "", resp); rec.Code != http.StatusCreated {
		t.Fatalf("first finish status = %d", rec.Code)
	}
	wantError(t, hr.registerFinish(t, tok, c, "", resp), http.StatusBadRequest, auth.ErrCeremonyInvalid)

	c = hr.registerStart(t, tok)
	resp, _, _ = a.Register(c.Options, 0)
	hr.clock.advance(6 * time.Minute) // past the challenge TTL, inside the 1h window
	wantError(t, hr.registerFinish(t, tok, c, "", resp), http.StatusBadRequest, auth.ErrCeremonyInvalid)
	if n := len(hr.listPasskeys(t, tok)); n != 1 {
		t.Fatalf("passkeys = %d, want only the first", n)
	}
}

func TestPasskeyRegisterRejectsWrongOriginAndMissingUV(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)

	c := hr.registerStart(t, tok)
	resp, _, _ := passkeytest.New("https://evil.example").Register(c.Options, 0)
	wantError(t, hr.registerFinish(t, tok, c, "", resp), http.StatusBadRequest, auth.ErrPasskeyInvalid)

	noUV := passkeytest.New(passkeyOrigin)
	noUV.SkipUV = true
	c = hr.registerStart(t, tok)
	resp, _, _ = noUV.Register(c.Options, 0)
	wantError(t, hr.registerFinish(t, tok, c, "", resp), http.StatusBadRequest, auth.ErrPasskeyInvalid)

	if n := len(hr.listPasskeys(t, tok)); n != 0 {
		t.Fatalf("rejected registrations stored %d passkeys", n)
	}
}

func TestPasskeyRegisterExcludesExistingCredentials(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	cred, _ := hr.registerPasskey(t, a, tok, "")

	c := hr.registerStart(t, tok)
	if !strings.Contains(string(c.Options), base64.RawURLEncoding.EncodeToString(cred.ID)) {
		t.Fatalf("existing credential missing from excludeCredentials: %s", c.Options)
	}
}

// --- sign-in -------------------------------------------------------------

func TestPasskeyLoginSucceeds(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	cred, id := hr.registerPasskey(t, a, tok, "Laptop")
	usersBefore, _ := hr.store.UserCount(context.Background())

	hr.clock.advance(time.Hour)
	c := hr.loginStart(t)
	if strings.Contains(string(c.Options), `"allowCredentials":[{`) {
		t.Fatalf("login options carry an allow-list: %s", c.Options)
	}
	resp, _ := a.Login(c.Options, cred)
	rec := hr.loginFinish(t, c, resp, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var body struct {
		User auth.User `json:"user"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.User.Email != "alice@example.com" {
		t.Fatalf("user = %+v", body.User)
	}
	var cookie *http.Cookie
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == auth.CookieName {
			cookie = ck
		}
	}
	if cookie == nil || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Fatalf("session cookie = %+v", cookie)
	}

	list := hr.listPasskeys(t, cookie.Value)
	if len(list) != 1 || list[0].ID != id || !list[0].Current || list[0].LastUsedAt == nil ||
		!list[0].LastUsedAt.Equal(hr.clock.Now()) {
		t.Fatalf("after sign-in list = %+v", list)
	}
	if usersAfter, _ := hr.store.UserCount(context.Background()); usersAfter != usersBefore {
		t.Fatalf("passkey sign-in changed the user count: %d → %d", usersBefore, usersAfter)
	}
}

func TestPasskeyLoginWithDeletedPasskeyFails(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	cred, id := hr.registerPasskey(t, a, tok, "")
	if rec := hr.doToken(t, httptest.NewRequest("DELETE", "/api/auth/passkeys/"+id, nil), tok, false); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d", rec.Code)
	}

	c := hr.loginStart(t)
	resp, _ := a.Login(c.Options, cred)
	rec := hr.loginFinish(t, c, resp, "")
	wantError(t, rec, http.StatusUnauthorized, auth.ErrPasskeyAuthFailed)
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("failed sign-in set a cookie")
	}
}

func TestPasskeyLoginMismatchedUserHandleFails(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	bob := hr.signIn(t, "bob@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	cred, _ := hr.registerPasskey(t, a, tok, "")
	bobUser, _ := hr.svc.Authenticate(context.Background(), bob)

	c := hr.loginStart(t)
	resp, _ := a.LoginAs(c.Options, cred, []byte(bobUser.ID))
	wantError(t, hr.loginFinish(t, c, resp, ""), http.StatusUnauthorized, auth.ErrPasskeyAuthFailed)
}

func TestPasskeyLoginBadSignatureFails(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	cred, _ := hr.registerPasskey(t, a, tok, "")
	other, _ := hr.registerPasskey(t, a, tok, "")

	forged := *cred
	forged.Key = other.Key
	c := hr.loginStart(t)
	resp, _ := a.Login(c.Options, &forged)
	wantError(t, hr.loginFinish(t, c, resp, ""), http.StatusUnauthorized, auth.ErrPasskeyAuthFailed)
}

func TestPasskeyLoginCounterRegressionFails(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	cred, id := hr.registerPasskey(t, a, tok, "")
	// The server has seen a higher counter than the device will now report.
	if err := hr.store.UpdatePasskeyUsage(context.Background(), id, 100, true, hr.clock.Now()); err != nil {
		t.Fatal(err)
	}

	c := hr.loginStart(t)
	resp, _ := a.Login(c.Options, cred)
	wantError(t, hr.loginFinish(t, c, resp, ""), http.StatusUnauthorized, auth.ErrPasskeyAuthFailed)
}

func TestPasskeyLoginDisabledAccountRefused(t *testing.T) {
	hr := newHarness(t)
	hr.signIn(t, "admin@example.com", false) // first user is the bootstrap admin
	tok := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	cred, _ := hr.registerPasskey(t, a, tok, "")
	alice, _ := hr.svc.Authenticate(context.Background(), tok)
	if _, err := hr.svc.DisableUser(context.Background(), alice.ID); err != nil {
		t.Fatal(err)
	}

	c := hr.loginStart(t)
	resp, _ := a.Login(c.Options, cred)
	rec := hr.loginFinish(t, c, resp, "")
	wantError(t, rec, http.StatusForbidden, auth.ErrAccountDisabled)
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("refused sign-in set a cookie")
	}
}

func TestPasskeyLoginCeremonySingleUse(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	cred, _ := hr.registerPasskey(t, a, tok, "")

	c := hr.loginStart(t)
	resp, _ := a.Login(c.Options, cred)
	if rec := hr.loginFinish(t, c, resp, ""); rec.Code != http.StatusOK {
		t.Fatalf("first finish = %d", rec.Code)
	}
	wantError(t, hr.loginFinish(t, c, resp, ""), http.StatusBadRequest, auth.ErrCeremonyInvalid)
}

func TestPasskeyLoginReplacesExistingCookieSession(t *testing.T) {
	hr := newHarness(t)
	old := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	cred, _ := hr.registerPasskey(t, a, old, "")
	hr.clock.advance(time.Hour)

	c := hr.loginStart(t)
	resp, _ := a.Login(c.Options, cred)
	rec := hr.loginFinish(t, c, resp, old)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if hr.authenticates(old) {
		t.Fatal("the previous session survived a passkey re-authentication")
	}
}

func TestPasskeySessionCountsAsFresh(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	cred, _ := hr.registerPasskey(t, a, tok, "first")

	hr.clock.advance(time.Hour) // the magic-link session is long stale
	fresh := hr.passkeyLogin(t, a, cred)
	hr.clock.advance(time.Minute)
	hr.registerPasskey(t, a, fresh, "second")
	if n := len(hr.listPasskeys(t, fresh)); n != 2 {
		t.Fatalf("passkeys = %d, want 2", n)
	}
}

// --- listing and deletion ------------------------------------------------

func TestPasskeyListOwnOnlyWithoutKeyMaterial(t *testing.T) {
	hr := newHarness(t)
	alice := hr.signIn(t, "alice@example.com", false)
	bob := hr.signIn(t, "bob@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	hr.registerPasskey(t, a, alice, "alice key")
	hr.registerPasskey(t, a, bob, "bob key")

	rec := hr.doToken(t, httptest.NewRequest("GET", "/api/auth/passkeys", nil), alice, false)
	var raw []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &raw)
	if len(raw) != 1 || raw[0]["name"] != "alice key" {
		t.Fatalf("alice sees %s", rec.Body)
	}
	for _, k := range []string{"public_key", "credential_id", "sign_count", "PublicKey", "CredentialID"} {
		if _, ok := raw[0][k]; ok {
			t.Errorf("list response exposes %s", k)
		}
	}
}

func TestPasskeyListBearerSessionAllowed(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", true)
	rec := hr.doToken(t, httptest.NewRequest("GET", "/api/auth/passkeys", nil), tok, true)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("bearer list = %d %s", rec.Code, rec.Body)
	}
}

func TestPasskeyDeleteEndsOnlyThatPasskeysSessions(t *testing.T) {
	hr := newHarness(t)
	s3 := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	credA, idA := hr.registerPasskey(t, a, s3, "A")
	credB, _ := hr.registerPasskey(t, a, s3, "B")
	s1 := hr.passkeyLogin(t, a, credA)
	s2 := hr.passkeyLogin(t, a, credB)

	hr.clock.advance(3 * time.Hour) // removal needs no fresh session
	rec := hr.doToken(t, httptest.NewRequest("DELETE", "/api/auth/passkeys/"+idA, nil), s3, false)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body %s", rec.Code, rec.Body)
	}
	conforms(t, "DELETE", "/api/auth/passkeys/"+idA, rec)
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			t.Fatal("deleting another passkey cleared the current session's cookie")
		}
	}
	if hr.authenticates(s1) {
		t.Error("S1 (created by the deleted passkey) still authenticates")
	}
	if !hr.authenticates(s2) || !hr.authenticates(s3) {
		t.Error("an unrelated session was revoked")
	}
}

func TestPasskeyDeleteCurrentSignsOut(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	a := passkeytest.New(passkeyOrigin)
	cred, id := hr.registerPasskey(t, a, tok, "")
	current := hr.passkeyLogin(t, a, cred)

	rec := hr.doToken(t, httptest.NewRequest("DELETE", "/api/auth/passkeys/"+id, nil), current, false)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	cleared := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("deleting the current session's passkey did not clear the cookie")
	}
	next := hr.doToken(t, httptest.NewRequest("GET", "/api/auth/me", nil), current, false)
	if next.Code != http.StatusUnauthorized {
		t.Fatalf("next request = %d, want 401", next.Code)
	}
}

func TestPasskeyDeleteForeignOrUnknown(t *testing.T) {
	hr := newHarness(t)
	alice := hr.signIn(t, "alice@example.com", false)
	bob := hr.signIn(t, "bob@example.com", false)
	_, id := hr.registerPasskey(t, passkeytest.New(passkeyOrigin), alice, "")

	rec := hr.doToken(t, httptest.NewRequest("DELETE", "/api/auth/passkeys/"+id, nil), bob, false)
	wantError(t, rec, http.StatusNotFound, nil)
	conforms(t, "DELETE", "/api/auth/passkeys/"+id, rec)
	if n := len(hr.listPasskeys(t, alice)); n != 1 {
		t.Fatal("bob deleted alice's passkey")
	}
	wantError(t, hr.doToken(t, httptest.NewRequest("DELETE", "/api/auth/passkeys/nope", nil), alice, false), http.StatusNotFound, nil)
}

// --- session freshness on /me --------------------------------------------

func TestMeReportsSessionFreshness(t *testing.T) {
	hr := newHarness(t)
	web := hr.signIn(t, "alice@example.com", false)
	created := hr.clock.Now()
	api := hr.signIn(t, "alice@example.com", true)

	type meBody struct {
		Session *struct {
			CreatedAt                time.Time  `json:"created_at"`
			Client                   string     `json:"client"`
			PasskeyRegistrationUntil *time.Time `json:"passkey_registration_until"`
		} `json:"session"`
	}

	rec := hr.doToken(t, httptest.NewRequest("GET", "/api/auth/me", nil), web, false)
	conforms(t, "GET", "/api/auth/me", rec)
	var body meBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Session == nil || body.Session.Client != "web" || !body.Session.CreatedAt.Equal(created) ||
		body.Session.PasskeyRegistrationUntil == nil || !body.Session.PasskeyRegistrationUntil.Equal(created.Add(5*time.Minute)) {
		t.Fatalf("web me session = %s", rec.Body)
	}

	rec = hr.doToken(t, httptest.NewRequest("GET", "/api/auth/me", nil), api, true)
	conforms(t, "GET", "/api/auth/me", rec)
	body = meBody{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Session == nil || body.Session.Client != "api" || body.Session.PasskeyRegistrationUntil != nil {
		t.Fatalf("api me session = %s", rec.Body)
	}
}
