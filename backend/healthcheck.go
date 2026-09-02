package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"at.draab/familyfinances/internal/config"
)

// healthcheck probes the running server's health endpoint and returns a
// process exit code. It backs the Docker HEALTHCHECK: the distroless runtime
// image has no shell or curl, so the server binary itself is the probe
// (`/app/server healthcheck`).
func healthcheck() int {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("healthcheck: load config: %v", err)
		return 1
	}
	return healthcheckURL("http://127.0.0.1:" + cfg.Port + "/api/healthz")
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
