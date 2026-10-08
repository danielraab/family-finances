# Spec Delta

## Purpose

Lets a signed-in user register WebAuthn passkeys on their account and then
sign in with them without an email address, while preventing a stolen session
cookie from turning into permanent access.

## ADDED Requirements

### Requirement: Passkeys belong to exactly one user

The backend SHALL store each passkey as a WebAuthn credential owned by exactly
one user, with its credential ID (globally unique), public key, signature
counter, transports, backup flags, a user-chosen name, `created_at` and
`last_used_at`. A user SHALL be able to own any number of passkeys. The WebAuthn user handle
SHALL be the user's opaque ID and SHALL NOT contain the email address.

#### Scenario: Several passkeys on one account

- **WHEN** a user registers a passkey on a laptop and another on a phone
- **THEN** both are stored for that user and both can be used to sign in

#### Scenario: Credential ID is unique

- **WHEN** a registration presents a credential ID that is already stored
- **THEN** the registration is rejected and the existing passkey is unchanged

### Requirement: Registration requires a fresh web session

`POST /api/auth/passkeys/register/start` and `.../register/finish` SHALL require
an authenticated session that is a web (cookie) session created no longer than
`AUTH_PASSKEY_REAUTH_WINDOW` (default `5m`) ago. Freshness SHALL be measured
from the session's creation time, which activity never changes, and SHALL be
checked again on finish. Otherwise the response SHALL be `403` and nothing
SHALL be stored.

#### Scenario: Fresh web session can register

- **WHEN** a user signed in 2 minutes ago through a magic link starts and
  finishes a passkey registration
- **THEN** the passkey is stored and the response is `201` with the passkey

#### Scenario: Stale session is refused

- **WHEN** a user whose session was created 20 minutes ago calls
  `POST /api/auth/passkeys/register/start`
- **THEN** the response is `403` with a re-authentication-required error and no
  challenge is issued

#### Scenario: Session goes stale mid-ceremony

- **WHEN** registration started 4 minutes into the session and finish arrives
  6 minutes into it
- **THEN** finish is refused with `403` and no passkey is stored

#### Scenario: Activity does not refresh freshness

- **WHEN** a session created an hour ago has been used continuously
- **THEN** it still cannot register a passkey

#### Scenario: Bearer session is refused

- **WHEN** a request authenticated with `Authorization: Bearer` calls either
  registration endpoint, however new the session is
- **THEN** the response is `403` and no passkey is stored

#### Scenario: Unauthenticated request

- **WHEN** either registration endpoint is called without a session
- **THEN** the response is `401`

### Requirement: Registration ceremony

Registration start SHALL return a single-use ceremony ID and WebAuthn creation
options requiring a resident (discoverable) key and user verification,
requesting no attestation, and excluding the user's existing credentials.
Finish SHALL accept the ceremony ID, the credential and an optional name. It
SHALL verify the response against the stored challenge, the relying-party ID
and origin derived from `AUTH_BASE_URL`, and user verification.

#### Scenario: Name is stored

- **WHEN** finish is called with name `"Work laptop"`
- **THEN** the stored passkey's name is `"Work laptop"`

#### Scenario: Default name

- **WHEN** finish is called without a name, or with only whitespace
- **THEN** the passkey is stored with the default name `"Passkey"`

#### Scenario: Over-long name is rejected

- **WHEN** finish is called with a name longer than 100 characters after
  trimming
- **THEN** the response is `400` and no passkey is stored

#### Scenario: Ceremony belongs to the session that started it

- **WHEN** a ceremony ID issued to one session is presented on finish by a
  different session
- **THEN** the response is `400` and no passkey is stored

#### Scenario: Ceremony is single-use and short-lived

- **WHEN** finish is called a second time with the same ceremony ID, or after
  the challenge expired
- **THEN** the response is `400` and no passkey is stored

#### Scenario: Wrong origin is rejected

- **WHEN** the credential's client data names an origin other than the
  `AUTH_BASE_URL` origin
- **THEN** the response is `400` and no passkey is stored

#### Scenario: Missing user verification is rejected

- **WHEN** the authenticator response does not assert user verification
- **THEN** the response is `400` and no passkey is stored

### Requirement: Sign-in with a discoverable passkey

`POST /api/auth/passkeys/login/start` SHALL require no authentication and SHALL
return a single-use ceremony ID and assertion options with an empty
allow-list and required user verification. `.../login/finish` SHALL verify the
assertion against the challenge, origin, stored public key and user handle.
On success it SHALL update the counter and `last_used_at`, create a web session
bound to the passkey, set `ff_session`, and respond `200` with the user.

#### Scenario: Successful passkey sign-in

- **WHEN** a person completes the browser passkey prompt with a registered
  passkey and finish is called
