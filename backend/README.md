# backend

Go HTTP API for family-finances.

## Commands (run from `backend/`)

```bash
docker compose up -d db   # from the repo root — PostgreSQL on localhost:5432
export DATABASE_URL=postgres://familyfinances:familyfinances@localhost:5432/familyfinances?sslmode=disable
go run .      # start the server (default port 8080, override with PORT)
go test ./... # run tests (Postgres integration tests skip without DATABASE_URL)
go build .    # production build
```

`go run .` / `go build .` binaries fail fast if `DATABASE_URL` is unset or the
database is unreachable; they apply embedded migrations at startup.

## Endpoints

| Method | Path                        | Response                              |
| ------ | --------------------------- | ------------------------------------- |
| GET    | `/api/healthz`              | `ok` (200) / 503 when the DB is down  |
| POST   | `/api/auth/email/start`     | `200` always (no account enumeration) |
| GET    | `/api/auth/email/callback`  | browser: 302 + `ff_session` cookie · API: `{session_token, user}` |
| GET    | `/api/auth/oidc/start`      | 302 to the provider (404 if OIDC unset) |
| GET    | `/api/auth/oidc/callback`   | same sign-in response as the email callback |
| GET    | `/api/auth/me`              | the current user, or `401`            |
| POST   | `/api/auth/logout`          | `204`; revokes the session (auth required) |
| POST   | `/api/auth/invites`         | `201` with the invite (auth required) |
| GET    | `/api/auth/invites/accept`  | sign-in response (creates the account) |
| GET    | `/`, other paths            | the embedded frontend static site     |

## Authentication

Two sign-in methods, one account: **email magic link** over SMTP and one
**OIDC** provider. Sessions are opaque 256-bit tokens (stored only as a SHA-256
hash) — an `HttpOnly` `ff_session` cookie for browsers, `Authorization: Bearer`
for API/mobile clients. See `AGENTS.md` §"Authentication" for the full model.

### Setup

Set at least `AUTH_BASE_URL` (the externally reachable origin) and the SMTP
group, so magic links can be sent:

```bash
export AUTH_BASE_URL=http://localhost:8080
export SMTP_HOST=localhost SMTP_PORT=1025 SMTP_TLS=none SMTP_FROM=dev@localhost
```

`docker compose -f docker-compose.dev.yml up -d` starts a Mailpit catcher on
`:1025` (web UI `:8025`) so no real mail is sent in development.

OIDC is optional — set `OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`
(and optionally `OIDC_SCOPES`, default `openid,email,profile`) to enable the
`/api/auth/oidc/*` routes. Register `AUTH_BASE_URL + /api/auth/oidc/callback`
as the provider's redirect URI. An unset `OIDC_ISSUER` disables those routes.

All auth knobs are documented in `.env.example`.

### First run (bootstrap admin)

While the `users` table is empty the backend is in **bootstrap mode**: signup
is forced open regardless of `AUTH_SIGNUP_ENABLED`, and the **first account to
sign in becomes an admin**. Close signup (`AUTH_SIGNUP_ENABLED=false`) or go
invite-only right after.

### Admin CLI

```bash
./server admin list                 # print every admin's email
./server admin grant  user@example.com
./server admin revoke user@example.com
```

Runs against `DATABASE_URL`; `grant`/`revoke` operate on an existing user and
exit non-zero for an unknown email. `is_admin` gates no behavior yet.

`/` and every other non-`/api/` path are served by the frontend's static
export, embedded into the binary at compile time (see `AGENTS.md` §"Serving
the frontend"). A plain `go build` here embeds an empty placeholder — the
production artifact is built via the root `Dockerfile`, which builds the
frontend first and embeds the real output.

## Docker image

Built from the repo root, not from here:

```bash
docker build -t family-finances -f ../Dockerfile ..   # from backend/
# or, from the repo root:
docker build -t family-finances .
```

The image is a single self-contained binary (no filesystem dependency at
runtime) serving both the frontend and `/api/` routes on `$PORT` (default
`8080`).
