package config_test

import (
	"testing"
	"time"

	"at.draab/familyfinances/internal/config"
)

func load(t *testing.T) config.Config {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")

	cfg := load(t)

	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "8080")
	}
}

func TestLoadPortOverride(t *testing.T) {
	t.Setenv("PORT", "9999")

	cfg := load(t)

	if cfg.Port != "9999" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "9999")
	}
}

func TestLoadDatabaseURLEmptyWhenUnset(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	cfg := load(t)

	if cfg.DatabaseURL != "" {
		t.Fatalf("DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}
}

func TestLoadDatabaseURLFromEnv(t *testing.T) {
	const dsn = "postgres://u:p@localhost:5432/family_finances?sslmode=disable"
	t.Setenv("DATABASE_URL", dsn)

	cfg := load(t)

	if cfg.DatabaseURL != dsn {
		t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, dsn)
	}
}

func TestLoadAuthDefaults(t *testing.T) {
	for _, k := range []string{
		"AUTH_SESSION_TTL", "AUTH_SESSION_MAX_TTL", "AUTH_INVITE_TTL",
		"AUTH_MAGIC_LINK_TTL", "AUTH_COOKIE_SECURE", "AUTH_SIGNUP_ENABLED",
		"AUTH_INVITE_ENABLED", "AUTH_ALLOWED_EMAIL_DOMAINS", "SMTP_TLS", "OIDC_SCOPES",
		"OIDC_LABEL",
	} {
		t.Setenv(k, "")
	}

	cfg := load(t)

	if cfg.Auth.SessionTTL != 720*time.Hour {
		t.Errorf("SessionTTL = %s, want 720h", cfg.Auth.SessionTTL)
	}
	if cfg.Auth.SessionMaxTTL != 2160*time.Hour {
		t.Errorf("SessionMaxTTL = %s, want 2160h", cfg.Auth.SessionMaxTTL)
	}
	if cfg.Auth.InviteTTL != 168*time.Hour {
		t.Errorf("InviteTTL = %s, want 168h", cfg.Auth.InviteTTL)
	}
	if cfg.Auth.MagicLinkTTL != 15*time.Minute {
		t.Errorf("MagicLinkTTL = %s, want 15m", cfg.Auth.MagicLinkTTL)
	}
	if !cfg.Auth.CookieSecure {
		t.Error("CookieSecure = false, want true")
	}
	if !cfg.Auth.SignupEnabled {
		t.Error("SignupEnabled = false, want true")
	}
	if !cfg.Auth.InviteEnabled {
		t.Error("InviteEnabled = false, want true")
	}
	if cfg.Auth.AllowedEmailDomains != nil {
		t.Errorf("AllowedEmailDomains = %v, want nil", cfg.Auth.AllowedEmailDomains)
	}
	if cfg.SMTP.Port != "587" {
		t.Errorf("SMTP.Port = %q, want 587", cfg.SMTP.Port)
	}
	if cfg.SMTP.TLS != config.SMTPTLSStartTLS {
		t.Errorf("SMTP.TLS = %q, want starttls", cfg.SMTP.TLS)
	}
	if want := []string{"openid", "email", "profile"}; !equal(cfg.OIDC.Scopes, want) {
		t.Errorf("OIDC.Scopes = %v, want %v", cfg.OIDC.Scopes, want)
	}
	if cfg.OIDC.Label != "Single sign-on" {
		t.Errorf("OIDC.Label = %q, want %q", cfg.OIDC.Label, "Single sign-on")
	}
}

func TestLoadOIDCLabelOverride(t *testing.T) {
	t.Setenv("OIDC_LABEL", "Continue with Google")
	if cfg := load(t); cfg.OIDC.Label != "Continue with Google" {
		t.Errorf("OIDC.Label = %q, want %q", cfg.OIDC.Label, "Continue with Google")
	}
}

func TestLoadAuthParsing(t *testing.T) {
	t.Setenv("AUTH_SESSION_TTL", "48h")
	t.Setenv("AUTH_MAGIC_LINK_TTL", "5m")
	t.Setenv("AUTH_COOKIE_SECURE", "false")
	t.Setenv("AUTH_SIGNUP_ENABLED", "0")
	t.Setenv("AUTH_INVITE_ENABLED", "no")
	t.Setenv("OIDC_SCOPES", "openid, email")

	cfg := load(t)

	if cfg.Auth.SessionTTL != 48*time.Hour {
		t.Errorf("SessionTTL = %s, want 48h", cfg.Auth.SessionTTL)
	}
	if cfg.Auth.MagicLinkTTL != 5*time.Minute {
		t.Errorf("MagicLinkTTL = %s, want 5m", cfg.Auth.MagicLinkTTL)
	}
	if cfg.Auth.CookieSecure {
		t.Error("CookieSecure = true, want false")
	}
	if cfg.Auth.SignupEnabled {
		t.Error("SignupEnabled = true, want false")
	}
	if cfg.Auth.InviteEnabled {
		t.Error("InviteEnabled = true, want false")
	}
	if want := []string{"openid", "email"}; !equal(cfg.OIDC.Scopes, want) {
		t.Errorf("OIDC.Scopes = %v, want %v", cfg.OIDC.Scopes, want)
	}
}

func TestLoadDomainListSplitting(t *testing.T) {
	t.Setenv("AUTH_ALLOWED_EMAIL_DOMAINS", " example.com ,, foo.org ,")

	cfg := load(t)

	want := []string{"example.com", "foo.org"}
	if !equal(cfg.Auth.AllowedEmailDomains, want) {
		t.Fatalf("AllowedEmailDomains = %v, want %v", cfg.Auth.AllowedEmailDomains, want)
	}
}

func TestLoadInvalidSMTPTLSRejected(t *testing.T) {
	t.Setenv("SMTP_TLS", "bogus")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() succeeded with SMTP_TLS=bogus, want error")
	}
}

func TestLoadInvalidDurationRejected(t *testing.T) {
	t.Setenv("AUTH_SESSION_TTL", "not-a-duration")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() succeeded with a bad AUTH_SESSION_TTL, want error")
	}
}

func TestLoadInvalidBoolRejected(t *testing.T) {
	t.Setenv("AUTH_SIGNUP_ENABLED", "maybe")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() succeeded with a bad AUTH_SIGNUP_ENABLED, want error")
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
