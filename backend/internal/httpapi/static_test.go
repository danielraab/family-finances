package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestStaticHandler(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":   {Data: []byte("<html>home</html>")},
		"_next/app.js": {Data: []byte("console.log('hi')")},
		"404.html":     {Data: []byte("<html>not found</html>")},
	}
	handler := staticHandler(fsys)

	t.Run("root serves index.html", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		assertBody(t, rec, "<html>home</html>")
	})

	t.Run("hashed asset serves as-is", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_next/app.js", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		assertBody(t, rec, "console.log('hi')")
	})

	t.Run("unknown path returns embedded 404 page", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/missing", nil))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want %q", ct, "text/html; charset=utf-8")
		}
		assertBody(t, rec, "<html>not found</html>")
	})
}

func TestStaticHandlerWithoutNotFoundPage(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": {Data: []byte("<html>home</html>")},
	}
	handler := staticHandler(fsys)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/missing", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func assertBody(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	body, err := io.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}
