## MODIFIED Requirements

### Requirement: The go:embed directive is pinned to package main at the module root

The `//go:embed` directive for the frontend bundle SHALL remain in a
`package main` file at the `backend/` module root, embedding `static/out` (the
directory the Docker build populates from `frontend/out/`). The serving logic
that consumes it (`staticHandler` and its SPA-fallback behaviour) SHALL live in
`internal/httpapi/static.go` and operate on an `fs.FS` value passed in by
`main.go` via `fs.Sub`.

#### Scenario: Embed directive location

- **WHEN** `backend/embed.go` is read
- **THEN** it is in `package main`, contains a `//go:embed` directive for
  `static/out`, and exports the resulting `embed.FS`

#### Scenario: Static handler takes an fs.FS

- **WHEN** `internal/httpapi` serves static files
- **THEN** its `staticHandler` accepts an `fs.FS` parameter and does not
  itself reference `embed` or `static/out`
