// Package buildinfo carries the running binary's release identity: the git
// tag and commit it was built from. Both are unknown at compile time — see
// the package vars below — and become known only through one of two paths,
// tried in order:
//
//  1. Stamped in by the container image build via
//     `go build -ldflags "-X .../buildinfo.version=... -X .../buildinfo.commit=..."`
//     (see the root Dockerfile and the CI publish job). This is how a
//     released image knows its own tag: the Docker build stage never copies
//     .git, so nothing else could tell it.
//  2. Falling back to the Go toolchain's own VCS stamp
//     (runtime/debug.ReadBuildInfo), available whenever the build ran
//     inside a .git working copy — true for every local `go build`/`go run`,
//     false for the Docker build. This path only ever yields a commit, never
//     a tag: Go's VCS info has no notion of the nearest tag.
package buildinfo

import "runtime/debug"

// version and commit are the -ldflags -X targets (see the root Dockerfile).
// Renaming this package or either variable silently breaks that flag — see
// design.md's Risks section.
var (
	version string
	commit  string
)

// Get returns the running build's release version and commit. Either may be
// an empty string when it could not be determined; callers never see nil or
// need to distinguish "unknown" from "empty" — empty is the answer.
func Get() (v, c string) {
	if version != "" || commit != "" {
		return version, commit
	}
	return "", vcsCommit()
}

// vcsCommit falls back to the Go toolchain's own VCS build-info, appending
// "-dirty" when it reports uncommitted local changes. Returns "" when no VCS
// info is embedded (e.g. a build outside any .git working copy).
func vcsCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	var revision string
	var dirty bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}
	if revision == "" {
		return ""
	}
	if dirty {
		return revision + "-dirty"
	}
	return revision
}
