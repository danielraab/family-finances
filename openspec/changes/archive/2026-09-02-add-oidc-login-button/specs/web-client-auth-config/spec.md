## ADDED Requirements

### Requirement: Client discovers available sign-in methods at runtime

Because the web client is a static bundle served by the same binary for any
backend configuration, it SHALL NOT assume which sign-in methods are available.
It SHALL learn them at runtime by calling `GET /api/auth/config` through the
typed API client. That endpoint requires no authentication and returns a JSON
object whose `oidc` field is either `null` or `{ label, start_path }`.

The client SHALL treat a non-null `oidc` object as "offer an OIDC sign-in
control labelled `label` that navigates to `start_path`", and a `null` `oidc`
(or any failed request) as "OIDC sign-in is not offered". The client SHALL NOT
require any other field to decide this, so the response can grow sibling keys
for future methods without a client change.

#### Scenario: Config drives the OIDC affordance

- **WHEN** the client needs to decide whether to show an OIDC sign-in control
- **THEN** it uses the `oidc` field of `GET /api/auth/config`, showing the
  control only when that field is a non-null object

#### Scenario: Failure is treated as "not available"

- **WHEN** `GET /api/auth/config` fails with a network error or `5xx`
- **THEN** the client behaves as if `oidc` were `null` and still renders the
  rest of the sign-in view

#### Scenario: Call is credentialed and same-origin

- **WHEN** the client calls `GET /api/auth/config`
- **THEN** the request uses a relative `/api/...` path through the typed client
  with credentials included, consistent with every other backend call
