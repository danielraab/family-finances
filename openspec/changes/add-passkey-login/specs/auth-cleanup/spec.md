# Spec Delta

## ADDED Requirements

### Requirement: Cleanup removes expired passkey challenges

Each cleanup pass SHALL also delete passkey ceremony challenges (registration
and sign-in) whose expiry has passed. Unexpired challenges SHALL be kept so an
in-progress ceremony can still finish. Passkeys themselves SHALL never be
deleted by the cleanup job, however long unused.

#### Scenario: Abandoned ceremony is removed

- **WHEN** a cleanup pass runs and a passkey challenge is past its expiry
- **THEN** that challenge is deleted

#### Scenario: In-progress ceremony survives

- **WHEN** a cleanup pass runs while a passkey challenge is still unexpired
- **THEN** the challenge is kept and its ceremony can still be finished

#### Scenario: Unused passkeys are kept

- **WHEN** a cleanup pass runs and a passkey has never been used
- **THEN** the passkey is kept
