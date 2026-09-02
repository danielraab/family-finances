## MODIFIED Requirements

### Requirement: Login route starts the magic-link flow

The route `/login` SHALL render a view with a single email input and a submit
control. On submit the client SHALL validate that the value looks like an email
address and then call `POST /api/auth/email/start` with `{ "email": <value> }`.
Regardless of the response body (the endpoint always returns `200`), the view
SHALL then replace the form with a confirmation panel stating that a sign-in
link has been sent to that address, and offering a way to enter a different
address. A network or `5xx` failure on the request SHALL show a non-blocking
"please try again" message with the form intact.

The view SHALL NOT reveal whether an account exists for the address. It SHALL
NOT offer a password field.

The view SHALL also offer OIDC sign-in when the backend reports it is available.
On mount the client SHALL call `GET /api/auth/config` through the typed API
client. When the response's `oidc` field is a non-null object, the view SHALL
render, **above** the email field, a control labelled with `oidc.label` that is a
link to `oidc.start_path` (a full-page navigation, not a `fetch`), followed by a
visual "or" divider separating it from the email form. When `oidc` is `null`, or
the request fails, the view SHALL render exactly the email form described above
with no OIDC control and no divider. While the request is in flight the email
form SHALL already be usable.

#### Scenario: Submitting an address

- **WHEN** the visitor enters `person@example.com` on `/login` and submits
- **THEN** the client sends `POST /api/auth/email/start` with that email
- **AND** the view shows a "check your inbox — link sent to person@example.com"
  confirmation with an option to use a different address

#### Scenario: Invalid address is caught client-side

- **WHEN** the visitor submits a value that is not a valid email address
- **THEN** no request is sent and the field shows a validation message

#### Scenario: Request failure is recoverable

- **WHEN** the `POST /api/auth/email/start` request fails or returns `5xx`
- **THEN** the form remains visible with a "please try again" message

#### Scenario: No account enumeration

- **WHEN** the visitor submits an address with no account
- **THEN** the confirmation panel is identical to the one shown for an address
  that does have an account

#### Scenario: OIDC provider button shown above the form when available

- **WHEN** `/login` mounts and `GET /api/auth/config` returns
  `{ "oidc": { "label": "Continue with Google", "start_path": "/api/auth/oidc/start" } }`
- **THEN** the view renders a "Continue with Google" link to
  `/api/auth/oidc/start` above the email field, with an "or" divider between it
  and the email form

#### Scenario: No OIDC button when unavailable

- **WHEN** `/login` mounts and `GET /api/auth/config` returns `{ "oidc": null }`
  or the request fails
- **THEN** the view shows only the email form, with no OIDC control and no
  divider

#### Scenario: OIDC control is a navigation, not a fetch

- **WHEN** the visitor activates the OIDC sign-in control
- **THEN** the browser performs a full-page navigation to
  `/api/auth/oidc/start` (the client does not `fetch` it)
