package main

import "embed"

// staticFiles holds the frontend's built bundle (Vite → frontend/out/),
// embedded into the binary at compile time.
//
// This directive must stay in package main at the module root: //go:embed
// cannot reference a parent directory, and the Docker build copies the real
// frontend/out/ into backend/static/out/ relative to this directory. The
// all: prefix is retained but no longer load-bearing — Vite writes hashed
// assets under assets/ (no leading _ or .), so plain //go:embed would suffice;
// keeping all: is harmless and guards against a future asset dir that does
// start with _. See AGENTS.md §"Serving the frontend". The serving logic
// (including the SPA fallback to index.html) lives in
// internal/httpapi/static.go and takes an fs.FS.
//
//go:embed all:static/out
var staticFiles embed.FS

// openAPISpec is the hand-written API contract (openapi/openapi.yaml), served
// verbatim at GET /api/openapi.yaml.
//
// Same parent-directory constraint as staticFiles: //go:embed cannot reach
// ../openapi/, so a synced copy lives at backend/openapi.yaml (kept identical
// by `go generate ./...` and a CI check; the Docker build overwrites it from
// the real openapi/openapi.yaml before compiling). The serving logic lives in
// internal/httpapi and takes the bytes as a value.
//
//go:generate cp ../openapi/openapi.yaml ./openapi.yaml
//go:embed openapi.yaml
var openAPISpec []byte
