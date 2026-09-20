package httpapi

import (
	"net/http"

	"at.draab/familyfinances/internal/buildinfo"
)

// BuildInfo is the version and commit the running backend was built from —
// the JSON shape GET /api/version reports (BuildInfo schema in
// openapi/openapi.yaml). Either field may be empty when unknown.
type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// versionHandler serves GET /api/version: the current build's identity, per
// buildinfo.Get() (package-level vars stamped at compile time — see
// internal/buildinfo). It requires no authentication, like /api/healthz and
// /api/openapi.yaml: the sidebar renders it for anonymous visitors too, and
// the value is not a secret.
func versionHandler(w http.ResponseWriter, r *http.Request) {
	version, commit := buildinfo.Get()
	writeJSON(w, r, http.StatusOK, BuildInfo{Version: version, Commit: commit})
}
