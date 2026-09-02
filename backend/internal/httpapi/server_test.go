package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// stubPinger is a DBPinger whose Ping returns err.
type stubPinger struct{ err error }

func (s stubPinger) Ping(context.Context) error { return s.err }

func testDeps() Deps {
	return Deps{
		Static: fstest.MapFS{
			"index.html": {Data: []byte("<html>home</html>")},
			"404.html":   {Data: []byte("<html>not found</html>")},
		},
		DB: stubPinger{},
	}
}

func TestRoutesHealthz(t *testing.T) {
	srv := httptest.NewServer(Routes(testDeps()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/healthz")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
}

func TestRoutesNonAPIServesStatic(t *testing.T) {
	rec := httptest.NewRecorder()
	Routes(testDeps()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body, _ := io.ReadAll(rec.Result().Body)
	if string(body) != "<html>home</html>" {
		t.Fatalf("body = %q, want index.html", body)
	}
}

func TestRoutesOpenAPIDocument(t *testing.T) {
	deps := testDeps()
	deps.OpenAPISpec = []byte("openapi: 3.0.3\ninfo:\n  title: t\n  version: 0\n")

	srv := httptest.NewServer(Routes(deps))
	defer srv.Close()

	// No credentials of any kind.
	resp, err := http.Get(srv.URL + "/api/openapi.yaml")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/yaml") {
		t.Fatalf("Content-Type = %q, want application/yaml", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 || !strings.Contains(string(body), "openapi:") {
		t.Fatalf("body = %q, want the OpenAPI document", body)
	}
}

func TestRoutesOpenAPIDocumentUnsetIs404(t *testing.T) {
	rec := httptest.NewRecorder()
	Routes(testDeps()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d when OpenAPISpec is nil", rec.Code, http.StatusNotFound)
	}
}

func TestRoutesUnknownAPIPathIsNotStatic(t *testing.T) {
	rec := httptest.NewRecorder()
	Routes(testDeps()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	body, _ := io.ReadAll(rec.Result().Body)
	if string(body) == "<html>not found</html>" || string(body) == "<html>home</html>" {
		t.Fatalf("unknown /api/ path served frontend content: %q", body)
	}
}
