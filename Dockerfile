# syntax=docker/dockerfile:1

# ---- frontend: build the static SPA bundle (Vite → frontend/out/) ----
FROM node:26-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
# Node 26 no longer bundles Corepack (nodejs/node#57617); install it from npm
# before enabling it, rather than relying on `corepack` being preinstalled.
RUN npm install -g corepack && corepack enable && pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

# ---- backend: embed the static export + API contract and compile ----
FROM golang:1.27-alpine AS backend
WORKDIR /src/backend
COPY backend/ ./
# The API contract lives at repo-root openapi/; //go:embed cannot reach a
# parent dir, so overwrite the committed copy with the source of truth before
# compiling (the embedded doc is then always current, regardless of drift).
COPY openapi/openapi.yaml ./openapi.yaml
RUN rm -rf static/out && mkdir -p static/out
COPY --from=frontend /src/frontend/out/. static/out/
# internal/buildinfo's own VCS fallback needs a .git directory, and this
# stage never has one (only backend/ and openapi/openapi.yaml are copied in
# above) — so a caller that already knows its version/commit (CI's publish
# job, which has github.ref_name/github.sha) passes them as build args.
# A caller that doesn't gets them derived here instead, from .git, if the
# build context happens to have one: --mount=type=bind exposes the outer
# `docker build` context read-only, without ever copying it into an image
# layer. Mounted at the context *root* ("."), not straight at ".git":
# a bind-mount source that doesn't exist fails the build outright with no
# way to make the mount itself conditional, whereas "." always exists (it
# is the context). ".git" itself is checked for with a plain [ -d ] before
# it's ever used, so a context that doesn't have one (confirmed true of
# Dokploy, which builds from an export, not a working copy) just skips
# this and builds unstamped, same as before this existed — never a broken
# build. VERSION stays empty unless HEAD is exactly a tag (an in-progress
# branch build correctly shows no version, only a commit); REVISION is
# HEAD's full hash. An explicit build arg always wins over the derived
# value.
ARG VERSION=""
ARG REVISION=""
RUN --mount=type=bind,source=.,target=/tmp/ctx,ro set -e && \
    V="$VERSION" && R="$REVISION" && \
    if { [ -z "$V" ] || [ -z "$R" ]; } && [ -d /tmp/ctx/.git ]; then \
      apk add --no-cache git >/dev/null && \
      export GIT_DIR=/tmp/ctx/.git && \
      { [ -n "$V" ] || V=$(git describe --tags --exact-match 2>/dev/null || echo ""); } && \
      { [ -n "$R" ] || R=$(git rev-parse HEAD 2>/dev/null || echo ""); }; \
    fi && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags "-X at.draab/familyfinances/internal/buildinfo.version=${V} -X at.draab/familyfinances/internal/buildinfo.commit=${R}" -o /out/server .

# ---- final: minimal non-root runtime ----
FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=backend /out/server /app/server
# distroless ships no shell; borrow the static busybox binary as /bin/sh so
# tools that shell out (e.g. `docker exec ... sh -c`) still work. The server
# itself never needs it: healthchecks call the binary directly, below.
COPY --from=busybox:1.36-musl /bin/busybox /bin/sh
EXPOSE 8080
# distroless still has no curl, so the server binary probes its own
# /api/healthz endpoint via `server healthcheck`.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/app/server", "healthcheck"]
ENTRYPOINT ["/app/server"]
