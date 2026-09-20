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
# A caller that doesn't (e.g. Dokploy building this Dockerfile directly off
# a git checkout, with no build-arg wiring of its own) gets them derived
# here instead: --mount=type=bind reads .git straight from the build
# context — the outer `docker build` context, not this stage's copied
# files — without ever copying it into an image layer. VERSION stays empty
# unless HEAD is exactly a tag (an in-progress branch build correctly shows
# no version, only a commit); REVISION is HEAD's full hash. An explicit
# build arg always wins over the derived value.
#
# This does mean the build context must contain .git for this RUN to
# succeed at all — a bind-mount source that doesn't exist fails the build,
# and Dockerfile syntax has no way to make that mount conditional. That's
# true of every normal `docker build` run from a git checkout (Dokploy's
# included), which is the only case this backs; it is not true of a
# context built from a source tarball/export with no .git, which is not a
# supported input to this Dockerfile.
ARG VERSION=""
ARG REVISION=""
RUN --mount=type=bind,source=.git,target=/tmp/.git,ro set -e && \
    V="$VERSION" && R="$REVISION" && \
    if [ -z "$V" ] || [ -z "$R" ]; then \
      apk add --no-cache git >/dev/null && \
      export GIT_DIR=/tmp/.git && \
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
