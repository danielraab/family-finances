package auth_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/storage/memory"
)

// --- test doubles --------------------------------------------------------

type sentMail struct {
	kind string // "magic" | "invite"
	addr string
	link string
	by   string
}

type stubMailer struct {
	mu   sync.Mutex
	sent []sentMail
	err  error
}

func (m *stubMailer) SendMagicLink(_ context.Context, addr, link string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, sentMail{kind: "magic", addr: addr, link: link})
	return nil
}

func (m *stubMailer) SendInvite(_ context.Context, addr, link, by string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, sentMail{kind: "invite", addr: addr, link: link, by: by})
	return nil
}

func (m *stubMailer) last() (sentMail, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		return sentMail{}, false
	}
	return m.sent[len(m.sent)-1], true
}

// tokenFromLink pulls the token query value out of a built URL.
func tokenFromLink(t *testing.T, link string) string {
	t.Helper()
	i := strings.Index(link, "token=")
	if i < 0 {
		t.Fatalf("no token in link %q", link)
	}
	return link[i+len("token="):]
}

type stubOIDC struct {
	claims  auth.OIDCClaims
	exchErr error
	verErr  error
}

func (o *stubOIDC) AuthCodeURL(state, nonce, verifier string) string {
	return "https://idp.example/authorize?state=" + state
}
func (o *stubOIDC) Exchange(context.Context, string, string) (string, error) {
	return "raw-id-token", o.exchErr
}
func (o *stubOIDC) VerifyIDToken(context.Context, string, string) (auth.OIDCClaims, error) {
	return o.claims, o.verErr
}

func baseParams() auth.Params {
	return auth.Params{
		BaseURL:       "https://ff.example",
		SessionTTL:    720 * time.Hour,
		SessionMaxTTL: 2160 * time.Hour,
		SignupEnabled: true,
		InviteEnabled: true,
		InviteTTL:     168 * time.Hour,
		MagicLinkTTL:  15 * time.Minute,
		OIDCIssuer:    "https://idp.example",
	}
}

func newSvc(t *testing.T, p auth.Params, opts ...svcOpt) (*auth.Service, *memory.AuthStore, *stubMailer, *clock) {
	t.Helper()
	store := memory.NewAuthStore()
	mailer := &stubMailer{}
	cfg := svcConfig{clock: &clock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}}
	for _, o := range opts {
		o(&cfg)
	}
	svc := auth.NewService(store, mailer, cfg.oidc, p, auth.WithClock(cfg.clock.Now))
	return svc, store, mailer, cfg.clock
}

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type svcConfig struct {
	clock *clock
	oidc  auth.OIDCClient
	label string
}
type svcOpt func(*svcConfig)

func withOIDC(o auth.OIDCClient) svcOpt { return func(c *svcConfig) { c.oidc = o } }
func withOIDCLabel(label string) svcOpt { return func(c *svcConfig) { c.label = label } }

// signInEmail runs a full magic-link sign-in and returns the resulting user
// and session token.
func signInEmail(t *testing.T, svc *auth.Service, mailer *stubMailer, email string) (auth.User, string) {
	t.Helper()
	if err := svc.StartEmailLogin(context.Background(), email); err != nil {
		t.Fatalf("StartEmailLogin(%q): %v", email, err)
	}
	m, ok := mailer.last()
	if !ok {
		t.Fatalf("no email sent for %q", email)
	}
	user, tok, err := svc.CompleteEmailLogin(context.Background(), tokenFromLink(t, m.link), "", auth.SessionContext{Client: auth.ClientAPI})
	if err != nil {
		t.Fatalf("CompleteEmailLogin: %v", err)
	}
	return user, tok
}

// --- tests -------------------------------------------------------------

func TestBootstrapFirstUserIsAdmin(t *testing.T) {
	svc, _, mailer, _ := newSvc(t, baseParams())

	first, _ := signInEmail(t, svc, mailer, "first@example.com")
	if !first.IsAdmin {
		t.Fatal("first account should be admin (bootstrap)")
	}

	second, _ := signInEmail(t, svc, mailer, "second@example.com")
	if second.IsAdmin {
		t.Fatal("second account should not be admin")
	}
}

