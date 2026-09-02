# backend

Go HTTP API for family-finances.

## Commands (run from `backend/`)

```bash
go run .      # start the server (default port 8080, override with PORT)
go test ./... # run tests
go build .    # production build
```

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
