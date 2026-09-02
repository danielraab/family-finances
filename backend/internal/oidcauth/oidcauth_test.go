package oidcauth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// stubProvider is a throwaway OIDC provider: discovery, JWKS, and a token
// endpoint that returns whatever id_token the test last armed.
type stubProvider struct {
	srv         *httptest.Server
	key         *rsa.PrivateKey
	kid         string
	nextIDToken string
}

func newStubProvider(t *testing.T) *stubProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa key: %v", err)
	}
	p := &stubProvider{key: key, kid: "test-key-1"}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"issuer":                 p.srv.URL,
			"authorization_endpoint": p.srv.URL + "/auth",
			"token_endpoint":         p.srv.URL + "/token",
			"jwks_uri":               p.srv.URL + "/jwks",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"keys": []map[string]any{{
			"kty": "RSA", "use": "sig", "alg": "RS256", "kid": p.kid,
			"n": b64(p.key.PublicKey.N.Bytes()),
			"e": "AQAB",
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"access_token": "stub-access", "token_type": "Bearer", "expires_in": 3600,
			"id_token": p.nextIDToken,
		})
	})

	p.srv = httptest.NewServer(mux)
	t.Cleanup(p.srv.Close)
	return p
}

// mint builds a signed RS256 id_token from claims, defaulting iss/aud/exp/iat.
func (p *stubProvider) mint(t *testing.T, clientID string, claims map[string]any) string {
	t.Helper()
	full := map[string]any{
		"iss": p.srv.URL,
		"aud": clientID,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	for k, v := range claims {
		full[k] = v
	}
	header := b64(mustJSON(t, map[string]any{"alg": "RS256", "typ": "JWT", "kid": p.kid}))
	payload := b64(mustJSON(t, full))
	signingInput := header + "." + payload
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, p.key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signingInput + "." + b64(sig)
}

func (p *stubProvider) newClient(t *testing.T, clientID string) *Client {
	t.Helper()
	c, err := New(context.Background(), Config{
		Issuer:      p.srv.URL,
		ClientID:    clientID,
		RedirectURL: "https://ff.example/api/auth/oidc/callback",
		Scopes:      []string{"openid", "email"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestAuthCodeURLCarriesPKCEAndNonce(t *testing.T) {
	p := newStubProvider(t)
	c := p.newClient(t, "client-1")

	u := c.AuthCodeURL("state-xyz", "nonce-abc", "verifier-0123456789012345678901234567890123456789")
	for _, want := range []string{"state=state-xyz", "nonce=nonce-abc", "code_challenge=", "code_challenge_method=S256", "redirect_uri="} {
		if !strings.Contains(u, want) {
			t.Errorf("AuthCodeURL missing %q:\n%s", want, u)
		}
	}
}

func TestVerifyIDTokenHappyPath(t *testing.T) {
	p := newStubProvider(t)
	c := p.newClient(t, "client-1")

	p.nextIDToken = p.mint(t, "client-1", map[string]any{
		"sub": "subject-42", "nonce": "n-1", "email": "person@example.com", "email_verified": true,
	})
	raw, err := c.Exchange(context.Background(), "any-code", "verifier-0123456789012345678901234567890123456789")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	claims, err := c.VerifyIDToken(context.Background(), raw, "n-1")
	if err != nil {
		t.Fatalf("VerifyIDToken: %v", err)
	}
	if claims.Subject != "subject-42" || claims.Email != "person@example.com" || !claims.EmailVerified {
		t.Fatalf("claims = %+v", claims)
	}
	if claims.Issuer != p.srv.URL {
		t.Fatalf("issuer = %q, want %q", claims.Issuer, p.srv.URL)
	}
}

func TestVerifyIDTokenEmailVerifiedFalseIsPassedThrough(t *testing.T) {
	p := newStubProvider(t)
	c := p.newClient(t, "client-1")

	p.nextIDToken = p.mint(t, "client-1", map[string]any{
		"sub": "s", "nonce": "n-2", "email": "x@example.com", "email_verified": false,
	})
	raw, _ := c.Exchange(context.Background(), "code", "verifier-0123456789012345678901234567890123456789")
	claims, err := c.VerifyIDToken(context.Background(), raw, "n-2")
	if err != nil {
		t.Fatalf("VerifyIDToken: %v", err)
	}
	if claims.EmailVerified {
		t.Fatal("email_verified should be false")
	}
}

func TestVerifyIDTokenNonceMismatchRejected(t *testing.T) {
	p := newStubProvider(t)
	c := p.newClient(t, "client-1")

	p.nextIDToken = p.mint(t, "client-1", map[string]any{"sub": "s", "nonce": "issued-nonce"})
	raw, _ := c.Exchange(context.Background(), "code", "verifier-0123456789012345678901234567890123456789")
	if _, err := c.VerifyIDToken(context.Background(), raw, "different-nonce"); err == nil {
		t.Fatal("nonce mismatch not rejected")
	}
}

func TestVerifyIDTokenWrongAudienceRejected(t *testing.T) {
	p := newStubProvider(t)
	c := p.newClient(t, "client-1")

	p.nextIDToken = p.mint(t, "someone-else", map[string]any{"sub": "s", "nonce": "n"})
	raw, _ := c.Exchange(context.Background(), "code", "verifier-0123456789012345678901234567890123456789")
	if _, err := c.VerifyIDToken(context.Background(), raw, "n"); err == nil {
		t.Fatal("wrong-aud id_token not rejected")
	}
}

func TestVerifyIDTokenTamperedSignatureRejected(t *testing.T) {
	p := newStubProvider(t)
	c := p.newClient(t, "client-1")

	tok := p.mint(t, "client-1", map[string]any{"sub": "s", "nonce": "n"})
	// Flip the last character of the signature segment.
	parts := strings.Split(tok, ".")
	sig := []byte(parts[2])
	sig[len(sig)-1] ^= 0x01
	tampered := parts[0] + "." + parts[1] + "." + string(sig)

	if _, err := c.VerifyIDToken(context.Background(), tampered, "n"); err == nil {
		t.Fatal("tampered id_token not rejected")
	}
}

// --- helpers ---------------------------------------------------------

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
