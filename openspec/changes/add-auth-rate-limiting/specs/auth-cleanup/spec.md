# Spec Delta

## Purpose

Keeps short-lived authentication records from accumulating by periodically
deleting rows that can never be used again, and bounds the memory held by the
in-process rate limiters.

## ADDED Requirements

### Requirement: Periodic cleanup of expired authentication records

While the server runs, the backend SHALL run a cleanup pass once at startup and
then every `AUTH_CLEANUP_INTERVAL` (optional, default `15m`). Each pass SHALL
delete sessions past `expires_at` or older than `AUTH_SESSION_MAX_TTL`,
magic-link tokens that are expired or consumed, and OIDC login state past
`expires_at`. It SHALL NOT delete invites, users, identities or still-valid
sessions and tokens.

#### Scenario: Expired session is removed

- **WHEN** a cleanup pass runs and a session's `expires_at` is in the past
- **THEN** that session row is deleted

#### Scenario: Over-age session is removed

- **WHEN** a cleanup pass runs and a session was created longer ago than
  `AUTH_SESSION_MAX_TTL`, even though its sliding `expires_at` is in the future
- **THEN** that session row is deleted

#### Scenario: Active session is kept

- **WHEN** a cleanup pass runs and a session is unexpired and within
  `AUTH_SESSION_MAX_TTL`
- **THEN** that session row is kept and the session still authenticates

#### Scenario: Used and stale magic-link tokens are removed

- **WHEN** a cleanup pass runs
- **THEN** every consumed magic-link token and every token past its
  `expires_at` is deleted, and unconsumed unexpired tokens are kept

#### Scenario: Abandoned OIDC state is removed

- **WHEN** a cleanup pass runs and an OIDC login state row is past its
  `expires_at`
- **THEN** that row is deleted

#### Scenario: Invites are untouched

- **WHEN** a cleanup pass runs and an invite is expired, accepted or revoked
- **THEN** the invite row remains and still appears in invite listings

### Requirement: Cleanup is safe to run concurrently and survives failures

A cleanup pass SHALL be idempotent, so several replicas running it at once
produce the same result as one. A failed pass SHALL be logged and SHALL NOT
stop the server or later passes. The job SHALL stop when the server shuts down.

#### Scenario: Database error during a pass

- **WHEN** a cleanup pass fails because the database is unreachable
- **THEN** the error is logged, the server keeps serving, and the next pass
  runs on schedule

#### Scenario: Graceful shutdown

- **WHEN** the server receives SIGTERM
- **THEN** the cleanup job stops without starting a new pass

### Requirement: Idle rate-limit buckets are evicted

Each cleanup pass SHALL evict in-memory rate-limit buckets whose window has
fully elapsed, so limiter memory stays proportional to recently active clients
and recipients. Evicting a bucket SHALL never let a client exceed its limit.

#### Scenario: Idle bucket is evicted

- **WHEN** a cleanup pass runs and a client IP has made no throttled request
  for longer than `RATE_LIMIT_IP_WINDOW`
- **THEN** its bucket is removed from memory

#### Scenario: Active bucket is kept

- **WHEN** a cleanup pass runs and a client IP is still inside its window
- **THEN** its bucket and its count are kept

### Requirement: Cleanup interval is configurable

`AUTH_CLEANUP_INTERVAL` SHALL be optional with a default of `15m`. A value that
is not a positive Go duration SHALL make the backend refuse to start with a
clear error.

#### Scenario: Default interval

- **WHEN** `AUTH_CLEANUP_INTERVAL` is unset
- **THEN** cleanup passes run every 15 minutes

#### Scenario: Invalid interval

- **WHEN** `AUTH_CLEANUP_INTERVAL=0s`
- **THEN** configuration loading fails and the backend does not start
