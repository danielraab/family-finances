package main

import (
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}

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

// healthcheck probes the running server's health endpoint and returns a
// process exit code. It backs the Docker HEALTHCHECK: the distroless runtime
// image has no shell or curl, so the server binary itself is the probe
// (`/app/server healthcheck`).
func healthcheck() int {
	return healthcheckURL("http://127.0.0.1:" + port() + "/api/healthz")
}

func healthcheckURL(url string) int {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("healthcheck: %v", err)
		return 1
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "ok" {
		log.Printf("healthcheck: status=%d body=%q", resp.StatusCode, body)
		return 1
	}
	return 0
}

func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}
