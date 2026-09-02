# syntax=docker/dockerfile:1

# ---- frontend: build the static export ----
FROM node:22-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN corepack enable && corepack prepare pnpm@11.5.3 --activate \
    && pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

# ---- backend: embed the static export and compile ----
FROM golang:1.26-alpine AS backend
WORKDIR /src/backend
COPY backend/ ./
RUN rm -rf static/out && mkdir -p static/out
COPY --from=frontend /src/frontend/out/. static/out/
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server .

# ---- final: minimal non-root runtime ----
FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=backend /out/server /app/server
EXPOSE 8080
# distroless has no shell/curl, so the server binary probes its own
# /api/healthz endpoint via `server healthcheck`.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/app/server", "healthcheck"]
ENTRYPOINT ["/app/server"]