func TestStartEmailLoginNeverEnumerates(t *testing.T) {
	p := baseParams()
	p.SignupEnabled = false
	svc, _, mailer, _ := newSvc(t, p)

	// No account, no invite, signup disabled: no error, no mail.
	if err := svc.StartEmailLogin(context.Background(), "nobody@example.com"); err != nil {
		t.Fatalf("StartEmailLogin: %v", err)
	}
	if _, ok := mailer.last(); ok {
		t.Fatal("mail sent for a disallowed address")
	}
}

func TestSignupDisabled(t *testing.T) {
	store := memory.NewAuthStore()
	mailer := &stubMailer{}
	clk := &clock{now: time.Now()}

	// Bootstrap one admin with signup enabled.
	p := baseParams()
	svc := auth.NewService(store, mailer, nil, p, auth.WithClock(clk.Now))
	if err := svc.StartEmailLogin(context.Background(), "admin@example.com"); err != nil {
		t.Fatal(err)
	}
	m, _ := mailer.last()
	if _, _, err := svc.CompleteEmailLogin(context.Background(), tokenFromLink(t, m.link), "", auth.SessionContext{}); err != nil {
		t.Fatal(err)
	}

	// Disable signup; a brand-new address must not be able to create an account.
	p.SignupEnabled = false
	svc = auth.NewService(store, mailer, nil, p, auth.WithClock(clk.Now))
	if err := svc.StartEmailLogin(context.Background(), "stranger@example.com"); err != nil {
		t.Fatal(err)
	}
	if last, _ := mailer.last(); last.addr == "stranger@example.com" {
		t.Fatal("mail should not be sent to a stranger while signup is disabled")
	}
}

func TestDomainAllowListGatesCreation(t *testing.T) {
	store := memory.NewAuthStore()
	mailer := &stubMailer{}

	// Bootstrap first (empty table => allowed regardless of domain list).
	p := baseParams()
	p.AllowedEmailDomains = []string{"example.com"}
	svc := auth.NewService(store, mailer, nil, p)
	u1, _ := signInEmail(t, svc, mailer, "boss@example.com")
	if !u1.IsAdmin {
		t.Fatal("bootstrap admin expected")
	}

	// Disallowed domain, no invite: StartEmailLogin sends nothing.
	if err := svc.StartEmailLogin(context.Background(), "x@other.org"); err != nil {
		t.Fatal(err)
	}
	if last, _ := mailer.last(); last.addr == "x@other.org" {
		t.Fatal("mail sent to a domain outside the allow-list")
	}

	// Allowed domain: works.
	u2, _ := signInEmail(t, svc, mailer, "colleague@example.com")
	if u2.IsAdmin {
		t.Fatal("second user should not be admin")
	}
}

func TestInviteBypassesSignupAndDomain(t *testing.T) {
	store := memory.NewAuthStore()
	mailer := &stubMailer{}
	p := baseParams()
	p.AllowedEmailDomains = []string{"example.com"}
	svc := auth.NewService(store, mailer, nil, p)

	admin, _ := signInEmail(t, svc, mailer, "admin@example.com")

	// Now close signup entirely.
	p.SignupEnabled = false
	p.InviteEnabled = true
	svc = auth.NewService(store, mailer, nil, p)

	inv, err := svc.CreateInvite(context.Background(), admin.ID, "guest@outside.net")
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	if inv.Email != "guest@outside.net" {
		t.Fatalf("invite email = %q", inv.Email)
	}
	m, ok := mailer.last()
	if !ok || m.kind != "invite" {
		t.Fatalf("invite mail not sent: %+v", m)
	}

	user, _, err := svc.AcceptInvite(context.Background(), tokenFromLink(t, m.link), "", auth.SessionContext{})
	if err != nil {
		t.Fatalf("AcceptInvite: %v", err)
	}
	if user.Email != "guest@outside.net" {
		t.Fatalf("accepted user email = %q", user.Email)
	}
	if user.IsAdmin {
		t.Fatal("invited user should not be admin")
	}
}

