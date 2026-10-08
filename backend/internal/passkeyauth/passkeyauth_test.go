package passkeyauth_test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/passkeyauth"
	"at.draab/familyfinances/internal/passkeyauth/passkeytest"
)

const baseURL = "https://money.example.com"

var alice = auth.PasskeyUser{ID: "user-alice", Name: "alice@example.com", DisplayName: "Alice"}

func newClient(t *testing.T) *passkeyauth.Client {
	t.Helper()
	c, err := passkeyauth.New(baseURL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

// register runs a full registration ceremony and returns the stored-form
// passkey plus the authenticator's credential.
func register(t *testing.T, c *passkeyauth.Client, a *passkeytest.Authenticator, counter uint32) (auth.Passkey, *passkeytest.Credential) {
	t.Helper()
	opts, state, err := c.BeginRegistration(alice, nil)
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}
	resp, cred, err := a.Register(opts, counter)
	if err != nil {
		t.Fatalf("authenticator Register: %v", err)
	}
	got, err := c.FinishRegistration(alice, state, resp)
	if err != nil {
		t.Fatalf("FinishRegistration: %v", err)
	}
	return auth.Passkey{
		ID: "pk1", UserID: alice.ID, CredentialID: got.CredentialID, PublicKey: got.PublicKey,
		SignCount: got.SignCount, Transports: got.Transports, AAGUID: got.AAGUID,
		BackupEligible: got.BackupEligible, BackupState: got.BackupState,
	}, cred
}

func TestNewRejectsRelativeBaseURL(t *testing.T) {
	if _, err := passkeyauth.New("/not-absolute"); err == nil {
		t.Fatal("New accepted a relative base URL")
	}
}

func TestRegistrationOptions(t *testing.T) {
	c := newClient(t)
	opts, _, err := c.BeginRegistration(alice, [][]byte{[]byte("existing-cred")})
	if err != nil {
		t.Fatal(err)
	}
	var o struct {
		RP                     struct{ ID string } `json:"rp"`
		User                   struct{ ID, Name string }
		Attestation            string `json:"attestation"`
		ExcludeCredentials     []any  `json:"excludeCredentials"`
		AuthenticatorSelection struct {
			ResidentKey      string `json:"residentKey"`
			UserVerification string `json:"userVerification"`
		} `json:"authenticatorSelection"`
	}
	if err := json.Unmarshal(opts, &o); err != nil {
		t.Fatal(err)
	}
	if o.RP.ID != "money.example.com" {
		t.Errorf("rp.id = %q, want money.example.com", o.RP.ID)
	}
	if handle, err := base64.RawURLEncoding.DecodeString(o.User.ID); err != nil || string(handle) != alice.ID {
		t.Errorf("user.id = %q, want the opaque user id %q (never the email)", o.User.ID, alice.ID)
	}
	if o.AuthenticatorSelection.ResidentKey != "required" || o.AuthenticatorSelection.UserVerification != "required" {
		t.Errorf("authenticatorSelection = %+v, want resident key and UV required", o.AuthenticatorSelection)
	}
	if o.Attestation != "none" {
		t.Errorf("attestation = %q, want none", o.Attestation)
	}
	if len(o.ExcludeCredentials) != 1 {
		t.Errorf("excludeCredentials = %v, want the one existing credential", o.ExcludeCredentials)
	}
}

func TestRegisterThenDiscoverableLogin(t *testing.T) {
	c := newClient(t)
	a := passkeytest.New(baseURL)
	stored, cred := register(t, c, a, 1)
	if !stored.BackupEligible || !stored.BackupState || len(stored.PublicKey) == 0 {
		t.Fatalf("registered credential = %+v", stored)
	}

	opts, state, err := c.BeginLogin()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(opts), "allowCredentials\":[{") {
		t.Fatalf("login options carry an allow-list: %s", opts)
	}
	resp, err := a.Login(opts, cred)
	if err != nil {
		t.Fatal(err)
	}
	var gotHandle []byte
	assertion, err := c.FinishLogin(state, resp, func(credentialID, userHandle []byte) (auth.Passkey, error) {
		gotHandle = userHandle
		return stored, nil
	})
	if err != nil {
		t.Fatalf("FinishLogin: %v", err)
	}
	if string(gotHandle) != alice.ID {
		t.Errorf("user handle = %q, want %q", gotHandle, alice.ID)
	}
	if assertion.SignCount != 2 || assertion.CloneWarning {
		t.Errorf("assertion = %+v, want counter 2 and no clone warning", assertion)
	}
}

