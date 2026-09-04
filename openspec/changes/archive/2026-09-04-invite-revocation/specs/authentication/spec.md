## MODIFIED Requirements

### Requirement: Invites

Any authenticated user SHALL be able to create an invite for an email address
via `POST /api/auth/invites`. Invites SHALL be stored in their own table with
the inviting user, the target email, a hashed single-use token, and an expiry
(`AUTH_INVITE_TTL`). An invite email SHALL be sent over SMTP with an acceptance
link.

Inviting SHALL be enabled whenever `AUTH_SIGNUP_ENABLED` is true. When signup is
false, inviting SHALL be governed by `AUTH_INVITE_ENABLED`; when both are false,
no new accounts can be created at all. Accepting a valid invite SHALL create
the account even when signup is disabled, and SHALL bypass the email-domain
allow-list. The invite token SHALL be single-use and SHALL be rejected once
consumed, expired, or revoked (see `user-administration` for revocation).

#### Scenario: Authenticated user invites someone

- **WHEN** an authenticated user posts a valid email to `/api/auth/invites`
  and inviting is enabled
- **THEN** an invite row is created and an acceptance email is sent

#### Scenario: Invite acceptance creates an account despite disabled signup

- **WHEN** `AUTH_SIGNUP_ENABLED=false` and a person follows a valid invite link
- **THEN** an account is created for that address

#### Scenario: Invite bypasses the domain allow-list

- **WHEN** an invited address is outside `AUTH_ALLOWED_EMAIL_DOMAINS` and the
  invite is accepted
- **THEN** the account is created

#### Scenario: Inviting can be disabled only when signup is disabled

- **WHEN** `AUTH_SIGNUP_ENABLED=true` and `AUTH_INVITE_ENABLED=false`
- **THEN** inviting is still enabled

#### Scenario: Fully closed instance

- **WHEN** `AUTH_SIGNUP_ENABLED=false` and `AUTH_INVITE_ENABLED=false`
- **THEN** `POST /api/auth/invites` does not send an invite and no new account
  can be created

#### Scenario: Invite token is single-use

- **WHEN** an invite acceptance link is followed a second time
- **THEN** it is rejected

#### Scenario: Revoked invite cannot be accepted

- **WHEN** an invite has been revoked and its acceptance link is followed
- **THEN** the request is rejected and no account is created