func TestInviteEnabledWhileSignupOn(t *testing.T) {
	p := baseParams()
	p.SignupEnabled = true
	p.InviteEnabled = false // ignored while signup is on
	svc, store, mailer, _ := newSvc(t, p)

	admin, _, err := createBootstrapAdmin(t, svc, store, mailer)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateInvite(context.Background(), admin.ID, "friend@example.com"); err != nil {
		t.Fatalf("invite should be allowed while signup is on: %v", err)
	}
}

func TestFullyClosedInstanceRejectsInvite(t *testing.T) {
	p := baseParams()
	svc, store, mailer, _ := newSvc(t, p)
	admin, _, err := createBootstrapAdmin(t, svc, store, mailer)
	if err != nil {
		t.Fatal(err)
	}

	p.SignupEnabled = false
	p.InviteEnabled = false
	closed := auth.NewService(store, mailer, nil, p)
	if _, err := closed.CreateInvite(context.Background(), admin.ID, "friend@example.com"); !errors.Is(err, auth.ErrSignupDisabled) {
		t.Fatalf("CreateInvite err = %v, want ErrSignupDisabled", err)
	}
}

func createBootstrapAdmin(t *testing.T, svc *auth.Service, _ *memory.AuthStore, mailer *stubMailer) (auth.User, string, error) {
	t.Helper()
	u, tok := signInEmail(t, svc, mailer, "admin@example.com")
	return u, tok, nil
}

func TestMagicLinkSingleUseAndExpiry(t *testing.T) {
	svc, _, mailer, clk := newSvc(t, baseParams())

	if err := svc.StartEmailLogin(context.Background(), "a@example.com"); err != nil {
		t.Fatal(err)
	}
	m, _ := mailer.last()
	tok := tokenFromLink(t, m.link)

	if _, _, err := svc.CompleteEmailLogin(context.Background(), tok, "", auth.SessionContext{}); err != nil {
		t.Fatalf("first use: %v", err)
	}
	if _, _, err := svc.CompleteEmailLogin(context.Background(), tok, "", auth.SessionContext{}); !errors.Is(err, auth.ErrTokenConsumed) {
		t.Fatalf("second use err = %v, want ErrTokenConsumed", err)
	}

	// Fresh token, then let it expire.
	if err := svc.StartEmailLogin(context.Background(), "b@example.com"); err != nil {
		t.Fatal(err)
	}
	m2, _ := mailer.last()
	clk.advance(16 * time.Minute)
	if _, _, err := svc.CompleteEmailLogin(context.Background(), tokenFromLink(t, m2.link), "", auth.SessionContext{}); !errors.Is(err, auth.ErrTokenExpired) {
		t.Fatalf("expired token err = %v, want ErrTokenExpired", err)
	}
}

func TestSessionSlidingAndAbsoluteCap(t *testing.T) {
	p := baseParams()
	p.SessionTTL = 48 * time.Hour
	p.SessionMaxTTL = 96 * time.Hour
	svc, _, mailer, clk := newSvc(t, p)

	user, tok := signInEmail(t, svc, mailer, "s@example.com")

	// Within half the TTL: still valid, no error.
	clk.advance(1 * time.Hour)
	if _, err := svc.Authenticate(context.Background(), tok); err != nil {
		t.Fatalf("early Authenticate: %v", err)
	}

	// Past half-life but under the cap: still authenticates (and slides).
	clk.advance(30 * time.Hour)
	if got, err := svc.Authenticate(context.Background(), tok); err != nil || got.ID != user.ID {
		t.Fatalf("mid Authenticate: got %v err %v", got, err)
	}

	// Now push well past the original 48h expiry: the slide should have kept
	// it alive.
	clk.advance(30 * time.Hour) // total 61h
	if _, err := svc.Authenticate(context.Background(), tok); err != nil {
		t.Fatalf("slid session should still be valid at 61h: %v", err)
	}

	// Past the absolute cap (96h): rejected regardless of recent activity.
	clk.advance(40 * time.Hour) // total 101h
	if _, err := svc.Authenticate(context.Background(), tok); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("over-cap Authenticate err = %v, want ErrNotFound", err)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	svc, _, mailer, _ := newSvc(t, baseParams())
	_, tok := signInEmail(t, svc, mailer, "z@example.com")

	if err := svc.Logout(context.Background(), tok); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := svc.Authenticate(context.Background(), tok); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("post-logout Authenticate err = %v, want ErrNotFound", err)
	}
	// Idempotent.
	if err := svc.Logout(context.Background(), tok); err != nil {
		t.Fatalf("second Logout: %v", err)
	}
}

