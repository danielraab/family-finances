# Spec Delta

## ADDED Requirements

### Requirement: Passkey rows show the provider

Each row on the Passkeys tab SHALL show the passkey's provider icon (the
light-theme or dark-theme variant to match the active theme, either one if
only one exists) in place of the generic key glyph, and the provider name in
the row's secondary line when it differs from the passkey's own name. A
passkey whose `provider` is `null` SHALL render exactly as before.

#### Scenario: Provider with icons

- **WHEN** a listed passkey has a provider named "1Password" with both icons
- **THEN** the row shows the light icon in light theme and the dark icon in
  dark theme, and "1Password" in the secondary line

#### Scenario: Name equals provider

- **WHEN** a passkey named "Apple Passwords" has provider "Apple Passwords"
- **THEN** the provider name is not repeated in the secondary line

#### Scenario: No provider

- **WHEN** a listed passkey's `provider` is `null`
- **THEN** the row shows the generic key glyph and no provider name
