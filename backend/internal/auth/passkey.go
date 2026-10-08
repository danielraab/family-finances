package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"
)

// WebAuthn runs the cryptographic half of the passkey ceremonies.
// internal/passkeyauth implements it with github.com/go-webauthn/webauthn;
// auth keeps everything else — who may register, which session a ceremony
// belongs to, and what a successful sign-in grants. Options and responses
// pass through as the WebAuthn JSON forms the browser speaks; State is the
// adapter's opaque per-ceremony state, stored by auth between start and
// finish.
type WebAuthn interface {
	// BeginRegistration returns creation options (resident key and user
	// verification required, no attestation) excluding the given credential
	// ids, plus the state FinishRegistration needs.
	BeginRegistration(user PasskeyUser, exclude [][]byte) (options json.RawMessage, state []byte, err error)
	// FinishRegistration verifies a registration response (challenge,
	// relying party, origin, user verification) against state.
	FinishRegistration(user PasskeyUser, state []byte, response json.RawMessage) (WebAuthnCredential, error)
	// BeginLogin returns assertion options with an empty allow-list
	// (discoverable credentials) and required user verification.
	BeginLogin() (options json.RawMessage, state []byte, err error)
	// FinishLogin verifies an assertion against state, resolving the
	// presented credential through lookup, which receives the credential id
	// and user handle from the response.
	FinishLogin(state []byte, response json.RawMessage, lookup func(credentialID, userHandle []byte) (Passkey, error)) (WebAuthnAssertion, error)
}

// PasskeyUser is the account a registration ceremony is for. Its WebAuthn
// user handle is []byte(ID) — opaque, never the email.
type PasskeyUser struct {
	ID          string
	Name        string
	DisplayName string
}

// WebAuthnCredential is a verified, newly registered credential.
type WebAuthnCredential struct {
	CredentialID   []byte
	PublicKey      []byte
	SignCount      uint32
	Transports     []string
	AAGUID         []byte
	BackupEligible bool
	BackupState    bool
}

// WebAuthnAssertion is the outcome of a verified sign-in assertion.
type WebAuthnAssertion struct {
	CredentialID []byte
	SignCount    uint32
	BackupState  bool
	// CloneWarning is set when the stored and presented signature counters
	// are both non-zero and the presented one did not increase.
	CloneWarning bool
}

// PasskeyCeremony is what a start endpoint hands the browser: the id to echo
// on finish, and the options for navigator.credentials.
type PasskeyCeremony struct {
	CeremonyID string          `json:"ceremony_id"`
	Options    json.RawMessage `json:"options"`
}

const (
	// passkeyChallengeTTL bounds how long a started ceremony can be finished:
	// comfortably above the 120s the browser prompt is given.
	passkeyChallengeTTL = 5 * time.Minute
	// defaultPasskeyName is stored when no name is supplied.
	defaultPasskeyName = "Passkey"
	// maxPasskeyNameLen is the longest accepted name, in characters.
	maxPasskeyNameLen = 100
)

// ProviderLookup names the provider of a passkey from its AAGUID, reporting
// false for an absent, all-zero or unknown one. package main adapts
// internal/aaguid to it.
type ProviderLookup func(aaguid []byte) (PasskeyProvider, bool)

// WithPasskeyProviders wires the AAGUID → provider lookup. Without it every
// passkey's provider is nil and unnamed passkeys are called "Passkey".
func WithPasskeyProviders(l ProviderLookup) Option { return func(s *Service) { s.providers = l } }

// WithWebAuthn wires the passkey ceremony implementation. Without it every
// passkey ceremony fails with ErrPasskeyInvalid/ErrPasskeyAuthFailed, while
// listing and deleting passkeys still work.
func WithWebAuthn(w WebAuthn) Option { return func(s *Service) { s.webauthn = w } }

// PasskeyRegistrationUntil is the deadline for registering a passkey on sess:
// its creation time plus the re-authentication window for a web session, or
// nil for an API session, which may never register one.
func (s *Service) PasskeyRegistrationUntil(sess Session) *time.Time {
	if sess.Client != ClientWeb {
		return nil
	}
	t := sess.CreatedAt.Add(s.p.PasskeyReauthWindow)
	return &t
}

// requireFreshWebSession is the registration guard: a browser session created
// within the re-authentication window. Freshness is measured from CreatedAt,
// which activity never moves, so a stolen long-lived cookie cannot qualify.
func (s *Service) requireFreshWebSession(sess Session) error {
	if sess.Client != ClientWeb {
		return ErrPasskeyWebOnly
	}
	if s.now().Sub(sess.CreatedAt) > s.p.PasskeyReauthWindow {
		return ErrReauthRequired
	}
	return nil
}

func passkeyUser(u User) PasskeyUser {
	display := u.DisplayName
	if display == "" {
		display = u.Email
	}
	return PasskeyUser{ID: u.ID, Name: u.Email, DisplayName: display}
}

