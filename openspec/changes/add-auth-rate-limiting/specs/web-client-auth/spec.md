# Spec Delta

## ADDED Requirements

### Requirement: Login view explains rate limiting

When a sign-in request from `/login` (such as `POST /api/auth/email/start`)
receives `429`, the view SHALL keep the form intact and show a non-blocking
"too many attempts — please wait a moment and try again" message, distinct
from the generic "please try again" failure message. Strings SHALL exist in
both the `en` and `de` locale files.

#### Scenario: Email start is throttled

- **WHEN** the visitor submits an address and the backend responds `429`
- **THEN** the form stays visible with the "too many attempts" message and no
  confirmation panel is shown

#### Scenario: Generic failure keeps its own message

- **WHEN** the request fails with a network error or `5xx`
- **THEN** the existing "please try again" message is shown, not the
  rate-limit message
