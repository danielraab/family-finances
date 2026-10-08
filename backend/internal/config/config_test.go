package config_test

import (
	"log/slog"
	"strings"
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
		"OIDC_LABEL", "AUTH_PASSKEY_REAUTH_WINDOW",
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
	if cfg.Auth.PasskeyReauthWindow != 5*time.Minute {
		t.Errorf("PasskeyReauthWindow = %s, want 5m", cfg.Auth.PasskeyReauthWindow)
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

func TestLoadPasskeyReauthWindowOverride(t *testing.T) {
	t.Setenv("AUTH_PASSKEY_REAUTH_WINDOW", "90s")

	cfg := load(t)
	if cfg.Auth.PasskeyReauthWindow != 90*time.Second {
		t.Errorf("PasskeyReauthWindow = %s, want 90s", cfg.Auth.PasskeyReauthWindow)
	}
}

func TestLoadInvalidPasskeyReauthWindowRejected(t *testing.T) {
	for _, v := range []string{"0s", "-1m", "soon"} {
		t.Run(v, func(t *testing.T) {
			t.Setenv("AUTH_PASSKEY_REAUTH_WINDOW", v)
			if _, err := config.Load(); err == nil {
				t.Fatalf("Load() succeeded with AUTH_PASSKEY_REAUTH_WINDOW=%s, want error", v)
			}
		})
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

func TestLoadRateLimitAndCleanupDefaults(t *testing.T) {
	for _, k := range []string{
		"RATE_LIMIT_IP_ENABLED", "RATE_LIMIT_IP_REQUESTS", "RATE_LIMIT_IP_WINDOW",
		"RATE_LIMIT_EMAIL_REQUESTS", "RATE_LIMIT_EMAIL_WINDOW",
		"AUTH_CLEANUP_INTERVAL", "AUTH_TRUSTED_PROXIES",
	} {
		t.Setenv(k, "")
	}
	cfg := load(t)
	rl := cfg.RateLimit
	if !rl.IPEnabled || rl.IPRequests != 20 || rl.IPWindow != time.Minute ||
		rl.EmailRequests != 5 || rl.EmailWindow != 15*time.Minute {
		t.Errorf("RateLimit = %+v, want on, 20/1m, 5/15m", rl)
	}
	if cfg.Auth.CleanupInterval != 15*time.Minute {
		t.Errorf("CleanupInterval = %s, want 15m", cfg.Auth.CleanupInterval)
	}
	if cfg.Auth.TrustedProxies != nil {
		t.Errorf("TrustedProxies = %v, want none", cfg.Auth.TrustedProxies)
	}
}

func TestLoadRateLimitAndCleanupOverrides(t *testing.T) {
	t.Setenv("RATE_LIMIT_IP_ENABLED", "false")
	t.Setenv("RATE_LIMIT_IP_REQUESTS", "100")
	t.Setenv("RATE_LIMIT_IP_WINDOW", "30s")
	t.Setenv("RATE_LIMIT_EMAIL_REQUESTS", "2")
	t.Setenv("RATE_LIMIT_EMAIL_WINDOW", "1h")
	t.Setenv("AUTH_CLEANUP_INTERVAL", "1h")
	t.Setenv("AUTH_TRUSTED_PROXIES", "10.0.0.0/8, 192.168.1.5 ,fd00::/8, ::1")

	cfg := load(t)
	rl := cfg.RateLimit
	if rl.IPEnabled || rl.IPRequests != 100 || rl.IPWindow != 30*time.Second ||
		rl.EmailRequests != 2 || rl.EmailWindow != time.Hour {
		t.Errorf("RateLimit = %+v", rl)
	}
	if cfg.Auth.CleanupInterval != time.Hour {
		t.Errorf("CleanupInterval = %s, want 1h", cfg.Auth.CleanupInterval)
	}
	var got []string
	for _, p := range cfg.Auth.TrustedProxies {
		got = append(got, p.String())
	}
	if want := []string{"10.0.0.0/8", "192.168.1.5/32", "fd00::/8", "::1/128"}; !equal(got, want) {
		t.Errorf("TrustedProxies = %v, want %v", got, want)
	}
}

func TestLoadInvalidRateLimitAndCleanupRejected(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"RATE_LIMIT_IP_REQUESTS", "0"},
		{"RATE_LIMIT_IP_REQUESTS", "-3"},
		{"RATE_LIMIT_IP_REQUESTS", "many"},
		{"RATE_LIMIT_EMAIL_REQUESTS", "0"},
		{"RATE_LIMIT_IP_WINDOW", "0s"},
		{"RATE_LIMIT_EMAIL_WINDOW", "soon"},
		{"RATE_LIMIT_IP_ENABLED", "maybe"},
		{"AUTH_CLEANUP_INTERVAL", "0s"},
		{"AUTH_TRUSTED_PROXIES", "10.0.0.0/33"},
		{"AUTH_TRUSTED_PROXIES", "not-an-ip"},
	} {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if _, err := config.Load(); err == nil {
				t.Fatalf("Load() accepted %s=%s", tc.key, tc.value)
			}
		})
	}
}

func TestLoadLogDefaults(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("LOG_REQUESTS", "")
	cfg := load(t)
	if cfg.Log.Level != slog.LevelInfo || cfg.Log.Requests != config.RequestLogAll {
		t.Fatalf("Log = %+v, want info/all", cfg.Log)
	}
}

func TestLoadLogValues(t *testing.T) {
	for raw, want := range map[string]slog.Level{
		"debug": slog.LevelDebug, "INFO": slog.LevelInfo, "warn": slog.LevelWarn,
		"Warning": slog.LevelWarn, " error ": slog.LevelError,
	} {
		t.Setenv("LOG_LEVEL", raw)
		if got := load(t).Log.Level; got != want {
			t.Errorf("LOG_LEVEL=%q → %v, want %v", raw, got, want)
		}
	}
	t.Setenv("LOG_LEVEL", "")
	for raw, want := range map[string]config.RequestLogMode{
		"all": config.RequestLogAll, "Errors": config.RequestLogErrors, "OFF": config.RequestLogOff,
	} {
		t.Setenv("LOG_REQUESTS", raw)
		if got := load(t).Log.Requests; got != want {
			t.Errorf("LOG_REQUESTS=%q → %q, want %q", raw, got, want)
		}
	}
}

func TestLoadInvalidLogRejected(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"LOG_LEVEL", "verbose"}, {"LOG_LEVEL", "3"}, {"LOG_REQUESTS", "some"}, {"LOG_REQUESTS", "true"},
	} {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			_, err := config.Load()
			if err == nil || !strings.Contains(err.Error(), tc.key) {
				t.Fatalf("Load() err = %v, want an error naming %s", err, tc.key)
			}
		})
	}
}
