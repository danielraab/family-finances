## Context

`internal/auth` already implements the full OIDC authorization-code flow
(`GET /api/auth/oidc/start` → provider → `GET /api/auth/oidc/callback` →
session). `main.buildAuth` constructs the OIDC client only when
`cfg.OIDC.Issuer != ""`; otherwise the service's `oidc` field is nil and the
routes return `ErrOIDCNotConfigured`. The service already exposes a
config-derived boolean accessor, `InviteEnabled() bool`.

The frontend is a static bundle with no build-time knowledge of backend config.
`frontend/src/routes/login.tsx` renders only the magic-link form; the
`web-client-auth` spec explicitly puts provider sign-in "out of scope". The
`adopt-openapi-contract` change (a prerequisite) introduces the OpenAPI document,
the committed generated types, and an `openapi-fetch` typed client.

## Goals / Non-Goals

**Goals:**

- A configurable, human-facing label for the OIDC sign-in button.
- A way for the static client to learn, at runtime, whether to show that button.
- The button on `/login`, above the email field, when OIDC is available.

**Non-Goals:**

- More than one OIDC provider (still out of scope, as in `authentication`).
- Any change to the OIDC flow itself, its callbacks, or session handling.
- Post-login redirect handling on `/login` (no `redirect_to` param today).
- A generic "capabilities" endpoint beyond the OIDC affordance — the shape is
  left open to grow, but this change only populates `oidc`.
- Branding beyond a text label (no provider logos/icons).

## Decisions

### D1 — New endpoint `GET /api/auth/config`, unauthenticated

The login page is viewed by anonymous visitors, so the discovery endpoint must
work without a session. `GET /api/auth/me` is `401` when anonymous and cannot
carry this. The endpoint lives on the existing `auth.Handler` mux (no new
package, no new dependency).

Response shape:

```json
{ "oidc": { "label": "Continue with Google", "start_path": "/api/auth/oidc/start" } }
```

or `{ "oidc": null }` when OIDC is not available. An object-or-null field (rather
than a bare boolean) lets a future method be added as a sibling key and lets the
client treat "render this button linking here with this text" as one unit.

- **Alternatives considered:**
  - *Extend `GET /api/auth/me`*: breaks on the anonymous case.
  - *Static build-time flag / env baked into the bundle*: the bundle is
    provider-agnostic and served by the same binary for any config; a build-time
    flag would require rebuilding the image per deployment.
  - *A top-level `GET /api/config`*: broader than needed now; `auth/config`
    keeps it in the package that owns the knowledge and can be generalised later.

### D2 — `ok` requires issuer AND client id

`auth.Service.OIDCLogin()` returns `ok == true` only when the service has a
non-nil `oidc` client. `main.buildAuth` will be tightened to build the client
only when both `OIDC_ISSUER` and `OIDC_CLIENT_ID` are set — an issuer without a
client id is a misconfiguration that currently discovers successfully but fails
at token exchange. Reporting `ok` for that state would show a button that always
errors.

- **Alternative considered:** gate on issuer alone (matches today's
  `buildAuth`). Rejected — it advertises a broken button.

### D3 — `OIDC_LABEL`, default `Single sign-on`

A button needs text. The default is provider-neutral; instances set
`OIDC_LABEL="Continue with Okta"` etc. Loaded in `internal/config` as
`OIDCConfig.Label`, passed via `auth.Params.OIDCLabel`, surfaced by
`Service.OIDCLogin()`. `.env.example` documents it in the `OIDC_*` block.

### D4 — Frontend fetches the config in the login route, not app-wide

`login.tsx` fetches `GET /api/auth/config` in a `useEffect` via the typed client,
local state, no global provider. It is the only consumer. Folding it into
`AuthProvider` would touch a spec ("resolved once", "MUST NOT poll") and add a
second concern to a deliberately minimal provider for no present benefit. If a
second consumer appears later, a small `useAuthConfig` hook can be extracted
then.

- While the config request is in flight the page renders the email form as
  today (no layout jump waiting on an optional button); the OIDC button appears
  when the response arrives. A failed/`5xx` config request → no button, form
  still fully usable.

### D5 — The OIDC control is a link, not a `fetch`

`/api/auth/oidc/start` issues a `302` to the provider — a full-page navigation.
The control is an `<a href={start_path}>` styled as a button, not a JS handler.
This is also why `adopt-openapi-contract` marks `oidc/start` as a browser-flow
operation excluded from the generated client: only `/api/auth/config` is called
via `fetch`.

### D6 — Layout: button on top, then a divider, then the form

Order on `/login`: provider button → `—— or ——` divider → existing
`Email` field + "Send me a sign-in link". When `oidc` is null, none of the
button/divider render and the page is byte-for-byte the current one.

## Risks / Trade-offs

- **Unauthenticated endpoint leaks that OIDC is configured and its label** →
  acceptable: the label is meant to be shown to anonymous visitors, and
  `oidc/start` already 302s unauthenticated. No secret (issuer URL, client id)
  is exposed.
- **`buildAuth` gate change (issuer+id) alters current behaviour** → today an
  issuer-only config builds a client whose flow fails later; after this it
  simply reports OIDC unavailable. Strictly an improvement; note it in the
  change and `AGENTS.md`.
- **Config request adds a round trip on `/login`** → one small GET, only on the
  login route, form is usable before it resolves. Negligible.
- **Spec dependency ordering** → this change must land after
  `adopt-openapi-contract`; `GET /api/auth/config` is added to
  `openapi/openapi.yaml` and the client regenerated as part of this change.

## Migration Plan

1. `adopt-openapi-contract` merged first.
2. Backend: `OIDCConfig.Label`, `Params.OIDCLabel`, `Service.OIDCLogin()`,
   tighten `buildAuth` to issuer+id, add `GET /api/auth/config` handler + tests,
   update `.env.example`.
3. Spec: add `GET /api/auth/config` (schema + 200) to `openapi/openapi.yaml`;
   sync `backend/openapi.yaml`; regenerate `frontend/src/api/schema.d.ts`.
4. Frontend: `login.tsx` fetches config via the typed client, renders button +
   divider above the form when present.
5. Docs.

**Rollback:** reverting the commit removes the endpoint, the env var, and the
button. No data or persistent state involved.

## Open Questions

- Exact key name: `start_path` vs `start_url` (relative either way) — settle in
  implementation; `start_path` chosen here.
- Whether `OIDC_LABEL` should have no default and instead fall back to a
  hard-coded string in the handler — equivalent; env default chosen for
  visibility in `.env.example`.
