package httpapi

import (
	"io/fs"
	"net/http"
)

// staticHandler serves the frontend's static export from fsys. Unlike
// http.FileServerFS's default plain-text 404, a request for a path with no
// matching file is answered with fsys's own 404.html (if present) so the
// frontend's not-found page is what visitors see.
func staticHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServerFS(fsys)
	notFoundBody, _ := fs.ReadFile(fsys, "404.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(&notFoundInterceptor{ResponseWriter: w, notFound: notFoundBody}, r)
	})
}

// notFoundInterceptor swaps http.FileServer's plain-text 404 body for the
// embedded 404.html page, when one is present.
type notFoundInterceptor struct {
	http.ResponseWriter
	notFound    []byte
	intercepted bool
}

func (w *notFoundInterceptor) WriteHeader(status int) {
	if status == http.StatusNotFound && w.notFound != nil {
		w.intercepted = true
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *notFoundInterceptor) Write(p []byte) (int, error) {
	if w.intercepted {
		w.intercepted = false // write the substituted body exactly once
		return w.ResponseWriter.Write(w.notFound)
	}
	return w.ResponseWriter.Write(p)
}
