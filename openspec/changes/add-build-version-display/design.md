## Context

The production artifact is one image built by CI on a tag push: the
frontend stage runs `pnpm build`, the backend stage copies `backend/`
(no `.git`) and compiles, and the result is pushed as
`ghcr.io/danielraab/family-finances:<tag>` and `:latest`. The git tag is
known only to the workflow; nothing downstream of `docker build` can
recover it.

## Goals / Non-Goals

- **Goal**: a deployment can state, in the UI, which release it is
  running, with a useful answer for a dev build too.
- **Goal**: exactly one source of that fact, not one per package.
- **Non-goal**: telling the visitor whether a newer release exists.

## Decisions

### The value is a runtime read from the backend, not baked into the bundle

The alternative — a Vite `define` writing `__APP_VERSION__` into the JS
at build time — was rejected for three reasons:

1. The frontend stage of the Docker build has no more access to the git
   tag than the backend stage does, so it would need the same build arg
   plumbed to a second stage: two stamping mechanisms for one fact.
2. Changing a compiled-in constant changes the hashed asset filenames on
   every build, so every commit invalidates the browser cache for the
   whole bundle, whether or not any component changed.
3. The backend is the deployed process. What the visitor wants to know
   is "which build is serving me", and only the server can answer that
   truthfully — a cached `index.html` referencing a stale bundle would
   report the old value.

So: `internal/buildinfo` holds it, `GET /api/version` serves it, the
sidebar fetches it once on mount.

### Stamped via `-ldflags -X`, falling back to `debug.ReadBuildInfo()`

`go build` records VCS metadata automatically, but only when it can see
a `.git` directory — which the Docker backend stage deliberately does
not copy. So the shipped binary needs explicit `-X` values, supplied by
the `publish` job which already holds `github.ref_name` and
`github.sha`.

Locally the opposite is true: nobody types `-ldflags` to run the server,
but `.git` *is* there, so `debug.ReadBuildInfo()` yields
`vcs.revision`. Resolution order per field is therefore: the stamped
value, then the VCS stamp, then empty. A dirty working tree
(`vcs.modified`) appends `-dirty` to the commit, so a locally patched
build never claims to be a clean commit.

Both fields may be empty (`go build` of an exported source tree with no
flags). Empty is a legitimate answer: the endpoint returns empty
strings, and the sidebar renders nothing rather than a placeholder. An
absent line is honest; "unknown" is noise in the one place the app's
chrome should stay quiet.

### Unauthenticated, like the other meta endpoints

`/api/healthz` and `/api/openapi.yaml` are already `security: []`. The
sidebar renders for anonymous visitors too (it is on `/login`), so
gating `/api/version` behind a session would blank the line exactly
where a first-time visitor might want it. The repository is public and
every published image tag is public, so neither the tag nor the commit
is a secret in this deployment.

### Both fields are required, and may be empty

The response is `{"version": string, "commit": string}` with both keys
always present. Optional keys would push a `| undefined` into the
generated client for a distinction — absent versus empty — that carries
no meaning: "we do not know" is one state, and empty string says it.

### Tag wins over commit, commit is shown short

A tagged release is the answer a human wants (`v0.4.2`); the commit is
the fallback for anything built off a tag. Showing both would double the
line's width for no gain, so only one renders, with the full commit in
the `title` attribute for when someone needs to paste it into an issue.
Seven hex characters is the git default abbreviation and fits the
collapsed sidebar's 4rem column.

## Risks / Trade-offs

- **One more request on load.** A tiny unauthenticated GET, fired once
  from a component that mounts once (the sidebar lives in the root
  route). It is not on any critical path: the line simply does not
  render until it resolves.
- **A `-X` flag references a package path as a string.** A rename of
  `internal/buildinfo` silently stops the stamping rather than failing
  the build. Mitigated by a handler test asserting the endpoint's shape
  and by the fallback: an unstamped build still reports a commit in
  every context where `.git` exists (that is, everywhere except the
  image build, which is the case the flag exists for). Accepted — the
  Go toolchain offers no compile-checked alternative.

## Migration Plan

None. No persistence, no configuration, no compatibility surface: a
deployment that has not rebuilt simply keeps not showing a line.
