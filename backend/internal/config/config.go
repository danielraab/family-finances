// Package config loads the backend's runtime configuration from the
// environment. It is the only place in the module that reads environment
// variables — every configurable value is a field on Config, passed down to
// the code that needs it. Documented in backend/.env.example.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds the backend's runtime configuration.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string

	// DatabaseURL is the PostgreSQL connection string (libpq/pgx DSN or URL).
	// It has no default: the backend fails fast at startup when it is unset.
	DatabaseURL string

	// Auth holds authentication and session policy.
	Auth AuthConfig

	// SMTP holds the outbound mail configuration used for magic-link and
	// invite emails.
	SMTP SMTPConfig

	// OIDC holds the single configured OpenID Connect provider.
	OIDC OIDCConfig
}

// AuthConfig is the authentication and session policy, all env-driven.
type AuthConfig struct {
	// BaseURL is the externally reachable origin of this backend, e.g.
	// https://finances.example.com. It builds magic-link URLs and the OIDC
	// redirect_uri. No default.
	BaseURL string

	// SessionTTL is the sliding lifetime of a session, extended on use.
	SessionTTL time.Duration

	// SessionMaxTTL is the absolute cap on a session's age regardless of
	// activity.
	SessionMaxTTL time.Duration

	// CookieSecure sets the Secure attribute on the ff_session cookie. Only
	// disable it for local http development.
	CookieSecure bool

	// SignupEnabled gates account creation. Invite acceptance and the
	// zero-users bootstrap ignore it.
	SignupEnabled bool

	// AllowedEmailDomains, when non-empty, restricts account creation to these
	// domains (case-insensitive). Empty means any domain.
	AllowedEmailDomains []string

	// InviteEnabled governs whether authenticated users can create invites
	// once SignupEnabled is false; while signup is enabled, inviting is always
	// on.
	InviteEnabled bool

	// InviteTTL is how long an invite acceptance link stays valid.
	InviteTTL time.Duration

	// MagicLinkTTL is how long a magic-link token stays valid.
	MagicLinkTTL time.Duration
}

// SMTPTLSMode is one of the accepted SMTP_TLS values.
type SMTPTLSMode string

const (
	SMTPTLSStartTLS SMTPTLSMode = "starttls"
	SMTPTLSImplicit SMTPTLSMode = "implicit"
	SMTPTLSNone     SMTPTLSMode = "none"
)

// SMTPConfig is the outbound mail configuration.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	TLS      SMTPTLSMode
}

// OIDCConfig is the single configured OpenID Connect provider.
type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	Scopes       []string

	// Label is the human-facing text for the OIDC sign-in button on the login
	// page (e.g. "Continue with Google"). Defaults to "Single sign-on".
	Label string
}

// Load reads configuration from the environment, applying documented defaults
// for anything unset. It returns an error only for a value that is set but
// unparseable (a bad duration, a bad bool, an unknown SMTP_TLS mode); values
// without a documented default (e.g. DatabaseURL, AUTH_BASE_URL) are returned
// as-is and the caller is responsible for rejecting an empty value.
func Load() (Config, error) {
	cfg := Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Auth: AuthConfig{
			BaseURL:             os.Getenv("AUTH_BASE_URL"),
			CookieSecure:        true,
			SignupEnabled:       true,
			InviteEnabled:       true,
			AllowedEmailDomains: splitList(os.Getenv("AUTH_ALLOWED_EMAIL_DOMAINS")),
		},
		SMTP: SMTPConfig{
			Host:     os.Getenv("SMTP_HOST"),
			Port:     getenv("SMTP_PORT", "587"),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("SMTP_FROM"),
		},
		OIDC: OIDCConfig{
			Issuer:       os.Getenv("OIDC_ISSUER"),
			ClientID:     os.Getenv("OIDC_CLIENT_ID"),
			ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
			Scopes:       splitListDefault(os.Getenv("OIDC_SCOPES"), []string{"openid", "email", "profile"}),
			Label:        getenv("OIDC_LABEL", "Single sign-on"),
		},
	}

	var err error
	if cfg.Auth.SessionTTL, err = durationEnv("AUTH_SESSION_TTL", 720*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.Auth.SessionMaxTTL, err = durationEnv("AUTH_SESSION_MAX_TTL", 2160*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.Auth.InviteTTL, err = durationEnv("AUTH_INVITE_TTL", 168*time.Hour); err != nil {
		return Config{}, err
	}
	if cfg.Auth.MagicLinkTTL, err = durationEnv("AUTH_MAGIC_LINK_TTL", 15*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.Auth.CookieSecure, err = boolEnv("AUTH_COOKIE_SECURE", true); err != nil {
		return Config{}, err
	}
	if cfg.Auth.SignupEnabled, err = boolEnv("AUTH_SIGNUP_ENABLED", true); err != nil {
		return Config{}, err
	}
	if cfg.Auth.InviteEnabled, err = boolEnv("AUTH_INVITE_ENABLED", true); err != nil {
		return Config{}, err
	}
	if cfg.SMTP.TLS, err = smtpTLSEnv("SMTP_TLS", SMTPTLSStartTLS); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// splitList parses a comma-separated list, trimming whitespace and dropping
// empty entries. An empty or unset value yields a nil slice.
func splitList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func splitListDefault(raw string, fallback []string) []string {
	if list := splitList(raw); list != nil {
		return list
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s: %w", key, err)
	}
	return d, nil
}

func boolEnv(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	switch strings.ToLower(raw) {
	case "1", "t", "true", "yes", "on":
		return true, nil
	case "0", "f", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("config: %s: invalid bool %q", key, raw)
	}
}

func smtpTLSEnv(key string, fallback SMTPTLSMode) (SMTPTLSMode, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	switch SMTPTLSMode(strings.ToLower(raw)) {
	case SMTPTLSStartTLS:
		return SMTPTLSStartTLS, nil
	case SMTPTLSImplicit:
		return SMTPTLSImplicit, nil
	case SMTPTLSNone:
		return SMTPTLSNone, nil
	default:
		return "", fmt.Errorf("config: %s: must be one of starttls|implicit|none, got %q", key, raw)
	}
}