// StartPasskeyRegistration is POST /api/auth/passkeys/register/start.
func (s *Service) StartPasskeyRegistration(ctx context.Context, user User, sess Session) (PasskeyCeremony, error) {
	if err := s.requireFreshWebSession(sess); err != nil {
		return PasskeyCeremony{}, err
	}
	if s.webauthn == nil {
		return PasskeyCeremony{}, ErrPasskeyInvalid
	}
	existing, err := s.store.ListPasskeysByUser(ctx, user.ID)
	if err != nil {
		return PasskeyCeremony{}, err
	}
	exclude := make([][]byte, 0, len(existing))
	for _, p := range existing {
		exclude = append(exclude, p.CredentialID)
	}
	options, state, err := s.webauthn.BeginRegistration(passkeyUser(user), exclude)
	if err != nil {
		return PasskeyCeremony{}, fmt.Errorf("begin passkey registration: %w", err)
	}
	return s.storeCeremony(ctx, ChallengeRegistration, sess.ID, options, state)
}

// FinishPasskeyRegistration is POST /api/auth/passkeys/register/finish. The
// freshness rule is checked again here — the check that actually matters,
// since start alone could be raced against the window's end.
func (s *Service) FinishPasskeyRegistration(ctx context.Context, user User, sess Session, ceremonyID, rawName string, response json.RawMessage) (PasskeyInfo, error) {
	if err := s.requireFreshWebSession(sess); err != nil {
		return PasskeyInfo{}, err
	}
	name, err := normalizePasskeyName(rawName)
	if err != nil {
		return PasskeyInfo{}, err
	}
	ch, err := s.consumeCeremony(ctx, ceremonyID, ChallengeRegistration)
	if err != nil {
		return PasskeyInfo{}, err
	}
	if ch.SessionID != sess.ID {
		return PasskeyInfo{}, ErrCeremonyInvalid
	}
	if s.webauthn == nil {
		return PasskeyInfo{}, ErrPasskeyInvalid
	}
	cred, err := s.webauthn.FinishRegistration(passkeyUser(user), ch.Data, response)
	if err != nil {
		slog.InfoContext(ctx, "passkey registration rejected", "user_id", user.ID, "error", err)
		return PasskeyInfo{}, ErrPasskeyInvalid
	}
	p, err := s.store.CreatePasskey(ctx, Passkey{
		UserID:         user.ID,
		CredentialID:   cred.CredentialID,
		PublicKey:      cred.PublicKey,
		SignCount:      cred.SignCount,
		Transports:     cred.Transports,
		AAGUID:         cred.AAGUID,
		BackupEligible: cred.BackupEligible,
		BackupState:    cred.BackupState,
		Name:           s.defaultPasskeyName(name, cred.AAGUID),
		CreatedAt:      s.now(),
	})
	if err != nil {
		return PasskeyInfo{}, err
	}
	return s.passkeyInfo(p, sess), nil
}

// StartPasskeyLogin is POST /api/auth/passkeys/login/start: a discoverable
// ceremony, bound to no user and no session.
func (s *Service) StartPasskeyLogin(ctx context.Context) (PasskeyCeremony, error) {
	if s.webauthn == nil {
		return PasskeyCeremony{}, ErrPasskeyAuthFailed
	}
	options, state, err := s.webauthn.BeginLogin()
	if err != nil {
		return PasskeyCeremony{}, fmt.Errorf("begin passkey login: %w", err)
	}
	return s.storeCeremony(ctx, ChallengeLogin, "", options, state)
}

// CompletePasskeyLogin is POST /api/auth/passkeys/login/finish. On success it
// records the passkey's use and issues a web session bound to the passkey —
// always a cookie session, since the ceremony is browser-only. A passkey
// never creates or links an account.
func (s *Service) CompletePasskeyLogin(ctx context.Context, ceremonyID string, response json.RawMessage, sc SessionContext) (User, string, error) {
	ch, err := s.consumeCeremony(ctx, ceremonyID, ChallengeLogin)
	if err != nil {
		return User{}, "", err
	}
	if s.webauthn == nil {
		return User{}, "", ErrPasskeyAuthFailed
	}

	var matched Passkey
	var lookupErr error
	lookup := func(credentialID, userHandle []byte) (Passkey, error) {
		p, err := s.store.PasskeyByCredentialID(ctx, credentialID)
		if err != nil {
			lookupErr = err
			return Passkey{}, err
		}
		if !bytes.Equal(userHandle, []byte(p.UserID)) {
			lookupErr = ErrPasskeyAuthFailed
			return Passkey{}, ErrPasskeyAuthFailed
		}
		matched = p
		return p, nil
	}
	assertion, err := s.webauthn.FinishLogin(ch.Data, response, lookup)
	if err != nil {
		// A store failure is ours, not the presenter's — surface it as such.
		if lookupErr != nil && !errors.Is(lookupErr, ErrNotFound) && !errors.Is(lookupErr, ErrPasskeyAuthFailed) {
			return User{}, "", lookupErr
		}
		slog.InfoContext(ctx, "passkey sign-in rejected", "error", err)
		return User{}, "", ErrPasskeyAuthFailed
	}
	if matched.ID == "" || !bytes.Equal(assertion.CredentialID, matched.CredentialID) {
		return User{}, "", ErrPasskeyAuthFailed
	}
	if assertion.CloneWarning {
		slog.WarnContext(ctx, "passkey signature counter did not increase; possible cloned authenticator",
			"user_id", matched.UserID, "passkey_id", matched.ID)
		return User{}, "", ErrPasskeyAuthFailed
	}

	user, err := s.store.UserByID(ctx, matched.UserID)
	if err != nil {
		return User{}, "", err
	}
	if user.Disabled || user.DeletedAt != nil {
		return User{}, "", ErrAccountDisabled
	}
	now := s.now()
	if err := s.store.UpdatePasskeyUsage(ctx, matched.ID, assertion.SignCount, assertion.BackupState, now); err != nil {
		return User{}, "", err
	}
	sc.Client = ClientWeb
	token, err := s.issueSessionTokenWith(ctx, user, sc, matched.ID)
	if err != nil {
		return User{}, "", err
	}
	return user, token, nil
}

