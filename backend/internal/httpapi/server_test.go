package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func testDeps() Deps {
	return Deps{Static: fstest.MapFS{
		"index.html": {Data: []byte("<html>home</html>")},
		"404.html":   {Data: []byte("<html>not found</html>")},
	}}
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
