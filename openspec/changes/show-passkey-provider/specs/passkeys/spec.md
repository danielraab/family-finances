# Spec Delta

## ADDED Requirements

### Requirement: Passkeys report their provider

The backend SHALL map a passkey's stored AAGUID to a provider using an
embedded snapshot of the community AAGUID list, without any network request.
Every passkey in `GET /api/auth/passkeys` and the register-finish response
SHALL carry `provider`: an object with `name` and optional `icon_light` and
`icon_dark` (SVG data URIs only), or `null` when the AAGUID is all zeros,
missing or unknown.

#### Scenario: Known provider

- **WHEN** a passkey's AAGUID is `ea9b8d66-4d01-1d21-3ce4-b6b48cb575d4`
- **THEN** its `provider.name` is `"Google Password Manager"` and its icons
  are SVG data URIs

#### Scenario: Zero AAGUID

- **WHEN** a passkey was registered by an authenticator that reported an
  all-zero AAGUID
- **THEN** its `provider` is `null`

#### Scenario: Unknown AAGUID

- **WHEN** a passkey's AAGUID is not in the embedded list
- **THEN** its `provider` is `null`

#### Scenario: Provider is display-only

- **WHEN** a passkey signs in
- **THEN** its provider plays no part in verification or authorization

### Requirement: Default passkey name follows the provider

When register-finish is called without a name, or with only whitespace, the
backend SHALL store the provider's name (truncated to 100 characters) when the
AAGUID resolves to a provider, and `"Passkey"` otherwise. A supplied name SHALL
always be stored as given.

#### Scenario: Unnamed passkey from a known provider

- **WHEN** a passkey whose AAGUID resolves to "Apple Passwords" is registered
  without a name
- **THEN** it is stored with the name `"Apple Passwords"`

#### Scenario: Unnamed passkey from an unknown provider

- **WHEN** a passkey with an all-zero AAGUID is registered without a name
- **THEN** it is stored with the name `"Passkey"`

#### Scenario: Explicit name wins

- **WHEN** a passkey from a known provider is registered with the name
  `"Work laptop"`
- **THEN** it is stored with the name `"Work laptop"`
