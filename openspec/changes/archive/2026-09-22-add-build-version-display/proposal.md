## Why

A running deployment cannot say what it is. The image is published to
GHCR tagged with the git tag that built it (`v0.4.2`) plus `latest`, and
`latest` is what a home server actually pulls — so after a `docker
compose pull && up -d` there is no way, from the app itself, to tell
whether the tab you are looking at is the release you just published or
the one from three weeks ago. The only answers today are on the host
(`docker image inspect`), not in the product.

Nothing in the binary carries that identity either: no `-ldflags` are
passed anywhere in the build, the Docker backend stage copies `backend/`
without `.git`, so even Go's own `debug.ReadBuildInfo()` VCS stamping
comes up empty in the shipped image. The value has to be put in at build
time, and there is exactly one place that knows it — the `publish` job,
which already holds the tag as `github.ref_name`.

## What Changes

- A new `internal/buildinfo` package holds the running build's identity:
  a release version and a commit hash, stamped in with
  `go build -ldflags -X`. When nothing is stamped (a plain `go run`
  during development) it falls back to `debug.ReadBuildInfo()`'s VCS
  stamp, so a dev build still reports its commit.
- `GET /api/version` returns that identity as JSON
  (`{"version": "v0.4.2", "commit": "8065011…"}`), unauthenticated,
  alongside the other two meta endpoints (`/api/healthz`,
  `/api/openapi.yaml`).
- The root `Dockerfile` takes `VERSION` and `REVISION` build args and
  passes them through as `-ldflags`; CI's `publish` job supplies
  `github.ref_name` and `github.sha`. When either arg is left empty, the
  backend stage falls back to deriving it from the build context's own
  `.git`, if the context has one — a `docker build` run from an ordinary
  git checkout then reports correctly with zero platform-specific
  configuration. Daniel's Dokploy stage environment builds this
  `Dockerfile` directly with no build-arg wiring of its own, which is
  what this fallback was added for — but Dokploy's build context turns
  out not to carry `.git` at all (it builds from an export), so on
  Dokploy specifically this still reports nothing, same as before the
  fallback existed; it isn't a broken build, just not a solved problem
  for that one platform.
- The Settings page's Profile tab renders the result as one small,
  muted, read-only line below the existing fields: the release tag when
  there is one, otherwise the short commit hash, and nothing at all when
  the backend reports neither. (First shipped in the sidebar footer;
  Daniel asked for it moved to Settings instead — see design.md.)

## Non-goals

- **No update check.** The app does not learn what the newest release
  is, does not call GitHub, and shows no "update available" badge. It
  reports what it is, nothing more.
- **No build timestamp, Go version, or dependency list.** Two fields,
  both about identity.
- **No new page or dialog.** There is no About screen; the Profile tab's
  version line is the whole surface.
- **No version in the frontend bundle.** See design.md — the value stays
  a runtime read, so the hashed assets do not churn per commit.

## Capabilities

### New Capabilities

- `build-version`: the running binary knows and reports the release tag
  and commit it was built from, stamped at build time and served at
  `GET /api/version`.

### Modified Capabilities

- `web-client-settings`: the Profile tab shows the running build's
  version (or short commit) as a small, read-only line below its
  existing fields.
- `release-pipeline`: the published image is stamped with the git tag
  and commit that produced it.

## Impact

- `backend/internal/buildinfo/buildinfo.go` — new.
- `backend/internal/httpapi/{server.go,version.go}` — the
  `GET /api/version` route.
- `openapi/openapi.yaml` (+ the two committed generated artifacts) — the
  new operation and its `BuildInfo` schema.
- `Dockerfile` — `VERSION`/`REVISION` args, `-ldflags` on `go build`,
  and a `.git` bind-mount fallback for when they're unset.
- `.github/workflows/ci.yml` — `build-args` on the publish step.
- `frontend/src/routes/settings.index.tsx` — the Profile tab's version
  line (moved here from an earlier `Sidebar`/`SidebarVersion.tsx` cut,
  removed).
- `frontend/src/i18n/locales/{en,de}.json` — its `settings.profile.version`
  label.
