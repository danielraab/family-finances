package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
)

// fakeAuth resolves a fixed set of tokens to users.
type fakeAuth struct{ byToken map[string]auth.User }

func (f fakeAuth) AuthenticateSession(_ context.Context, token string) (auth.User, auth.Session, error) {
	if u, ok := f.byToken[token]; ok {
		return u, auth.Session{ID: "sess-" + token, UserID: u.ID}, nil
	}
	return auth.User{}, auth.Session{}, auth.ErrNotFound
}

func probeUser(w http.ResponseWriter, r *http.Request) {
	if u, ok := auth.UserFromContext(r.Context()); ok {
		_, _ = w.Write([]byte("user:" + u.ID))
		return
	}
	_, _ = w.Write([]byte("anon"))
}

func TestAuthResolveBearer(t *testing.T) {
	fa := fakeAuth{byToken: map[string]auth.User{"tok-a": {ID: "u1"}}}
	h := authResolve(fa, http.HandlerFunc(probeUser))

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer tok-a")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Body.String() != "user:u1" {
		t.Fatalf("body = %q, want user:u1", rec.Body.String())
	}
}

func TestAuthResolveCookie(t *testing.T) {
	fa := fakeAuth{byToken: map[string]auth.User{"tok-c": {ID: "u2"}}}
	h := authResolve(fa, http.HandlerFunc(probeUser))

	req := httptest.NewRequest("GET", "/x", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "tok-c"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Body.String() != "user:u2" {
		t.Fatalf("body = %q, want user:u2", rec.Body.String())
	}
}

func TestAuthResolveHeaderWinsOverCookie(t *testing.T) {
	fa := fakeAuth{byToken: map[string]auth.User{
		"tok-hdr":    {ID: "from-header"},
		"tok-cookie": {ID: "from-cookie"},
	}}
	h := authResolve(fa, http.HandlerFunc(probeUser))

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer tok-hdr")
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "tok-cookie"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Body.String() != "user:from-header" {
		t.Fatalf("body = %q, want user:from-header", rec.Body.String())
	}
}

func TestAuthResolveAttachesSession(t *testing.T) {
	fa := fakeAuth{byToken: map[string]auth.User{"tok-b": {ID: "u1"}, "tok-c": {ID: "u2"}}}
	h := authResolve(fa, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := auth.SessionFromContext(r.Context())
		if !ok {
			_, _ = w.Write([]byte("no-session"))
			return
		}
		_, _ = w.Write([]byte(sess.ID))
	}))

	bearer := httptest.NewRequest("GET", "/x", nil)
	bearer.Header.Set("Authorization", "Bearer tok-b")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, bearer)
	if rec.Body.String() != "sess-tok-b" {
		t.Fatalf("bearer session = %q, want sess-tok-b", rec.Body.String())
	}

	cookie := httptest.NewRequest("GET", "/x", nil)
	cookie.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "tok-c"})
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, cookie)
	if rec.Body.String() != "sess-tok-c" {
		t.Fatalf("cookie session = %q, want sess-tok-c", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Body.String() != "no-session" {
		t.Fatalf("anonymous request carried a session: %q", rec.Body.String())
	}
}

func TestAuthResolveMissingOrInvalidStaysAnonymous(t *testing.T) {
	fa := fakeAuth{byToken: map[string]auth.User{}}
	h := authResolve(fa, http.HandlerFunc(probeUser))

	// No token.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Body.String() != "anon" {
		t.Fatalf("no-token body = %q, want anon", rec.Body.String())
	}

	// Bad token.
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer nope")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Body.String() != "anon" {
		t.Fatalf("bad-token body = %q, want anon", rec.Body.String())
	}
}

func TestRequireAuth(t *testing.T) {
	protected := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Anonymous -> 401.
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anon status = %d, want 401", rec.Code)
	}

	// With a user on the context -> passes through.
	req := httptest.NewRequest("GET", "/x", nil)
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{ID: "u1"}))
	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authed status = %d, want 200", rec.Code)
	}
}

func TestSignInRoutesReachableWithoutAuth(t *testing.T) {
	// A stub auth handler standing in for the real one; the point is that
	// Routes mounts it under /api/auth/ with no RequireAuth in front.
	stub := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("reached"))
	})
	deps := testDeps()
	deps.AuthHandler = stub

	rec := httptest.NewRecorder()
	Routes(deps).ServeHTTP(rec, httptest.NewRequest("POST", "/api/auth/email/start", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "reached" {
		t.Fatalf("sign-in route: status %d body %q", rec.Code, rec.Body.String())
	}
}

// TestHTTPAPIImportGraphHasNoStorageOrDriver enforces the architecture rule:
// internal/httpapi may resolve an authenticated user, but only through the
// Authenticator interface — never by importing a storage package or a DB
// driver. It shells out to `go list` for the full transitive graph.
func TestHTTPAPIImportGraphHasNoStorageOrDriver(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go tool not on PATH")
	}
	out, err := exec.Command(goBin, "list", "-deps", "-f", "{{.ImportPath}}",
		"at.draab/familyfinances/internal/httpapi").CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.Contains(line, "at.draab/familyfinances/internal/storage/") {
			t.Errorf("internal/httpapi transitively imports a storage package: %s", line)
		}
		if strings.HasPrefix(line, "github.com/jackc/pgx") {
			t.Errorf("internal/httpapi transitively imports the pgx driver: %s", line)
		}
	}
}

// TestEveryAuthSentinelHasAStatus guards the mapping table: an auth sentinel
// that reaches WriteError unregistered would surface as a 500.
func TestEveryAuthSentinelHasAStatus(t *testing.T) {
	want := map[error]int{
		auth.ErrReauthRequired:     http.StatusForbidden,
		auth.ErrPasskeyWebOnly:     http.StatusForbidden,
		auth.ErrCeremonyInvalid:    http.StatusBadRequest,
		auth.ErrPasskeyInvalid:     http.StatusBadRequest,
		auth.ErrPasskeyConflict:    http.StatusConflict,
		auth.ErrPasskeyAuthFailed:  http.StatusUnauthorized,
		auth.ErrInvalidPasskeyName: http.StatusBadRequest,
		auth.ErrRateLimited:        http.StatusTooManyRequests,
	}
	for _, sentinel := range auth.Sentinels {
		rec := httptest.NewRecorder()
		WriteError(rec, httptest.NewRequest("GET", "/x", nil), sentinel)
		if rec.Code == http.StatusInternalServerError {
			t.Errorf("%q maps to 500; register it in init()", sentinel)
		}
		if code, ok := want[sentinel]; ok && rec.Code != code {
			t.Errorf("%q maps to %d, want %d", sentinel, rec.Code, code)
		}
	}
}
