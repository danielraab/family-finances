# Spec Delta

## ADDED Requirements

### Requirement: Login route offers passkey sign-in

When the browser exposes WebAuthn (`window.PublicKeyCredential` exists), `/login`
SHALL render a "Sign in with a passkey" button above the email form, with the
OIDC control if one exists, and an "or" divider before the email form. It SHALL
have no email field of its own. Activating it SHALL call
`POST /api/auth/passkeys/login/start`, open the browser's passkey prompt with
the returned options, and send the result to `.../login/finish`. Strings SHALL
exist in `en` and `de`.

#### Scenario: Successful passkey sign-in

- **WHEN** the visitor activates "Sign in with a passkey", picks a passkey and
  the finish call returns `200`
- **THEN** the shared auth state becomes authenticated with the returned user
  and the client navigates to `/`

#### Scenario: Browser without WebAuthn

- **WHEN** `/login` renders in a browser without `window.PublicKeyCredential`
- **THEN** no passkey button is shown and the rest of the view is unchanged

#### Scenario: Visitor cancels the prompt

- **WHEN** the visitor dismisses the browser's passkey prompt
- **THEN** no error message is shown and the view returns to its idle state

#### Scenario: Passkey rejected by the server

- **WHEN** login finish responds `401` or `403`
- **THEN** the view shows a "passkey sign-in failed — try another method"
  message and the email form remains usable

#### Scenario: Email form still works alongside

- **WHEN** the passkey button is shown
- **THEN** the email form and its behaviour are unchanged
