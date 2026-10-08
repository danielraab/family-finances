// Package aaguid names the provider of a passkey from its authenticator's
// AAGUID, using an embedded snapshot of the community list at
// https://github.com/passkeydeveloper/passkey-authenticator-aaguids — a list
// meant for exactly this: naming passkeys in account-settings UIs. The result
// is for display only; an AAGUID is self-asserted and never a trust signal.
//
// The snapshot is refreshed by hand (deliberately not a go:generate step: CI
// re-runs go generate and fails on drift, and upstream changes on its own
// schedule):
//
//	curl -fsSL -o internal/aaguid/aaguid.json \
//	  https://raw.githubusercontent.com/passkeydeveloper/passkey-authenticator-aaguids/main/aaguid.json
//
// Upstream retires the list by emptying it to {}; the package test fails if
// such a snapshot is copied in.
package aaguid

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed aaguid.json
var snapshot []byte

// Provider is a passkey provider as shown to the user. The icons are SVG
// data URIs ("" when the list has none) for light and dark backgrounds.
type Provider struct {
	Name      string
	IconLight string
	IconDark  string
}

type entry struct {
	Name      string `json:"name"`
	IconLight string `json:"icon_light"`
	IconDark  string `json:"icon_dark"`
}

var (
	loadOnce  sync.Once
	providers map[string]Provider
	loadErr   error
)

func load() {
	var raw map[string]entry
	if err := json.Unmarshal(snapshot, &raw); err != nil {
		loadErr = fmt.Errorf("aaguid: parse embedded list: %w", err)
		return
	}
	providers = make(map[string]Provider, len(raw))
	for id, e := range raw {
		if strings.TrimSpace(e.Name) == "" {
			continue
		}
		providers[strings.ToLower(id)] = Provider{
			Name:      e.Name,
			IconLight: svgDataURI(e.IconLight),
			IconDark:  svgDataURI(e.IconDark),
		}
	}
}

// svgDataURI keeps an icon only if it is an inline SVG data URI, so the
// frontend never loads a URL taken from this list.
func svgDataURI(s string) string {
	if strings.HasPrefix(s, "data:image/svg+xml;base64,") {
		return s
	}
	return ""
}

// Lookup returns the provider for a 16-byte AAGUID. It reports false for an
// absent, malformed or all-zero AAGUID (authenticators and browsers that
// don't disclose one) and for one the list doesn't know.
func Lookup(id []byte) (Provider, bool) {
	if len(id) != 16 || isZero(id) {
		return Provider{}, false
	}
	loadOnce.Do(load)
	p, ok := providers[format(id)]
	return p, ok
}

// Count is the number of providers in the embedded list, and its load
// error, for tests and startup diagnostics.
func Count() (int, error) {
	loadOnce.Do(load)
	return len(providers), loadErr
}

func isZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}

// format renders an AAGUID in the list's key form:
// lowercase 8-4-4-4-12 hex.
func format(b []byte) string {
	const hex = "0123456789abcdef"
	var sb strings.Builder
	for i, c := range b {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			sb.WriteByte('-')
		}
		sb.WriteByte(hex[c>>4])
		sb.WriteByte(hex[c&0x0f])
	}
	return sb.String()
}
