# AGENTS.md — backend

Go HTTP API for family-finances. Module `at.draab/familyfinances`.

## Stack

- Go 1.26, **standard library only** (`net/http`, `log`, `os`). No web framework,
  no ORM, no third-party dependencies.
- Do not add a dependency without an OpenSpec proposal.

## Conventions

- Routing uses Go 1.22+ pattern syntax: `mux.HandleFunc("GET /{$}", handler)`.
- Configuration comes from environment variables. Current vars are documented in
  `.env.example` (`PORT`, default `8080`). Go does not auto-load `.env` — export
  the vars or use direnv.
- No persistence layer yet. When one is added it lives here — the frontend never
  talks to a database.

## Before you're done

```bash
gofmt -l .        # must print nothing
go vet ./...
go test ./...
```

## Commands

```bash
go run .          # start on :8080 (or $PORT)
go build .        # production binary
```
