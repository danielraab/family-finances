package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", handleHealthz)

	static, err := fs.Sub(embeddedStatic, "static/out")
	if err != nil {
		log.Fatalf("static sub fs: %v", err)
	}
	mux.Handle("/", staticHandler(static))

	addr := ":" + port()
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := w.Write([]byte("ok")); err != nil {
		log.Printf("write response: %v", err)
	}
}

func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}
