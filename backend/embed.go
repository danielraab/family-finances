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
