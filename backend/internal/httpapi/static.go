package httpapi

import (
	"io/fs"
	"net/http"
	"path"
)

// staticHandler serves the frontend's Vite bundle from fsys. The bundle is a
// single-page app: a GET/HEAD request for a path with no file extension that
// does not resolve to a bundled file is a client route (e.g. /login,
// /account/234/edit), so it is answered with index.html and 200 and the
// in-browser router renders it. A request that looks like an asset (it has a
// file extension) and misses still gets a 404 — an embedded 404.html body if
// the bundle ships one, otherwise http.FileServer's default.
func staticHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServerFS(fsys)
	indexBody, _ := fs.ReadFile(fsys, "index.html")
	notFoundBody, _ := fs.ReadFile(fsys, "404.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ic := &staticInterceptor{ResponseWriter: w}
		if spaEligible(r) {
			ic.swapOn404 = indexBody
			ic.swapStatus = http.StatusOK
		} else {
			ic.swapOn404 = notFoundBody
			ic.swapStatus = http.StatusNotFound
		}
		fileServer.ServeHTTP(ic, r)
	})
}

// spaEligible reports whether a static miss for this request should fall back
// to the SPA shell rather than a 404: a GET or HEAD navigation whose path has
// no file extension.
func spaEligible(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	return path.Ext(r.URL.Path) == ""
}

// staticInterceptor rewrites http.FileServer's 404: when the server is about to
// write a 404 and swapOn404 is non-nil, it substitutes that body (as
// text/html) with swapStatus instead.
type staticInterceptor struct {
	http.ResponseWriter
	swapOn404  []byte
	swapStatus int
	swapping   bool
}

func (w *staticInterceptor) WriteHeader(status int) {
	if status == http.StatusNotFound && w.swapOn404 != nil {
		w.swapping = true
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Del("Content-Length")
		w.ResponseWriter.WriteHeader(w.swapStatus)
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *staticInterceptor) Write(p []byte) (int, error) {
	if w.swapping {
		w.swapping = false // write the substituted body exactly once
		return w.ResponseWriter.Write(w.swapOn404)
	}
	return w.ResponseWriter.Write(p)
}
