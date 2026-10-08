# Proposal

## Why

The Settings → Passkeys list shows only the name the user typed, or "Passkey"
by default. With several passkeys it's hard to tell which is the iCloud
Keychain one and which lives in 1Password or a security key, and so which to
remove when a device or password manager is retired. Every passkey already
stores its authenticator's AAGUID, which identifies the provider model.

## What Changes

- The backend maps a passkey's AAGUID to a **provider** (a display name, plus
  light/dark icons when known). It uses an embedded snapshot of the community
  list `passkeydeveloper/passkey-authenticator-aaguids` (`aaguid.json`), which
  exists for exactly this purpose: naming passkeys in account-settings UIs.
  There is no runtime fetch.
- `GET /api/auth/passkeys` and the register-finish response carry
  `provider: { name, icon_light?, icon_dark? } | null`. It is `null` for an
  all-zero AAGUID (some authenticators and browsers report none) or one the
  list doesn't know.
- A passkey registered without a name is named after its provider (e.g.
  "Apple Passwords") when one is known, falling back to "Passkey".
- The Passkeys tab shows each passkey's provider icon (theme-aware) and name
  next to the user-chosen name.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `passkeys` (introduced by `add-passkey-login`): passkeys report their
  provider; the default name uses it.
- `web-client-settings`: passkey rows show the provider.

## Impact

- **Backend:** new leaf package `internal/aaguid` (embedded `aaguid.json` and
  a lookup), wired into `auth` through a small interface option; `PasskeyInfo`
  gains `provider`. No schema change, since the AAGUID is already stored.
- **API:** `Passkey` schema gains `provider`; both generated copies are
  regenerated.
- **Frontend:** the Passkeys tab row.
- **Data:** ~380 KB embedded JSON, mostly SVG icons. The upstream repo
  publishes no license file; it states its purpose as naming passkeys in RP
  account UIs. Icons are vendor marks shown only next to the vendor's own
  passkeys.
- **Ordering:** archive after `add-passkey-login`, whose `passkeys` spec this
  modifies.