func TestSecondMethodLinksToSameAccount(t *testing.T) {
	p := baseParams()
	oidc := &stubOIDC{claims: auth.OIDCClaims{Issuer: "https://idp.example", Subject: "sub-1", Email: "person@example.com", EmailVerified: true}}
	svc, _, mailer, _ := newSvc(t, p, withOIDC(oidc))

	// Magic-link account first.
	emailUser, _ := signInEmail(t, svc, mailer, "person@example.com")

	// Then OIDC with the same verified email: must resolve to the same user.
	redirect, err := svc.StartOIDC(context.Background(), "")
	if err != nil {
		t.Fatalf("StartOIDC: %v", err)
	}
	state := redirect[strings.Index(redirect, "state=")+len("state="):]
	oidcUser, _, _, err := svc.CompleteOIDC(context.Background(), state, "code", "", auth.SessionContext{})
	if err != nil {
		t.Fatalf("CompleteOIDC: %v", err)
	}
	if oidcUser.ID != emailUser.ID {
		t.Fatalf("OIDC created a new account %q, want link to %q", oidcUser.ID, emailUser.ID)
	}

	// Signing in with OIDC again returns the same user.
	redirect2, _ := svc.StartOIDC(context.Background(), "")
	state2 := redirect2[strings.Index(redirect2, "state=")+len("state="):]
	again, _, _, err := svc.CompleteOIDC(context.Background(), state2, "code", "", auth.SessionContext{})
	if err != nil || again.ID != emailUser.ID {
		t.Fatalf("repeat OIDC: user %q err %v", again.ID, err)
	}
}

func TestOIDCUnverifiedEmailDoesNotMerge(t *testing.T) {
	p := baseParams()
	oidc := &stubOIDC{claims: auth.OIDCClaims{Issuer: "https://idp.example", Subject: "sub-x", Email: "person@example.com", EmailVerified: false}}
	svc, _, mailer, _ := newSvc(t, p, withOIDC(oidc))

	signInEmail(t, svc, mailer, "person@example.com")

	// Unverified OIDC email that collides with an existing account is not
	// merged and not silently duplicated: the automatic flow refuses it, and
	// the person must sign in and link explicitly (or use a different email).
	redirect, _ := svc.StartOIDC(context.Background(), "")
	state := redirect[strings.Index(redirect, "state=")+len("state="):]
	if _, _, _, err := svc.CompleteOIDC(context.Background(), state, "code", "", auth.SessionContext{}); err == nil {
		t.Fatal("unverified OIDC email must not auto-merge into the existing account")
	}
}

func TestOIDCUnverifiedEmailNewUserStillCreated(t *testing.T) {
	p := baseParams()
	oidc := &stubOIDC{claims: auth.OIDCClaims{Issuer: "https://idp.example", Subject: "sub-fresh", Email: "fresh@example.com", EmailVerified: false}}
	svc, _, _, _ := newSvc(t, p, withOIDC(oidc))

	// No existing account for this address: even unverified, a first sign-in
	// creates the account (there is nothing to merge with).
	redirect, _ := svc.StartOIDC(context.Background(), "")
	state := redirect[strings.Index(redirect, "state=")+len("state="):]
	user, _, _, err := svc.CompleteOIDC(context.Background(), state, "code", "", auth.SessionContext{})
	if err != nil {
		t.Fatalf("CompleteOIDC: %v", err)
	}
	if user.Email != "fresh@example.com" {
		t.Fatalf("user email = %q", user.Email)
	}
}

