# syntax=docker/dockerfile:1

# ---- frontend: build the static SPA bundle (Vite → frontend/out/) ----
FROM node:26-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
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
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server .

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
