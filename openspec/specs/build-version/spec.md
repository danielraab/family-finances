# build-version Specification

## Purpose

Lets a running backend report its own release identity — the git tag (if
any) and commit it was built from — so the frontend can surface it (see
`web-client-settings`'s Profile tab) and so the release pipeline's
published image is self-describing.

## Requirements

### Requirement: The running build reports its release version and commit

The backend SHALL know, at runtime, the git tag (if any) and commit hash
it was built from, and SHALL serve them unauthenticated at
`GET /api/version` as `{"version": string, "commit": string}`.

`version` SHALL be the git tag the release image was built from (e.g.
`v0.4.2`), or an empty string when the build was not made from a tag.
`commit` SHALL be the full git commit hash the build was made from, or
an empty string when it cannot be determined. Both fields are always
present; an unknown value is an empty string, never `null` or an
omitted key.

The values SHALL be fixed at build time via linker flags
(`go build -ldflags`), supplied by the container image build. When no
such flags are supplied (a plain `go build`/`go run` in a working copy
with `.git` present), the backend SHALL fall back to the Go toolchain's
own VCS build-info (`debug.ReadBuildInfo()`) for the commit, appending
`-dirty` to it when the toolchain reports uncommitted local changes;
`version` has no such fallback and stays empty outside a stamped build.

#### Scenario: A release image reports its tag and commit

- **WHEN** the backend was built with `-ldflags` stamping version
  `v0.4.2` and a commit hash, and a client calls `GET /api/version`
- **THEN** the response is `200` with `version: "v0.4.2"` and `commit`
  equal to the stamped hash

#### Scenario: An unstamped local build falls back to the VCS commit

- **WHEN** the backend was built with no `-ldflags` inside a working
  copy that has a `.git` directory, and a client calls
  `GET /api/version`
- **THEN** the response is `200` with `version: ""` and `commit` equal
  to the current commit hash as reported by `debug.ReadBuildInfo()`

#### Scenario: A dirty local build is marked

- **WHEN** the backend was built with no `-ldflags` inside a working
  copy that has uncommitted changes
- **THEN** `commit` in the `GET /api/version` response ends with
  `-dirty`

#### Scenario: Neither is determinable

- **WHEN** the backend was built with no `-ldflags` and no `.git`
  directory or VCS build info is available
- **THEN** `GET /api/version` still responds `200`, with `version: ""`
  and `commit: ""`

#### Scenario: The endpoint requires no authentication

- **WHEN** an anonymous visitor (no session cookie) calls
  `GET /api/version`
- **THEN** the response is `200`, identical to an authenticated caller's
