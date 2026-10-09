package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/passkeyauth/passkeytest"
	"at.draab/familyfinances/internal/ratelimit"
	"at.draab/familyfinances/internal/storage/memory"
)

func (m *stubMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sent)
}

// fromIP runs an anonymous request as if from ip (the direct peer).
func (hr *harness) fromIP(t *testing.T, req *http.Request, ip string) *httptest.ResponseRecorder {
	t.Helper()
	req.RemoteAddr = ip + ":40000"
	return hr.do(t, req, nil)
}

func emailStart(addr string) *http.Request {
	return httptest.NewRequest("POST", "/api/auth/email/start", strings.NewReader(`{"email":"`+addr+`"}`))
}

// --- per-IP --------------------------------------------------------------

func TestIPLimitRefusesOverBudget(t *testing.T) {
	hr := newHarness(t, withIPLimit(3))
	for i := 0; i < 3; i++ {
		if rec := hr.fromIP(t, emailStart("a@example.com"), "203.0.113.7"); rec.Code != http.StatusOK {
			t.Fatalf("request %d = %d", i+1, rec.Code)
		}
	}
	sent := hr.mailer.count()

	rec := hr.fromIP(t, emailStart("a@example.com"), "203.0.113.7")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("over-budget status = %d, want 429", rec.Code)
	}
	if ra, err := strconv.Atoi(rec.Header().Get("Retry-After")); err != nil || ra < 1 || ra > 60 {
		t.Fatalf("Retry-After = %q, want 1..60 seconds", rec.Header().Get("Retry-After"))
	}
	conforms(t, "POST", "/api/auth/email/start", rec)
	if hr.mailer.count() != sent {
		t.Fatal("a refused request still sent a mail")
	}
}

func TestIPLimitRefusalIsLoggedWithTheClientIP(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	hr := newHarness(t, withIPLimit(1), withTrustedProxies("10.0.0.0/8"))
	viaProxy := func() *httptest.ResponseRecorder {
		req := emailStart("a@example.com")
		req.Header.Set("X-Forwarded-For", "198.51.100.1")
		return hr.fromIP(t, req, "10.0.0.2")
	}
	viaProxy()
	if strings.Contains(buf.String(), "client IP rate limit") {
		t.Fatal("an allowed request was logged as refused")
	}
	if rec := viaProxy(); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}

	var entry map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if strings.Contains(line, "client IP rate limit") {
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				t.Fatal(err)
			}
		}
	}
	if entry == nil {
		t.Fatalf("no refusal log line in %q", buf.String())
	}
	if entry["ip"] != "198.51.100.1" || entry["path"] != "/api/auth/email/start" ||
		entry["method"] != "POST" || entry["retry_after_s"] == nil {
		t.Fatalf("log entry = %v, want the forwarded client IP, method, path and retry_after_s", entry)
	}
}

func TestIPLimitRecoversAfterWindow(t *testing.T) {
	hr := newHarness(t, withIPLimit(1))
	hr.fromIP(t, emailStart("a@example.com"), "203.0.113.7")
	if rec := hr.fromIP(t, emailStart("a@example.com"), "203.0.113.7"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request = %d, want 429", rec.Code)
	}
	hr.clock.advance(61 * time.Second)
	if rec := hr.fromIP(t, emailStart("a@example.com"), "203.0.113.7"); rec.Code != http.StatusOK {
		t.Fatalf("after the window = %d, want 200", rec.Code)
	}
}

func TestIPLimitIsSharedAcrossSignInRoutes(t *testing.T) {
	hr := newHarness(t, withIPLimit(4))
	ip := "203.0.113.7"
	hr.fromIP(t, emailStart("a@example.com"), ip)
	hr.fromIP(t, httptest.NewRequest("GET", "/api/auth/email/callback?token=nope", nil), ip)
	hr.fromIP(t, httptest.NewRequest("GET", "/api/auth/invites/accept?token=nope", nil), ip)
	hr.fromIP(t, httptest.NewRequest("POST", "/api/auth/passkeys/login/start", nil), ip)

	for _, req := range []*http.Request{
		httptest.NewRequest("POST", "/api/auth/passkeys/login/start", nil),
		httptest.NewRequest("POST", "/api/auth/passkeys/login/finish", strings.NewReader(`{}`)),
		httptest.NewRequest("GET", "/api/auth/oidc/start", nil),
		httptest.NewRequest("GET", "/api/auth/oidc/callback?state=x&code=y", nil),
		httptest.NewRequest("GET", "/api/auth/email/callback?token=nope", nil),
		httptest.NewRequest("GET", "/api/auth/invites/accept?token=nope", nil),
		emailStart("a@example.com"),
	} {
		if rec := hr.fromIP(t, req, ip); rec.Code != http.StatusTooManyRequests {
			t.Errorf("%s %s = %d, want 429 once the shared budget is spent", req.Method, req.URL.Path, rec.Code)
		}
	}
}

