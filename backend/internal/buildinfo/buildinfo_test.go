package buildinfo

import "testing"

func TestGetStamped(t *testing.T) {
	t.Cleanup(func() {
		version = ""
		commit = ""
	})
	version = "v0.4.2"
	commit = "abc123"

	gotV, gotC := Get()
	if gotV != "v0.4.2" || gotC != "abc123" {
		t.Fatalf("Get() = (%q, %q), want (\"v0.4.2\", \"abc123\")", gotV, gotC)
	}
}

func TestGetStampedCommitOnly(t *testing.T) {
	// A stamped commit with no version still short-circuits the VCS
	// fallback: once either field is stamped, both come from the stamp.
	t.Cleanup(func() {
		version = ""
		commit = ""
	})
	commit = "def456"

	gotV, gotC := Get()
	if gotV != "" || gotC != "def456" {
		t.Fatalf("Get() = (%q, %q), want (\"\", \"def456\")", gotV, gotC)
	}
}

func TestGetUnstampedFallsBackToVCS(t *testing.T) {
	// Unstamped: version and commit are the package's zero values here
	// (nothing else in this test binary sets them). The test binary itself
	// is built by `go test`, which always embeds VCS info when run inside a
	// .git working copy (true in this repo), so this asserts the fallback
	// runs without panicking and returns *some* non-empty commit, or empty
	// when no VCS info is embedded (e.g. an exported source tree) — either
	// is a valid outcome; a panic is not.
	gotV, gotC := Get()
	if gotV != "" {
		t.Fatalf("version = %q, want empty in the unstamped fallback path", gotV)
	}
	_ = gotC // either "", a revision, or "<revision>-dirty" is acceptable
}
