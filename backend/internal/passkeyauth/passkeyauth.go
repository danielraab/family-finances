// Package passkeyauth implements auth.WebAuthn with
// github.com/go-webauthn/webauthn. The relying party is derived from
// AUTH_BASE_URL: its host is the RP ID and its origin the only accepted
// origin. Every credential is discoverable (resident key required) and every
// ceremony requires user verification; no attestation is requested. Package
// main builds one Client and injects it into the auth service.
package passkeyauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"at.draab/familyfinances/internal/auth"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// rpDisplayName is shown by some authenticators next to the passkey.
const rpDisplayName = "Family Finances"

// ceremonyTimeout is the client-side timeout put in the options. Expiry on
// the server is enforced by auth from its own challenge TTL, not here.
const ceremonyTimeout = 2 * time.Minute

// Client is the auth.WebAuthn implementation.
type Client struct {
	wa *webauthn.WebAuthn
}

var _ auth.WebAuthn = (*Client)(nil)

// New builds a Client for the relying party at baseURL (AUTH_BASE_URL).
func New(baseURL string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" || u.Hostname() == "" {
		return nil, fmt.Errorf("passkeyauth: AUTH_BASE_URL %q is not an absolute URL", baseURL)
	}
	origin := u.Scheme + "://" + u.Host
	timeout := webauthn.TimeoutConfig{Enforce: false, Timeout: ceremonyTimeout, TimeoutUVD: ceremonyTimeout}
	wa, err := webauthn.New(&webauthn.Config{
		RPID:          u.Hostname(),
		RPDisplayName: rpDisplayName,
		RPOrigins:     []string{origin},
		Timeouts:      webauthn.TimeoutsConfig{Login: timeout, Registration: timeout},
	})
	if err != nil {
		return nil, fmt.Errorf("passkeyauth: %w", err)
	}
	return &Client{wa: wa}, nil
}

// user adapts an account (and, for sign-in, its one presented passkey) to
// the library's User interface. The user handle is the opaque user id.
type user struct {
	id          []byte
	name        string
	displayName string
	creds       []webauthn.Credential
}

func (u *user) WebAuthnID() []byte                         { return u.id }
func (u *user) WebAuthnName() string                       { return u.name }
func (u *user) WebAuthnDisplayName() string                { return u.displayName }
func (u *user) WebAuthnCredentials() []webauthn.Credential { return u.creds }

func (c *Client) BeginRegistration(pu auth.PasskeyUser, exclude [][]byte) (json.RawMessage, []byte, error) {
	excl := make([]protocol.CredentialDescriptor, 0, len(exclude))
	for _, id := range exclude {
		excl = append(excl, protocol.CredentialDescriptor{Type: protocol.PublicKeyCredentialType, CredentialID: id})
	}
	requireResident := true
	creation, session, err := c.wa.BeginRegistration(
		&user{id: []byte(pu.ID), name: pu.Name, displayName: pu.DisplayName},
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			RequireResidentKey: &requireResident,
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			UserVerification:   protocol.VerificationRequired,
		}),
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
		webauthn.WithExclusions(excl),
	)
	if err != nil {
		return nil, nil, err
	}
	return marshalPair(creation.Response, session)
}

func (c *Client) FinishRegistration(pu auth.PasskeyUser, state []byte, response json.RawMessage) (auth.WebAuthnCredential, error) {
	var session webauthn.SessionData
	if err := json.Unmarshal(state, &session); err != nil {
		return auth.WebAuthnCredential{}, fmt.Errorf("passkeyauth: decode ceremony state: %w", err)
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(response)
	if err != nil {
		return auth.WebAuthnCredential{}, err
	}
	cred, err := c.wa.CreateCredential(&user{id: []byte(pu.ID), name: pu.Name, displayName: pu.DisplayName}, session, parsed)
	if err != nil {
		return auth.WebAuthnCredential{}, err
	}
	transports := make([]string, 0, len(cred.Transport))
	for _, t := range cred.Transport {
		transports = append(transports, string(t))
	}
	return auth.WebAuthnCredential{
		CredentialID:   cred.ID,
		PublicKey:      cred.PublicKey,
		SignCount:      cred.Authenticator.SignCount,
		Transports:     transports,
		AAGUID:         cred.Authenticator.AAGUID,
		BackupEligible: cred.Flags.BackupEligible,
		BackupState:    cred.Flags.BackupState,
	}, nil
}

func (c *Client) BeginLogin() (json.RawMessage, []byte, error) {
	assertion, session, err := c.wa.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return nil, nil, err
	}
	return marshalPair(assertion.Response, session)
}

func (c *Client) FinishLogin(state []byte, response json.RawMessage, lookup func(credentialID, userHandle []byte) (auth.Passkey, error)) (auth.WebAuthnAssertion, error) {
	var session webauthn.SessionData
	if err := json.Unmarshal(state, &session); err != nil {
		return auth.WebAuthnAssertion{}, fmt.Errorf("passkeyauth: decode ceremony state: %w", err)
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		return auth.WebAuthnAssertion{}, err
	}
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		p, err := lookup(rawID, userHandle)
		if err != nil {
			return nil, err
		}
		return &user{id: []byte(p.UserID), creds: []webauthn.Credential{storedCredential(p)}}, nil
	}
	_, cred, err := c.wa.ValidatePasskeyLogin(handler, session, parsed)
	if err != nil {
		return auth.WebAuthnAssertion{}, err
	}
	if cred == nil {
		return auth.WebAuthnAssertion{}, errors.New("passkeyauth: no credential validated")
	}
	return auth.WebAuthnAssertion{
		CredentialID: cred.ID,
		SignCount:    cred.Authenticator.SignCount,
		BackupState:  cred.Flags.BackupState,
		CloneWarning: cred.Authenticator.CloneWarning,
	}, nil
}

// storedCredential rebuilds the library's credential record from a stored
// passkey. Every stored passkey was registered with user verification, so
// the UV flag is latched on.
func storedCredential(p auth.Passkey) webauthn.Credential {
	transports := make([]protocol.AuthenticatorTransport, 0, len(p.Transports))
	for _, t := range p.Transports {
		transports = append(transports, protocol.AuthenticatorTransport(t))
	}
	return webauthn.Credential{
		ID:        p.CredentialID,
		PublicKey: p.PublicKey,
		Transport: transports,
		Flags: webauthn.CredentialFlags{
			UserPresent:    true,
			UserVerified:   true,
			BackupEligible: p.BackupEligible,
			BackupState:    p.BackupState,
		},
		Authenticator: webauthn.Authenticator{AAGUID: p.AAGUID, SignCount: p.SignCount},
	}
}

func marshalPair(options any, session *webauthn.SessionData) (json.RawMessage, []byte, error) {
	opts, err := json.Marshal(options)
	if err != nil {
		return nil, nil, err
	}
	state, err := json.Marshal(session)
	if err != nil {
		return nil, nil, err
	}
	return opts, state, nil
}
