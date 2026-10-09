# Spec Delta

## Purpose

Defines what the backend writes to its log and how operators control its
volume through environment variables, without changing the line format.

## ADDED Requirements

### Requirement: Request logging is configurable

The backend SHALL log one line per HTTP request (method, path, status,
duration) according to `LOG_REQUESTS` (optional, default `all`): `all` logs
every request, `errors` logs only responses with status 400 or above, and
`off` logs none. Request lines SHALL be logged at `Info`.

#### Scenario: Default logs every request

- **WHEN** `LOG_REQUESTS` is unset and `GET /api/healthz` returns `200`
- **THEN** a request line is logged for it

#### Scenario: Errors only

- **WHEN** `LOG_REQUESTS=errors`
- **THEN** a `200` response logs no request line, while `404`, `429` and
  `500` responses each log one

#### Scenario: Off

- **WHEN** `LOG_REQUESTS=off`
- **THEN** no request line is logged for any response, and log lines written
  by request handlers (such as rate-limit refusals) still appear

### Requirement: Log level is configurable

The backend SHALL write only log output at or above `LOG_LEVEL` (optional,
default `info`; one of `debug`, `info`, `warn`, `error`, case-insensitive),
keeping the existing line format. Because request lines are `Info`, a level
above `info` SHALL also suppress them.

#### Scenario: Warn hides informational lines

- **WHEN** `LOG_LEVEL=warn`
- **THEN** `Info` lines (startup, cleanup passes, request lines) are not
  written, and warnings and errors are

#### Scenario: Debug shows debug lines

- **WHEN** `LOG_LEVEL=debug`
- **THEN** `Debug` lines are written

### Requirement: Invalid logging configuration fails startup

A `LOG_LEVEL` or `LOG_REQUESTS` value outside its accepted set SHALL make
configuration loading fail, so the backend does not start.

#### Scenario: Unknown value

- **WHEN** `LOG_LEVEL=verbose` or `LOG_REQUESTS=some`
- **THEN** configuration loading fails with an error naming the variable
