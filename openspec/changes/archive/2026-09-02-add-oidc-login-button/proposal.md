## Why

The backend already supports one OIDC provider, but the login page only offers
the email magic-link form — a static bundle has no way to know whether OIDC is
configured or what to call it. Instances that run with an OIDC provider want a
labelled "sign in with …" button on `/login`.

## What Changes

- Add `OIDC_LABEL` (env, `OIDC_*` group) — the text shown on the OIDC sign-in
  button, e.g. `Continue with Google`. Default: `Single sign-on`.
- Add `auth.Service.OIDCLogin() (label string, ok bool)` — `ok` is true only
  when an OIDC client was actually built (issuer **and** client id set); `label`
  is the configured `OIDC_LABEL`.
- Add `GET /api/auth/config` — an unauthenticated JSON endpoint returning the
  OIDC login affordance the client should show, or that it is absent:
  `{ "oidc": { "label": "...", "start_path": "/api/auth/oidc/start" } }` or
  `{ "oidc": null }`. Documented in `openapi/openapi.yaml`; consumed through the
  generated typed client.
- Login page: when `GET /api/auth/config` reports OIDC is available, render the
  provider button **above** the email field, followed by an `—— or ——` divider,
  then the existing magic-link form. The button is a link to
  `/api/auth/oidc/start` (full-page navigation). When OIDC is absent the page is
  unchanged.

## Capabilities

### New Capabilities

- `web-client-auth-config`: the web client discovers, at runtime, which sign-in
  methods the backend offers (currently: whether a labelled OIDC provider is
  available) and renders the login page accordingly.

### Modified Capabilities

- `authentication`: the single configured OIDC provider gains a human-facing
  display label (`OIDC_LABEL`), and the backend exposes an unauthenticated
  endpoint reporting whether OIDC sign-in is available and its label.
- `web-client-auth`: the `/login` route offers the configured OIDC provider
  above the magic-link form; the previous "OIDC / provider sign-in is out of
  scope" exclusion is lifted.

## Impact

- **Depends on** `adopt-openapi-contract` (typed client + spec workflow).
- **Backend**: `internal/config` (`OIDCConfig.Label`), `internal/auth`
  (`Params.OIDCLabel`, `Service.OIDCLogin()`, `handler.go` route + `_test.go`),
  `backend/.env.example`, `openapi/openapi.yaml` (+`GET /api/auth/config`).
- **Frontend**: `src/routes/login.tsx` (fetch config, conditional button +
  divider), regenerated `src/api/schema.d.ts`.
- **Docs**: `backend/AGENTS.md` (OIDC env group), `frontend/AGENTS.md` if the
  login flow description changes.
- No database change. No change when `OIDC_ISSUER` is unset.