func TestCounterRegressionWarns(t *testing.T) {
	c := newClient(t)
	a := passkeytest.New(baseURL)
	stored, cred := register(t, c, a, 5)
	stored.SignCount = 50 // the server has seen a higher counter than the device will report

	opts, state, _ := c.BeginLogin()
	resp, _ := a.Login(opts, cred)
	assertion, err := c.FinishLogin(state, resp, func([]byte, []byte) (auth.Passkey, error) { return stored, nil })
	if err != nil {
		t.Fatalf("FinishLogin: %v", err)
	}
	if !assertion.CloneWarning {
		t.Fatal("counter regression did not raise CloneWarning")
	}
}

func TestZeroCountersAreAccepted(t *testing.T) {
	c := newClient(t)
	a := passkeytest.New(baseURL)
	stored, cred := register(t, c, a, 0)

	opts, state, _ := c.BeginLogin()
	resp, _ := a.Login(opts, cred)
	assertion, err := c.FinishLogin(state, resp, func([]byte, []byte) (auth.Passkey, error) { return stored, nil })
	if err != nil || assertion.CloneWarning {
		t.Fatalf("synced passkey with zero counter rejected: %+v, %v", assertion, err)
	}
}

func TestWrongOriginIsRejected(t *testing.T) {
	c := newClient(t)
	opts, state, _ := c.BeginRegistration(alice, nil)
	resp, _, _ := passkeytest.New("https://evil.example.com").Register(opts, 0)
	if _, err := c.FinishRegistration(alice, state, resp); err == nil {
		t.Fatal("registration from a foreign origin was accepted")
	}

	a := passkeytest.New(baseURL)
	stored, cred := register(t, c, a, 0)
	a.Origin = "http://localhost:5173"
	lopts, lstate, _ := c.BeginLogin()
	lresp, _ := a.Login(lopts, cred)
	if _, err := c.FinishLogin(lstate, lresp, func([]byte, []byte) (auth.Passkey, error) { return stored, nil }); err == nil {
		t.Fatal("login from a foreign origin was accepted")
	}
}

func TestMissingUserVerificationIsRejected(t *testing.T) {
	c := newClient(t)
	opts, state, _ := c.BeginRegistration(alice, nil)
	a := passkeytest.New(baseURL)
	a.SkipUV = true
	resp, _, _ := a.Register(opts, 0)
	if _, err := c.FinishRegistration(alice, state, resp); err == nil {
		t.Fatal("registration without user verification was accepted")
	}

	a.SkipUV = false
	stored, cred := register(t, c, a, 0)
	a.SkipUV = true
	lopts, lstate, _ := c.BeginLogin()
	lresp, _ := a.Login(lopts, cred)
	if _, err := c.FinishLogin(lstate, lresp, func([]byte, []byte) (auth.Passkey, error) { return stored, nil }); err == nil {
		t.Fatal("login without user verification was accepted")
	}
}

func TestStateIsBoundToItsCeremony(t *testing.T) {
	c := newClient(t)
	a := passkeytest.New(baseURL)
	opts1, _, _ := c.BeginRegistration(alice, nil)
	_, state2, _ := c.BeginRegistration(alice, nil)
	resp, _, _ := a.Register(opts1, 0)
	if _, err := c.FinishRegistration(alice, state2, resp); err == nil {
		t.Fatal("a response for one challenge verified against another ceremony's state")
	}
}

func TestBadSignatureIsRejected(t *testing.T) {
	c := newClient(t)
	a := passkeytest.New(baseURL)
	stored, cred := register(t, c, a, 0)
	_, other := register(t, c, a, 0)
	opts, state, _ := c.BeginLogin()
	// Sign with another credential's key but present the first credential's id.
	forged := *cred
	forged.Key = other.Key
	resp, _ := a.Login(opts, &forged)
	if _, err := c.FinishLogin(state, resp, func([]byte, []byte) (auth.Passkey, error) { return stored, nil }); err == nil {
		t.Fatal("assertion signed by the wrong key was accepted")
	}
}
