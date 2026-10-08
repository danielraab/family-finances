# Design

## Context

- `webauthn_credentials.aaguid` is already stored at registration
  (`add-passkey-login`). Registration requests `attestation: "none"`.
  Browsers and authenticators generally still report the AAGUID in that
  case, but some report all zeros, so "no provider" must be a normal state.
- The upstream list is a flat JSON object: lowercase AAGUID →
  `{name, icon_light?, icon_dark?}`, where the icons are
  `data:image/svg+xml;base64,…` URIs. The README warns it may be retired by
  being emptied to `{}`.

## Goals / Non-Goals

**Goals:** a human-recognisable provider per passkey, with no network
dependency and no new runtime dependency.

**Non-Goals:**

- FIDO MDS, and attestation verification or authenticator policy (AAGUIDs
  are self-asserted without attestation and are used for display only, never
  for trust).
- Automatic refresh of the list.

## Decisions

### D1. Embed a snapshot in a new `internal/aaguid` package

- `//go:embed aaguid.json` holds the snapshot, parsed once (`sync.Once`)
  into a map.
- `Lookup(aaguid []byte) (Provider, bool)`: false for a nil or 16-zero
  AAGUID, for an unknown one, or for a wrong length.
- Icons are kept only when they are `data:image/svg+xml;base64,` URIs, so
  the frontend never loads an external URL from this data. An `<img>` also
  cannot run script from an SVG.
- **Refresh is manual**, by `curl`-ing the upstream file over the snapshot
  (the command is in the package doc). It deliberately is not a
  `go:generate` step: CI re-runs `go generate` and fails on any diff, and
  upstream changes on its own schedule.
- A unit test asserts the snapshot is non-empty and parses, catching an
  upstream retirement (`{}`) being copied in.
- *Alternatives:*
  - Fetching at runtime: rejected for privacy and offline reasons, and it
    adds a failure mode.
  - Bundling in the frontend: rejected because it puts ~380 KB in the client
    bundle for every visitor, versus a few hundred bytes per passkey in the
    API response.

### D2. Wiring into `internal/auth`

- `auth` declares `ProviderLookup interface { Lookup([]byte) (PasskeyProvider, bool) }`
  and takes it through `auth.WithPasskeyProviders`. It does not import
  `internal/aaguid`, mirroring `LanguageLookup`.
- `passkeyInfo` becomes a `Service` method that fills
  `Provider *PasskeyProvider` (`json:"provider"`).
- When `FinishPasskeyRegistration` gets an empty name and the AAGUID resolves,
  the stored name is the provider name (truncated to the 100-character limit),
  else "Passkey". The name is stored, not derived on read, so it stays stable
  if the list later renames the provider.

### D3. Frontend

- The row shows `provider.icon_light` with `dark:hidden` and
  `provider.icon_dark` with `hidden dark:block`, the same class-based theming
  as the rest of the app. If either is missing it uses the other, and with
  neither it shows the existing `KeyRound` glyph.
- The provider name goes in the secondary line ("iCloud Keychain · Added …")
  when it differs from the passkey's own name.

## Risks / Trade-offs

- **[Stale snapshot: a new provider shows no icon or name]** → It degrades to
  today's behaviour. The refresh is a one-line command.
- **[Upstream retirement]** → The snapshot keeps working. The non-empty test
  guards against copying in `{}`.
- **[Licensing of vendor icons]** → Upstream states this exact use. If that
  is ever a concern, dropping the icon fields leaves names only, with no API
  change because the icons are optional.
