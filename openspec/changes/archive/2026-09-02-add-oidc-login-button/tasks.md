## 1. Backend: label config and service accessor

- [x] 1.1 `internal/config`: add `Label string` to `OIDCConfig`, populated from
  `OIDC_LABEL` with default `Single sign-on`
- [x] 1.2 `backend/.env.example`: document `OIDC_LABEL` in the `OIDC_*` block
- [x] 1.3 `internal/auth`: add `OIDCLabel string` to `Params`; wire it through
  `main.buildAuth`
- [x] 1.4 `internal/auth`: add `func (s *Service) OIDCLogin() (label string, ok bool)`
  — `ok` is `s.oidc != nil`, `label` is `s.p.OIDCLabel`
- [x] 1.5 `main.buildAuth`: build the OIDC client only when `OIDC_ISSUER` **and**
  `OIDC_CLIENT_ID` are both set (today it checks issuer only)
- [x] 1.6 Unit tests: `OIDCLogin()` returns `("<label>", true)` with a stub OIDC
  client and `("", false)` without one; config default test for `OIDC_LABEL`

## 2. Backend: GET /api/auth/config endpoint

- [x] 2.1 `internal/auth/handler.go`: register `GET /api/auth/config`; handler
  calls `svc.OIDCLogin()` and writes
  `{ "oidc": { "label": ..., "start_path": "/api/auth/oidc/start" } }` or
  `{ "oidc": null }`
- [x] 2.2 Ensure the route is reachable without auth (it is a sign-in route, not
  behind `RequireAuth`)
- [x] 2.3 Handler tests: `oidc` object when configured, `oidc: null` when not,
  `200` with no session, no issuer/client-id/secret in the body

## 3. Spec: document the endpoint

- [x] 3.1 Add `GET /api/auth/config` to `openapi/openapi.yaml`: `200` response,
  `AuthConfig` schema with nullable `oidc` object (`label`, `start_path`)
- [x] 3.2 Sync `backend/openapi.yaml`; run the backend response-validation tests
- [x] 3.3 Regenerate and commit `frontend/src/api/schema.d.ts`
- [x] 3.4 `spectral lint openapi/openapi.yaml` clean

## 4. Frontend: login page

- [x] 4.1 `src/routes/login.tsx`: on mount, `GET /api/auth/config` via the typed
  client into local state (`oidc: {label, start_path} | null`); default null,
  failure → null
- [x] 4.2 When `oidc` is present, render an `<a href={oidc.start_path}>` styled
  as a primary button with text `oidc.label`, **above** the email field
- [x] 4.3 Render an "or" divider between the OIDC button and the email form;
  render neither button nor divider when `oidc` is null
- [x] 4.4 Keep the email form usable while the config request is in flight (no
  layout dependency on the response)
- [x] 4.5 `pnpm lint`, `pnpm exec tsc`, `pnpm build` clean

## 5. Verify end to end

- [x] 5.1 Run the backend with no `OIDC_*` set → `/login` shows only the email
  form; `GET /api/auth/config` returns `{ "oidc": null }`
- [x] 5.2 Run with a test OIDC provider + `OIDC_LABEL` set → button appears above
  the form with the label, clicking it navigates to `/api/auth/oidc/start`
- [x] 5.3 `cd backend && gofmt -l . && go vet ./... && go test ./...` clean

## 6. Documentation

- [x] 6.1 `backend/AGENTS.md`: add `OIDC_LABEL` to the env group list; note the
  `GET /api/auth/config` route and the issuer+client-id gate change
- [x] 6.2 `frontend/AGENTS.md`: note the login page's OIDC affordance and the
  `GET /api/auth/config` call if the auth section describes the login flow
