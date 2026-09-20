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
# This stage never has a .git directory (only backend/ and openapi/openapi.yaml
# are copied in above), so internal/buildinfo's own VCS fallback comes up
# empty — the release version and commit have to be stamped in explicitly.
# Empty defaults keep a plain `docker build` (no --build-arg) working exactly
# as before, just unstamped.
ARG VERSION=""
ARG REVISION=""
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X at.draab/familyfinances/internal/buildinfo.version=${VERSION} -X at.draab/familyfinances/internal/buildinfo.commit=${REVISION}" -o /out/server .

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