// ListPasskeys is GET /api/auth/passkeys: the caller's own passkeys, oldest
// first, with the one behind the requesting session marked current.
func (s *Service) ListPasskeys(ctx context.Context, userID string, sess Session) ([]PasskeyInfo, error) {
	list, err := s.store.ListPasskeysByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PasskeyInfo, 0, len(list))
	for _, p := range list {
		out = append(out, s.passkeyInfo(p, sess))
	}
	return out, nil
}

// DeletePasskey is DELETE /api/auth/passkeys/{id}: allowed on any session of
// any age. The store deletes the passkey's sessions with it. It reports
// whether the requesting session was one of them, so the handler can clear
// the cookie.
func (s *Service) DeletePasskey(ctx context.Context, userID, id string, sess Session) (endedCurrent bool, err error) {
	if err := s.store.DeletePasskey(ctx, userID, id); err != nil {
		return false, err
	}
	return sess.PasskeyCredentialID != "" && sess.PasskeyCredentialID == id, nil
}

func (s *Service) storeCeremony(ctx context.Context, kind ChallengeKind, sessionID string, options json.RawMessage, state []byte) (PasskeyCeremony, error) {
	now := s.now()
	ch, err := s.store.CreateWebAuthnChallenge(ctx, WebAuthnChallenge{
		Kind:      kind,
		SessionID: sessionID,
		Data:      state,
		ExpiresAt: now.Add(passkeyChallengeTTL),
	})
	if err != nil {
		return PasskeyCeremony{}, err
	}
	return PasskeyCeremony{CeremonyID: ch.ID, Options: options}, nil
}

// consumeCeremony takes a challenge out of the store exactly once; an
// unknown, already-used or expired one is ErrCeremonyInvalid.
func (s *Service) consumeCeremony(ctx context.Context, id string, kind ChallengeKind) (WebAuthnChallenge, error) {
	if id == "" {
		return WebAuthnChallenge{}, ErrCeremonyInvalid
	}
	ch, err := s.store.ConsumeWebAuthnChallenge(ctx, id, kind)
	if errors.Is(err, ErrNotFound) {
		return WebAuthnChallenge{}, ErrCeremonyInvalid
	}
	if err != nil {
		return WebAuthnChallenge{}, err
	}
	if s.now().After(ch.ExpiresAt) {
		return WebAuthnChallenge{}, ErrCeremonyInvalid
	}
	return ch, nil
}

// normalizePasskeyName trims a submitted name and enforces its length. ""
// means "none given"; defaultPasskeyName fills it in once the AAGUID is known.
func normalizePasskeyName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", nil
	}
	if utf8.RuneCountInString(name) > maxPasskeyNameLen {
		return "", ErrInvalidPasskeyName
	}
	return name, nil
}

// defaultPasskeyName names an unnamed passkey after its provider ("Apple
// Passwords"), falling back to "Passkey". The name is stored, so it stays put
// if the provider list later renames the provider.
func (s *Service) defaultPasskeyName(name string, aaguid []byte) string {
	if name != "" {
		return name
	}
	if p := s.provider(aaguid); p != nil {
		if r := []rune(p.Name); len(r) > maxPasskeyNameLen {
			return string(r[:maxPasskeyNameLen])
		}
		return p.Name
	}
	return defaultPasskeyName
}

func (s *Service) provider(aaguid []byte) *PasskeyProvider {
	if s.providers == nil {
		return nil
	}
	if p, ok := s.providers(aaguid); ok {
		return &p
	}
	return nil
}

func (s *Service) passkeyInfo(p Passkey, sess Session) PasskeyInfo {
	return PasskeyInfo{
		Provider:   s.provider(p.AAGUID),
		ID:         p.ID,
		Name:       p.Name,
		CreatedAt:  p.CreatedAt,
		LastUsedAt: p.LastUsedAt,
		BackedUp:   p.BackupState,
		Current:    sess.PasskeyCredentialID != "" && sess.PasskeyCredentialID == p.ID,
	}
}