- **THEN** the response is `200` with the user, the `ff_session` cookie is set
  with the same attributes as other sign-ins, and the passkey's `last_used_at`
  is updated

#### Scenario: Unknown credential

- **WHEN** finish presents a credential ID that is not stored (for example, a
  passkey deleted on the server but still on the device)
- **THEN** the response is `401` with a generic passkey sign-in failure and no
  session is created

#### Scenario: Invalid signature or mismatched user handle

- **WHEN** the assertion signature does not verify, or the user handle does not
  match the credential's owner
- **THEN** the response is `401` and no session is created

#### Scenario: Counter regression

- **WHEN** the stored signature counter and the presented counter are both
  non-zero and the presented one is not greater than the stored one
- **THEN** the response is `401` and no session is created

#### Scenario: Disabled or deleted account

- **WHEN** a valid passkey belongs to a disabled or soft-deleted user
- **THEN** the response is `403` and no session is created

#### Scenario: Passkeys never create or link accounts

- **WHEN** any passkey sign-in completes
- **THEN** no user and no identity is created or linked; signup, invite and
  domain allow-list settings do not apply

#### Scenario: Login ceremony is single-use

- **WHEN** login finish is called twice with the same ceremony ID
- **THEN** the second call is rejected with `400`

### Requirement: Passkey-created sessions count as fresh

A session created by passkey sign-in SHALL record which passkey created it, and
SHALL be subject to the same freshness rule as any other web session. A
passkey sign-in therefore counts as re-authentication for registering a
further passkey.

#### Scenario: Re-authenticating with a passkey

- **WHEN** a user signs in with an existing passkey and, within the
  re-authentication window, registers a second one
- **THEN** the registration succeeds

### Requirement: Listing passkeys

`GET /api/auth/passkeys` SHALL require authentication (any session type) and
SHALL return only the caller's passkeys, oldest first, each with `id`, `name`,
`created_at`, `last_used_at` (null if never used), `backed_up`, and `current`.
`current` is true exactly when the requesting session was created by that
passkey. Key material SHALL NOT be returned.

#### Scenario: Own passkeys only

- **WHEN** two users each have passkeys and one calls `GET /api/auth/passkeys`
- **THEN** only that user's passkeys are returned

#### Scenario: Current passkey is marked

- **WHEN** a user signed in with passkey A lists their passkeys A and B
- **THEN** A has `current: true` and B has `current: false`

#### Scenario: No public key in the response

- **WHEN** the list response is inspected
- **THEN** it contains no public key, credential ID bytes or signature counter

### Requirement: Deleting a passkey ends that passkey's sessions

`DELETE /api/auth/passkeys/{id}` SHALL require authentication (any session type,
any session age) and SHALL delete the caller's passkey, responding `204`. The
same operation SHALL delete every session that passkey created, and SHALL NOT
touch any other session. Deleting the passkey behind the current session SHALL
also clear the `ff_session` cookie. Another user's passkey or an unknown ID
SHALL yield `404`.

#### Scenario: Removal is always allowed

- **WHEN** a user whose session is hours old deletes one of their passkeys
- **THEN** the response is `204` with no re-authentication required

#### Scenario: Only that passkey's sessions end

- **WHEN** a user has session S1 from passkey A, S2 from passkey B and S3 from
  a magic link, and deletes passkey A from S3
- **THEN** S1 is revoked and S2 and S3 keep working

#### Scenario: Deleting the current session's passkey

- **WHEN** a user signed in with passkey A deletes passkey A
- **THEN** the response is `204`, clears the `ff_session` cookie, and the next
  request with that session is `401`

#### Scenario: Cannot delete someone else's passkey

- **WHEN** a user calls `DELETE /api/auth/passkeys/{id}` with another user's
  passkey ID
- **THEN** the response is `404` and nothing is deleted

#### Scenario: Deleted passkey cannot sign in

- **WHEN** a passkey has been deleted and the device still offers it
- **THEN** a sign-in attempt with it is rejected with `401`

### Requirement: Session freshness is reported to the client

`GET /api/auth/me` (and `PATCH /api/auth/me`) SHALL include a `session` object
for the requesting session with `created_at`, `client` (`web` or `api`), and
`passkey_registration_until`: creation time plus `AUTH_PASSKEY_REAUTH_WINDOW`
for a web session, or `null` for an API session.

#### Scenario: Fresh web session

- **WHEN** a user whose web session was created at 10:00 calls
  `GET /api/auth/me` with the default window
- **THEN** `session.passkey_registration_until` is 10:05 on that date

#### Scenario: API session

- **WHEN** `GET /api/auth/me` is called with a bearer token
- **THEN** `session.client` is `api` and `session.passkey_registration_until`
  is `null`
