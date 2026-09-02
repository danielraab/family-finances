package main

import "embed"

// staticFiles holds the frontend's static export, embedded into the binary at
// compile time.
//
// This directive must stay in package main at the module root: //go:embed
// cannot reference a parent directory, and the Docker build copies the real
// frontend/out/ into backend/static/out/ relative to this directory. The
// all: prefix is required — Next writes hashed assets under _next/, and embed
// patterns silently skip names starting with _ or . without it. See
// AGENTS.md §"Serving the frontend". The serving logic lives in
// internal/httpapi/static.go and takes an fs.FS.
//
//go:embed all:static/out
var staticFiles embed.FS