func TestExplicitLinkWhileAuthenticated(t *testing.T) {
	p := baseParams()
	oidc := &stubOIDC{claims: auth.OIDCClaims{Issuer: "https://idp.example", Subject: "sub-link", Email: "other@example.com", EmailVerified: false}}
	svc, _, mailer, _ := newSvc(t, p, withOIDC(oidc))

	me, _ := signInEmail(t, svc, mailer, "me@example.com")

	// Authenticated: completing an OIDC flow attaches the identity to me, even
	// though the OIDC email differs and is unverified.
	redirect, _ := svc.StartOIDC(context.Background(), "")
	state := redirect[strings.Index(redirect, "state=")+len("state="):]
	linked, _, _, err := svc.CompleteOIDC(context.Background(), state, "code", me.ID, auth.SessionContext{})
	if err != nil {
		t.Fatalf("CompleteOIDC (linking): %v", err)
	}
	if linked.ID != me.ID {
		t.Fatalf("linked identity attached to %q, want %q", linked.ID, me.ID)
	}
}

func TestOIDCStateMismatchRejected(t *testing.T) {
	p := baseParams()
	oidc := &stubOIDC{claims: auth.OIDCClaims{Issuer: "x", Subject: "s"}}
	svc, _, _, _ := newSvc(t, p, withOIDC(oidc))

	if _, _, _, err := svc.CompleteOIDC(context.Background(), "never-issued", "code", "", auth.SessionContext{}); !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("err = %v, want ErrTokenInvalid", err)
	}
}

func TestOIDCVerificationFailureRejected(t *testing.T) {
	p := baseParams()
	oidc := &stubOIDC{verErr: errors.New("bad signature")}
	svc, _, _, _ := newSvc(t, p, withOIDC(oidc))

	redirect, _ := svc.StartOIDC(context.Background(), "")
	state := redirect[strings.Index(redirect, "state=")+len("state="):]
	if _, _, _, err := svc.CompleteOIDC(context.Background(), state, "code", "", auth.SessionContext{}); !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("err = %v, want ErrTokenInvalid", err)
	}
}

func TestSetAndListAdmins(t *testing.T) {
	svc, _, mailer, _ := newSvc(t, baseParams())
	signInEmail(t, svc, mailer, "admin@example.com")
	second, _ := signInEmail(t, svc, mailer, "second@example.com")
	_ = second

	if err := svc.SetAdmin(context.Background(), "second@example.com", true); err != nil {
		t.Fatalf("SetAdmin: %v", err)
	}
	admins, err := svc.ListAdmins(context.Background())
	if err != nil {
		t.Fatalf("ListAdmins: %v", err)
	}
	if len(admins) != 2 {
		t.Fatalf("admins = %v, want 2", admins)
	}

	if err := svc.SetAdmin(context.Background(), "nobody@example.com", true); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("SetAdmin(unknown) = %v, want ErrNotFound", err)
	}
}

func TestServiceOIDCLogin(t *testing.T) {
	p := baseParams()
	p.OIDCLabel = "Continue with Google"

	on, _, _, _ := newSvc(t, p, withOIDC(&stubOIDC{}))
	if label, ok := on.OIDCLogin(); !ok || label != "Continue with Google" {
		t.Fatalf("OIDCLogin() = (%q, %v), want (%q, true)", label, ok, "Continue with Google")
	}

	off, _, _, _ := newSvc(t, p) // no OIDC client constructed
	if label, ok := off.OIDCLogin(); ok || label != "" {
		t.Fatalf(`OIDCLogin() with no provider = (%q, %v), want ("", false)`, label, ok)
	}
}
