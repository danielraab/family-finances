// Package config loads the backend's runtime configuration from the
// environment. It is the only place in the module that reads environment
// variables — every configurable value is a field on Config, passed down to
// the code that needs it. Documented in backend/.env.example.
package config

import "os"

// Config holds the backend's runtime configuration.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string
}

// Load reads configuration from the environment, applying documented defaults
// for anything unset.
func Load() Config {
	return Config{
		Port: getenv("PORT", "8080"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