func TestIPLimitThrottledPasskeyStartStoresNoChallenge(t *testing.T) {
	hr := newHarness(t, withIPLimit(1))
	ip := "203.0.113.7"
	hr.fromIP(t, httptest.NewRequest("POST", "/api/auth/passkeys/login/start", nil), ip)
	rec := hr.fromIP(t, httptest.NewRequest("POST", "/api/auth/passkeys/login/start", nil), ip)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	conforms(t, "POST", "/api/auth/passkeys/login/start", rec)
	if strings.Contains(rec.Body.String(), "ceremony_id") {
		t.Fatal("a throttled start still issued a ceremony")
	}
}

func TestIPLimitOtherClientsUnaffected(t *testing.T) {
	hr := newHarness(t, withIPLimit(1))
	hr.fromIP(t, emailStart("a@example.com"), "203.0.113.7")
	hr.fromIP(t, emailStart("a@example.com"), "203.0.113.7")
	if rec := hr.fromIP(t, emailStart("b@example.com"), "198.51.100.9"); rec.Code != http.StatusOK {
		t.Fatalf("other IP = %d, want 200", rec.Code)
	}
}

func TestIPLimitDoesNotTouchAuthenticatedRoutes(t *testing.T) {
	hr := newHarness(t, withIPLimit(1))
	tok := hr.signIn(t, "alice@example.com", false) // spends the budget of 192.0.2.1 (httptest's default peer)
	for i := 0; i < 5; i++ {
		for _, req := range []*http.Request{
			httptest.NewRequest("GET", "/api/auth/me", nil),
			httptest.NewRequest("GET", "/api/auth/passkeys", nil),
		} {
			if rec := hr.doToken(t, req, tok, false); rec.Code != http.StatusOK {
				t.Fatalf("%s = %d, want 200", req.URL.Path, rec.Code)
			}
		}
	}
}

func TestNoIPLimiterNeverRefuses(t *testing.T) {
	hr := newHarness(t) // no IP limiter, as with RATE_LIMIT_IP_ENABLED=false
	for i := 0; i < 50; i++ {
		if rec := hr.fromIP(t, httptest.NewRequest("POST", "/api/auth/passkeys/login/start", nil), "203.0.113.7"); rec.Code == http.StatusTooManyRequests {
			t.Fatal("429 without an IP limiter")
		}
	}
}

func TestIPLimitKeysOnTheForwardedClient(t *testing.T) {
	hr := newHarness(t, withIPLimit(1), withTrustedProxies("10.0.0.0/8"))
	viaProxy := func(client string) *httptest.ResponseRecorder {
		req := emailStart("a@example.com")
		req.Header.Set("X-Forwarded-For", client)
		return hr.fromIP(t, req, "10.0.0.2")
	}
	viaProxy("198.51.100.1")
	if rec := viaProxy("198.51.100.2"); rec.Code != http.StatusOK {
		t.Fatalf("second client behind the same proxy = %d, want 200 (own budget)", rec.Code)
	}
	if rec := viaProxy("198.51.100.1"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("repeat client = %d, want 429", rec.Code)
	}
}

// --- client IP on sessions -------------------------------------------------

func TestSessionRecordsForwardedClientIP(t *testing.T) {
	hr := newHarness(t, withTrustedProxies("10.0.0.0/8"))
	if err := hr.svc.StartEmailLogin(context.Background(), "alice@example.com"); err != nil {
		t.Fatal(err)
	}
	m, _ := hr.mailer.last()
	req := httptest.NewRequest("GET", "/api/auth/email/callback?token="+tokenFromLink(t, m.link), nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 198.51.100.1")
	rec := hr.fromIP(t, req, "10.0.0.2")
	var tok string
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			tok = c.Value
		}
	}
	_, sess, err := hr.svc.AuthenticateSession(context.Background(), tok)
	if err != nil {
		t.Fatalf("sign-in through proxy: %d %v", rec.Code, err)
	}
	if sess.IP != "198.51.100.1" {
		t.Fatalf("session IP = %q, want the forwarded client 198.51.100.1", sess.IP)
	}
}

// --- per-recipient -----------------------------------------------------------

