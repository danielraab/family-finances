package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestServerBinaryExcludesOpenAPIValidator enforces the api-contract rule that
// the OpenAPI response validator is test-support only: the compiled server's
// transitive dependency graph must contain neither kin-openapi nor the
// internal/openapicheck helper that wraps it. It shells out to `go list` for
// the non-test import graph of package main.
func TestServerBinaryExcludesOpenAPIValidator(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go tool not on PATH")
	}
	out, err := exec.Command(goBin, "list", "-deps", "-f", "{{.ImportPath}}", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.Contains(line, "github.com/getkin/kin-openapi") {
			t.Errorf("server binary transitively imports the OpenAPI validator: %s", line)
		}
		if strings.Contains(line, "at.draab/familyfinances/internal/openapicheck") {
			t.Errorf("server binary transitively imports internal/openapicheck: %s", line)
		}
	}
}
