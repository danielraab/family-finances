# Proposal

## Why

The backend logs one line per HTTP request, every request, at `Info`,
including the Docker health probe of `/api/healthz`. Nothing about logging is
configurable: operators can neither quiet the access log nor turn down (or up)
the rest of the output.

## What Changes

- `LOG_REQUESTS` (optional, default `all`): `all` logs every request as
  today, `errors` logs only responses with status ≥ 400 (including `429`
  rate-limit refusals), and `off` logs no request lines.
- `LOG_LEVEL` (optional, default `info`): the minimum level of all log output
  — `debug`, `info`, `warn` or `error`. The output format is unchanged.
- An invalid value for either makes the backend refuse to start, like every
  other setting.

## Capabilities

### New Capabilities

- `backend-logging`: what the backend logs and how operators control it.

### Modified Capabilities

_None._

## Impact

- `internal/config` (two fields, two parsers), `main.go` (applies the level
  right after loading config), `internal/httpapi` (the access-log middleware
  takes the mode), `backend/.env.example`, `backend/AGENTS.md`. No API,
  schema or frontend change.