func TestMailLimitSuppressesSilently(t *testing.T) {
	hr := newHarness(t, withMailLimit(5))
	for i := 0; i < 5; i++ {
		hr.do(t, emailStart("person@example.com"), nil)
	}
	if hr.mailer.count() != 5 {
		t.Fatalf("sent %d mails, want 5", hr.mailer.count())
	}
	normal := hr.do(t, emailStart("other@example.com"), nil)
	throttled := hr.do(t, emailStart("person@example.com"), nil)

	if hr.mailer.count() != 6 {
		t.Fatalf("sent %d mails, want 6 (the throttled one suppressed)", hr.mailer.count())
	}
	if throttled.Code != normal.Code || throttled.Body.String() != normal.Body.String() {
		t.Fatalf("throttled response %d %q differs from normal %d %q",
			throttled.Code, throttled.Body, normal.Code, normal.Body)
	}
}

func TestMailLimitIsPerNormalizedAddress(t *testing.T) {
	hr := newHarness(t, withMailLimit(2))
	_ = hr.svc.StartEmailLogin(context.Background(), "Person@Example.com")
	_ = hr.svc.StartEmailLogin(context.Background(), "  person@example.COM ")
	_ = hr.svc.StartEmailLogin(context.Background(), "person@example.com")
	if hr.mailer.count() != 2 {
		t.Fatalf("sent %d mails, want 2", hr.mailer.count())
	}
}

func TestMailLimitRecoversAfterWindow(t *testing.T) {
	hr := newHarness(t, withMailLimit(1))
	_ = hr.svc.StartEmailLogin(context.Background(), "person@example.com")
	_ = hr.svc.StartEmailLogin(context.Background(), "person@example.com")
	hr.clock.advance(16 * time.Minute)
	_ = hr.svc.StartEmailLogin(context.Background(), "person@example.com")
	if hr.mailer.count() != 2 {
		t.Fatalf("sent %d mails, want 2", hr.mailer.count())
	}
}

func TestMailLimitIgnoresUnpermittedAddresses(t *testing.T) {
	store := memory.NewAuthStore()
	mailer := &stubMailer{}
	c := &clock{now: time.Now()}
	p := baseParams()
	p.SignupEnabled = false
	svc := auth.NewService(store, mailer, nil, p, auth.WithClock(c.Now),
		auth.WithMailLimiter(ratelimit.New(1, 15*time.Minute, c.Now)))
	ctx := context.Background()
	if _, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "member@example.com"},
		auth.Identity{Kind: auth.IdentityEmail, Email: "member@example.com", EmailVerified: true}); err != nil {
		t.Fatal(err)
	}

	// Unpermitted (no account, signup off): no mail, and the budget is not
	// touched — not for that address, and not for anyone else's.
	for i := 0; i < 3; i++ {
		_ = svc.StartEmailLogin(ctx, "stranger@example.com")
	}
	if mailer.count() != 0 {
		t.Fatal("an unpermitted address got a mail")
	}
	_ = svc.StartEmailLogin(ctx, "member@example.com")
	if mailer.count() != 1 {
		t.Fatal("the member's first mail was suppressed")
	}
}

func TestMailLimitAppliesWithoutIPLimit(t *testing.T) {
	hr := newHarness(t, withMailLimit(1)) // no IP limiter
	hr.fromIP(t, emailStart("person@example.com"), "203.0.113.1")
	hr.fromIP(t, emailStart("person@example.com"), "203.0.113.2")
	if hr.mailer.count() != 1 {
		t.Fatalf("sent %d mails, want 1", hr.mailer.count())
	}
}

// --- passkey sign-in and the IP budget, end to end ---------------------------

func TestIPLimitCoversPasskeyFinish(t *testing.T) {
	hr := newHarness(t, withIPLimit(3))
	tok := hr.signIn(t, "alice@example.com", false) // spends budget from 192.0.2.1, not 203.0.113.7
	a := passkeytest.New(passkeyOrigin)
	cred, _ := hr.registerPasskey(t, a, tok, "")

	start := hr.fromIP(t, httptest.NewRequest("POST", "/api/auth/passkeys/login/start", nil), "203.0.113.7")
	var c ceremony
	_ = json.Unmarshal(start.Body.Bytes(), &c)
	resp, _ := a.Login(c.Options, cred)
	body, _ := json.Marshal(map[string]any{"ceremony_id": c.CeremonyID, "credential": resp})
	hr.fromIP(t, emailStart("x@example.com"), "203.0.113.7")
	hr.fromIP(t, emailStart("x@example.com"), "203.0.113.7")

	rec := hr.fromIP(t, httptest.NewRequest("POST", "/api/auth/passkeys/login/finish", strings.NewReader(string(body))), "203.0.113.7")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("finish over budget = %d, want 429", rec.Code)
	}
	conforms(t, "POST", "/api/auth/passkeys/login/finish", rec)
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("a throttled finish set a session cookie")
	}
}
