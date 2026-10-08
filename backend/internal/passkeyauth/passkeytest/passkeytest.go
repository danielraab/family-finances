// Package passkeytest is a software WebAuthn authenticator for tests: it
// answers creation and assertion options the way a browser plus a platform
// authenticator would (ES256 key, "none" attestation, discoverable
// credentials), so handler tests can drive real passkey ceremonies through
// internal/passkeyauth. It is imported only from _test.go files.
package passkeytest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// Authenticator flag bits (WebAuthn §6.1).
const (
	flagUP = 0x01
	flagUV = 0x04
	flagBE = 0x08
	flagBS = 0x10
	flagAT = 0x40
)

// Credential is one passkey held by the authenticator.
type Credential struct {
	ID         []byte
	UserHandle []byte
	RPID       string
	Key        *ecdsa.PrivateKey
	// Counter is the signature counter reported on the next assertion; 0
	// keeps reporting 0 (like a synced passkey), otherwise it increments.
	Counter uint32
}

// Authenticator answers WebAuthn options. Origin is the origin the "browser"
// reports in clientDataJSON; SkipUV leaves the user-verified flag unset.
type Authenticator struct {
	Origin      string
	SkipUV      bool
	Credentials []*Credential
}

// New returns an authenticator acting from origin.
func New(origin string) *Authenticator { return &Authenticator{Origin: origin} }

type rpEntity struct {
	ID string `json:"id"`
}

type userEntity struct {
	ID string `json:"id"`
}

type creationOptions struct {
	Challenge string     `json:"challenge"`
	RP        rpEntity   `json:"rp"`
	User      userEntity `json:"user"`
}

type requestOptions struct {
	Challenge string `json:"challenge"`
	RPID      string `json:"rpId"`
}

var b64 = base64.RawURLEncoding

// Register answers creation options (the JSON a register/start endpoint
// returns as "options") with a registration response, and keeps the new
// credential. counter is the credential's initial signature counter.
func (a *Authenticator) Register(options json.RawMessage, counter uint32) (json.RawMessage, *Credential, error) {
	var opts creationOptions
	if err := json.Unmarshal(options, &opts); err != nil {
		return nil, nil, fmt.Errorf("passkeytest: decode creation options: %w", err)
	}
	userHandle, err := b64.DecodeString(opts.User.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("passkeytest: user handle: %w", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	credID := make([]byte, 16)
	if _, err := rand.Read(credID); err != nil {
		return nil, nil, err
	}
	cred := &Credential{ID: credID, UserHandle: userHandle, RPID: opts.RP.ID, Key: key, Counter: counter}

	pub := key.PublicKey
	x, y := make([]byte, 32), make([]byte, 32)
	pub.X.FillBytes(x)
	pub.Y.FillBytes(y)
	coseKey, err := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: x, -3: y})
	if err != nil {
		return nil, nil, err
	}

	authData := a.authData(opts.RP.ID, flagAT, counter)
	authData = append(authData, make([]byte, 16)...) // AAGUID: none
	authData = binary.BigEndian.AppendUint16(authData, uint16(len(credID)))
	authData = append(authData, credID...)
	authData = append(authData, coseKey...)

	attObj, err := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": authData})
	if err != nil {
		return nil, nil, err
	}
	clientData, err := a.clientData("webauthn.create", opts.Challenge)
	if err != nil {
		return nil, nil, err
	}

	resp, err := json.Marshal(map[string]any{
		"id":    b64.EncodeToString(credID),
		"rawId": b64.EncodeToString(credID),
		"type":  "public-key",
		"response": map[string]any{
			"clientDataJSON":    b64.EncodeToString(clientData),
			"attestationObject": b64.EncodeToString(attObj),
			"transports":        []string{"internal", "hybrid"},
		},
		"clientExtensionResults":  map[string]any{},
		"authenticatorAttachment": "platform",
	})
	if err != nil {
		return nil, nil, err
	}
	a.Credentials = append(a.Credentials, cred)
	return resp, cred, nil
}

// Login answers assertion options with cred (discoverable: the options carry
// no allow-list, the authenticator picks).
func (a *Authenticator) Login(options json.RawMessage, cred *Credential) (json.RawMessage, error) {
	return a.LoginAs(options, cred, cred.UserHandle)
}

// LoginAs is Login reporting an arbitrary user handle, for mismatch tests.
func (a *Authenticator) LoginAs(options json.RawMessage, cred *Credential, userHandle []byte) (json.RawMessage, error) {
	if cred == nil {
		return nil, errors.New("passkeytest: no credential")
	}
	var opts requestOptions
	if err := json.Unmarshal(options, &opts); err != nil {
		return nil, fmt.Errorf("passkeytest: decode request options: %w", err)
	}
	if cred.Counter != 0 {
		cred.Counter++
	}
	authData := a.authData(opts.RPID, 0, cred.Counter)
	clientData, err := a.clientData("webauthn.get", opts.Challenge)
	if err != nil {
		return nil, err
	}
	clientHash := sha256.Sum256(clientData)
	digest := sha256.Sum256(append(append([]byte(nil), authData...), clientHash[:]...))
	sig, err := ecdsa.SignASN1(rand.Reader, cred.Key, digest[:])
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{
		"id":    b64.EncodeToString(cred.ID),
		"rawId": b64.EncodeToString(cred.ID),
		"type":  "public-key",
		"response": map[string]any{
			"clientDataJSON":    b64.EncodeToString(clientData),
			"authenticatorData": b64.EncodeToString(authData),
			"signature":         b64.EncodeToString(sig),
			"userHandle":        b64.EncodeToString(userHandle),
		},
		"clientExtensionResults":  map[string]any{},
		"authenticatorAttachment": "platform",
	})
}

func (a *Authenticator) authData(rpID string, extra byte, counter uint32) []byte {
	rpHash := sha256.Sum256([]byte(rpID))
	flags := byte(flagUP|flagBE|flagBS) | extra
	if !a.SkipUV {
		flags |= flagUV
	}
	out := append([]byte(nil), rpHash[:]...)
	out = append(out, flags)
	return binary.BigEndian.AppendUint32(out, counter)
}

func (a *Authenticator) clientData(typ, challenge string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":        typ,
		"challenge":   challenge,
		"origin":      a.Origin,
		"crossOrigin": false,
	})
}
