# Spec Delta

## ADDED Requirements

### Requirement: Passkeys tab

The settings page SHALL offer every authenticated visitor a **Passkeys** tab at
`/settings/passkeys`, between Profile and My Invitations. On mount it SHALL
load `GET /api/auth/passkeys` and list each passkey's name, date added and last
use ("never" when null), marking the passkey of the current session. With no
passkeys it SHALL say so plainly. Labels SHALL come from `settings.passkeys.*`
in `en` and `de`.

#### Scenario: Tab is visible to everyone

- **WHEN** any authenticated visitor opens `/settings`
- **THEN** the tab list includes "Passkeys" between Profile and My Invitations

#### Scenario: Passkeys are listed

- **WHEN** a visitor with two passkeys opens the Passkeys tab
- **THEN** both are listed with name, date added and last use, and the one
  that created the current session is marked as "this session"

#### Scenario: Empty state

- **WHEN** a visitor without passkeys opens the tab
- **THEN** the tab states that no passkeys are registered yet

### Requirement: Adding a passkey requires a fresh sign-in

The tab SHALL always explain that adding a passkey needs a sign-in from the
last few minutes. While `session.passkey_registration_until` from
`GET /api/auth/me` is in the future, an enabled "Add passkey" control SHALL ask
for an optional name and run registration. Otherwise the control SHALL be
disabled and a "Sign in again" control SHALL be offered. The state SHALL switch
to stale when the deadline passes while the tab is open.

#### Scenario: Fresh session adds a passkey

- **WHEN** a visitor who signed in a minute ago activates "Add passkey", enters
  "Laptop" and completes the browser prompt
- **THEN** register start and finish are called and the new "Laptop" passkey
  appears in the list

#### Scenario: Stale session sees the requirement

- **WHEN** a visitor whose session is older than the window opens the tab
- **THEN** "Add passkey" is disabled, the fresh-sign-in notice is shown, and a
  "Sign in again" control is offered

#### Scenario: Window elapses on screen

- **WHEN** the tab is open and `passkey_registration_until` passes
- **THEN** the tab switches to the stale state without a reload

#### Scenario: Server reports re-authentication required

- **WHEN** register start or finish responds `403`
- **THEN** the tab switches to the stale state and shows the fresh-sign-in
  notice

#### Scenario: Visitor cancels the browser prompt

- **WHEN** the visitor dismisses the browser's passkey prompt during
  registration
- **THEN** nothing is added and no error message is shown

#### Scenario: API-session visitor

- **WHEN** `session.passkey_registration_until` is `null`
- **THEN** "Add passkey" is disabled with the fresh-sign-in notice

### Requirement: Signing in again from the Passkeys tab

"Sign in again" SHALL offer a passkey sign-in in place when the visitor already
has at least one passkey. On success the tab SHALL refresh the auth state and
become fresh without leaving `/settings/passkeys`. Otherwise, or as an
alternative, it SHALL sign the visitor out and navigate to `/login`.

#### Scenario: Re-authenticate with an existing passkey

- **WHEN** a visitor with a passkey and a stale session activates "Sign in
  again" and completes the passkey prompt
- **THEN** the session is replaced, the tab becomes fresh, and "Add passkey" is
  enabled

#### Scenario: Re-authenticate without a passkey

- **WHEN** a visitor without passkeys activates "Sign in again"
- **THEN** the client signs out and navigates to `/login`

### Requirement: Removing a passkey

Each listed passkey SHALL have a Remove control behind the same confirmation
pattern used elsewhere in Settings, available regardless of session age. The
confirmation SHALL warn that sessions signed in with that passkey will be
signed out, and more strongly when it is the current session's passkey. After
removing the current session's passkey, the client SHALL clear its auth state
and navigate to `/login`.

#### Scenario: Removing another passkey

- **WHEN** a visitor confirms removal of a passkey that is not the current one
- **THEN** `DELETE /api/auth/passkeys/{id}` is called and the row disappears
  while the visitor stays signed in

#### Scenario: Removing the current passkey

- **WHEN** a visitor signed in with passkey A confirms removal of A
- **THEN** the confirmation warned about signing out, the passkey is deleted,
  and the client navigates to `/login`

#### Scenario: Removal on a stale session

- **WHEN** a visitor whose session is hours old removes a passkey
- **THEN** removal proceeds without a re-authentication step
