package auth_test

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/passkeyauth/passkeytest"
)

// googlePasswordManager is a real entry of the embedded AAGUID list.
const googlePasswordManager = "ea9b8d66-4d01-1d21-3ce4-b6b48cb575d4"

func authenticatorWithAAGUID(t *testing.T, id string) *passkeytest.Authenticator {
	t.Helper()
	a := passkeytest.New(passkeyOrigin)
	b, err := hex.DecodeString(strings.ReplaceAll(id, "-", ""))
	if err != nil {
		t.Fatal(err)
	}
	a.AAGUID = b
	return a
}

func TestPasskeyProviderFromKnownAAGUID(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	a := authenticatorWithAAGUID(t, googlePasswordManager)

	c := hr.registerStart(t, tok)
	resp, _, _ := a.Register(c.Options, 0)
	rec := hr.registerFinish(t, tok, c, "", resp)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var created auth.PasskeyInfo
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.Provider == nil || created.Provider.Name != "Google Password Manager" {
		t.Fatalf("register response provider = %+v", created.Provider)
	}
	if created.Name != "Google Password Manager" {
		t.Fatalf("unnamed passkey name = %q, want the provider name", created.Name)
	}

	list := hr.listPasskeys(t, tok)
	p := list[0].Provider
	if p == nil || p.Name != "Google Password Manager" ||
		!strings.HasPrefix(p.IconLight, "data:image/svg+xml;base64,") ||
		!strings.HasPrefix(p.IconDark, "data:image/svg+xml;base64,") {
		t.Fatalf("listed provider = %+v", p)
	}
}

func TestPasskeyExplicitNameBeatsProvider(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	_, id := hr.registerPasskey(t, authenticatorWithAAGUID(t, googlePasswordManager), tok, "Work laptop")
	list := hr.listPasskeys(t, tok)
	if list[0].ID != id || list[0].Name != "Work laptop" || list[0].Provider == nil {
		t.Fatalf("list = %+v", list[0])
	}
}

func TestPasskeyWithoutKnownProvider(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	hr.registerPasskey(t, passkeytest.New(passkeyOrigin), tok, "")                                     // zero AAGUID
	hr.registerPasskey(t, authenticatorWithAAGUID(t, "00112233-4455-6677-8899-aabbccddeeff"), tok, "") // unknown

	for _, p := range hr.listPasskeys(t, tok) {
		if p.Provider != nil || p.Name != "Passkey" {
			t.Errorf("passkey %+v: want no provider and the default name", p)
		}
	}
	// provider must be present as an explicit null, not omitted.
	rec := hr.doToken(t, mustReq("GET", "/api/auth/passkeys"), tok, false)
	if !strings.Contains(rec.Body.String(), `"provider":null`) {
		t.Fatalf("provider not serialized as null: %s", rec.Body)
	}
}

func mustReq(method, target string) *http.Request {
	req, _ := http.NewRequest(method, target, nil)
	return req
}
