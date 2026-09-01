# Open topic: how the deployed static frontend reaches the backend

**Status:** undecided — deferred to the first real data feature.
**Context:** `frontend/` builds to a static bundle (`pnpm build` → `frontend/out/`),
served by a plain static host (nginx in a container). It ships no backend URL and
currently fetches nothing. When the first endpoint is consumed, we need to pick
how the browser talks to the Go backend.

Originally captured as open question **O1** in
`openspec/changes/archive/2026-09-02-frontend-static-shell/design.md`.

## Options

```
① nginx proxies /api                ② Go serves out/                 ③ browser → backend direct
   Browser ─▶ nginx ─┬─ /   out/       Browser ─▶ Go ─┬─ /   out/       Browser ─▶ CDN/nginx  out/
                     └─ /api ▶ backend               └─ /api handlers   Browser ─▶ backend (public)
```

| Option | CORS needed | Backend public | Env var | Keeps packages separate | Notes |
| --- | --- | --- | --- | --- | --- |
| **① nginx `location /api` → `proxy_pass` backend**; client fetches relative `/api/...` | no | no | no | yes | +~5 lines of nginx config; nginx already planned for serving `out/`. Browser only ever sees the nginx origin. |
| **② Go binary serves `out/` (embed) and `/api` handlers** | no | no | no | **no** — fuses the frontend build into the Go build/deploy | One origin, one deployable. Violates the root `AGENTS.md` rule "no shared build, no shared code; independent deployables". |
| **③ Browser calls the backend directly** | **yes** | **yes** | `NEXT_PUBLIC_API_URL` baked in at build time | yes | Backend becomes internet-facing (needs an auth story). Reintroduces the env var that `frontend-static-shell` deleted, now client-side. |

## Current lean

**Option ①.** It matches the "nginx serves static files in a Docker container"
deployment plan (just add an `/api` location), needs no CORS, keeps the backend
private, adds no environment variable, and preserves the existing invariant that
the browser never holds a backend URL. The `BACKEND_URL` / "server-side only"
idea from the old docs effectively moves from Next's server to nginx's
`proxy_pass` target.

## When this is decided

- Update `frontend/AGENTS.md`, `frontend/README.md`, root `AGENTS.md`, root
  `README.md`, and `openspec/config.yaml` with the concrete mechanism (keep the
  "no `BACKEND_URL`" wording; add the chosen path).
- If ①: add the nginx config and the Docker/compose wiring (frontend container
  serving `out/` + proxying `/api` to the backend container over a private
  network).
- Do it through an OpenSpec change (deployment / first data feature).
