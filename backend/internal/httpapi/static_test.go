package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func testBundle() fstest.MapFS {
	return fstest.MapFS{
		"index.html":              {Data: []byte("<html>shell</html>")},
		"assets/index-abc123.js":  {Data: []byte("console.log('hi')")},
		"assets/index-abc123.css": {Data: []byte("body{}")},
		"favicon.ico":             {Data: []byte("icodata")},
	}
}

func TestStaticHandlerServesBundledFiles(t *testing.T) {
	handler := staticHandler(testBundle())

	cases := []struct{ path, want string }{
		{"/", "<html>shell</html>"},
		{"/assets/index-abc123.js", "console.log('hi')"},
		{"/favicon.ico", "icodata"},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", tc.path, rec.Code)
		}
		assertBody(t, rec, tc.want)
	}
}

func TestStaticHandlerSPAFallback(t *testing.T) {
	handler := staticHandler(testBundle())

	// An extensionless GET that resolves to no file is a client route: serve
	// the shell with 200 so the in-browser router renders it.
	for _, p := range []string{"/login", "/account/234/edit", "/deeply/nested/thing"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))

		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200 (SPA shell)", p, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("%s: Content-Type = %q, want text/html", p, ct)
		}
		assertBody(t, rec, "<html>shell</html>")
	}
}

func TestStaticHandlerMissingAssetStill404(t *testing.T) {
	handler := staticHandler(testBundle())

	// A path that looks like an asset (has an extension) and misses is a real
	// 404 — never the SPA shell.
	for _, p := range []string{"/assets/gone.js", "/nope.css", "/missing.woff2"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))

		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", p, rec.Code)
		}
		body, _ := io.ReadAll(rec.Result().Body)
		if string(body) == "<html>shell</html>" {
			t.Errorf("%s: served the SPA shell for a missing asset", p)
		}
	}
}

func TestStaticHandlerNonGETMissIs404(t *testing.T) {
	handler := staticHandler(testBundle())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/login", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("POST /login: status = %d, want 404", rec.Code)
	}
}

func TestStaticHandlerUsesEmbedded404WhenPresent(t *testing.T) {
	fsys := testBundle()
	fsys["404.html"] = &fstest.MapFile{Data: []byte("<html>not found</html>")}
	handler := staticHandler(fsys)

	// Asset miss with a 404.html in the bundle: serve that page, still 404.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/gone.js", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	assertBody(t, rec, "<html>not found</html>")
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
