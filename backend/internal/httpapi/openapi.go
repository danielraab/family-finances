package httpapi

import (
	"bytes"
	"net/http"
	"time"
)

// openAPIHandler serves the embedded API contract (openapi/openapi.yaml)
// verbatim at GET /api/openapi.yaml. It requires no authentication — the
// contract is public API surface — and is registered ahead of the "/api/"
// catch-all so it never becomes a JSON 404 or falls through to the static site.
func openAPIHandler(spec []byte) http.HandlerFunc {
	modtime := time.Now()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		http.ServeContent(w, r, "openapi.yaml", modtime, bytes.NewReader(spec))
	}
}
