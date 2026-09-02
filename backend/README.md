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

| Method | Path            | Response                        |
| ------ | --------------- | -------------------------------- |
| GET    | `/api/healthz`  | `ok`                             |
| GET    | `/`, other paths | the embedded frontend static site |

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
