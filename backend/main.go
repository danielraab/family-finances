package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/cli"
	"at.draab/familyfinances/internal/config"
	"at.draab/familyfinances/internal/entry"
	"at.draab/familyfinances/internal/httpapi"
	"at.draab/familyfinances/internal/mailer"
	"at.draab/familyfinances/internal/oidcauth"
	"at.draab/familyfinances/internal/settings"
	"at.draab/familyfinances/internal/storage/postgres"
	"at.draab/familyfinances/internal/tag"
)

func main() {
	ctx := context.Background()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			os.Exit(healthcheck())
		case "admin":
			os.Exit(cli.Admin(ctx, os.Args[2:]))
		}
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required and unset")
		os.Exit(1)
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool); err != nil {
		slog.Error("apply migrations", "error", err)
		os.Exit(1)
	}

	staticFS, err := fs.Sub(staticFiles, "static/out")
	if err != nil {
		slog.Error("sub static fs", "error", err)
		os.Exit(1)
	}

	settingsSvc, settingsHandler := buildSettings(pool)

	authSvc, authHandler, err := buildAuth(ctx, cfg, pool, settingsSvc)
	if err != nil {
		slog.Error("build auth", "error", err)
		os.Exit(1)
	}

	accountSvc, accountHandler := buildAccount(pool)
	categorySvc, categoryHandler := buildCategory(pool)
	tagSvc, tagHandler := buildTag(pool)
	entryHandler := buildEntry(pool, accountSvc, categorySvc, tagSvc)

	srv := httpapi.New(cfg, httpapi.Deps{
		Static:          staticFS,
		DB:              pool,
		Auth:            authSvc,
		AuthHandler:     authHandler,
		SettingsHandler: settingsHandler,
		AccountHandler:  accountHandler,
		CategoryHandler: categoryHandler,
		TagHandler:      tagHandler,
		EntryHandler:    entryHandler,
		OpenAPISpec:     openAPISpec,
	})

	if err := run(srv); err != nil {
		slog.Error("server", "error", err)
		os.Exit(1)
	}
}

// buildSettings constructs the settings service and its HTTP handler over the
// Postgres store.
func buildSettings(pool *postgres.Pool) (*settings.Service, http.Handler) {
	store := postgres.NewSettingsStore(pool)
	svc := settings.NewService(store)
	handler := settings.NewHandler(svc, settings.HandlerOptions{RenderError: httpapi.WriteError})
	return svc, handler
}

// buildAccount constructs the account service and its HTTP handler over the
// Postgres store.
func buildAccount(pool *postgres.Pool) (*account.Service, http.Handler) {
	store := postgres.NewAccountStore(pool)
	svc := account.NewService(store)
	handler := account.NewHandler(svc, account.HandlerOptions{RenderError: httpapi.WriteError})
	return svc, handler
}

// buildCategory constructs the category service and its HTTP handler over
// the Postgres store.
func buildCategory(pool *postgres.Pool) (*category.Service, http.Handler) {
	store := postgres.NewCategoryStore(pool)
	svc := category.NewService(store)
	handler := category.NewHandler(svc, category.HandlerOptions{RenderError: httpapi.WriteError})
	return svc, handler
}

// buildTag constructs the tag service and its HTTP handler over the
// Postgres store.
func buildTag(pool *postgres.Pool) (*tag.Service, http.Handler) {
	store := postgres.NewTagStore(pool)
	svc := tag.NewService(store)
	handler := tag.NewHandler(svc, tag.HandlerOptions{RenderError: httpapi.WriteError})
	return svc, handler
}

// buildEntry constructs the entry service and its HTTP handler over the
// Postgres store, wiring accountSvc/categorySvc/tagSvc in as its
// AccountLookup/CategoryLookup/TagLookup dependencies (see design.md's
// package-boundaries decision).
func buildEntry(pool *postgres.Pool, accountSvc *account.Service, categorySvc *category.Service, tagSvc *tag.Service) http.Handler {
	store := postgres.NewEntryStore(pool)
	svc := entry.NewService(store, accountSvc, categorySvc, tagSvc)
	return entry.NewHandler(svc, entry.HandlerOptions{RenderError: httpapi.WriteError})
}

// buildAuth constructs the auth service and its HTTP handler: the Postgres
// store, the SMTP mailer, and — only when OIDC_ISSUER is set — a discovered
// OIDC client. settingsSvc is wired in as the raw-language-preference source
// for GET /api/auth/me (see internal/settings' design note).
func buildAuth(ctx context.Context, cfg config.Config, pool *postgres.Pool, settingsSvc *settings.Service) (*auth.Service, http.Handler, error) {
	store := postgres.NewAuthStore(pool)

	mail := mailer.New(mailer.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		TLS:      mailer.TLSMode(cfg.SMTP.TLS),
	})

	var oidcClient auth.OIDCClient
	// Both are required for a working flow; an issuer without a client id
	// discovers fine but fails at token exchange, so treat it as unconfigured.
	if cfg.OIDC.Issuer != "" && cfg.OIDC.ClientID != "" {
		c, err := oidcauth.New(ctx, oidcauth.Config{
			Issuer:       cfg.OIDC.Issuer,
			ClientID:     cfg.OIDC.ClientID,
			ClientSecret: cfg.OIDC.ClientSecret,
			RedirectURL:  cfg.Auth.BaseURL + "/api/auth/oidc/callback",
			Scopes:       cfg.OIDC.Scopes,
		})
		if err != nil {
			return nil, nil, err
		}
		oidcClient = c
	}

	svc := auth.NewService(store, mail, oidcClient, auth.Params{
		BaseURL:             cfg.Auth.BaseURL,
		SessionTTL:          cfg.Auth.SessionTTL,
		SessionMaxTTL:       cfg.Auth.SessionMaxTTL,
		SignupEnabled:       cfg.Auth.SignupEnabled,
		AllowedEmailDomains: cfg.Auth.AllowedEmailDomains,
		InviteEnabled:       cfg.Auth.InviteEnabled,
		InviteTTL:           cfg.Auth.InviteTTL,
		MagicLinkTTL:        cfg.Auth.MagicLinkTTL,
		OIDCIssuer:          cfg.OIDC.Issuer,
		OIDCLabel:           cfg.OIDC.Label,
	}, auth.WithLanguageLookup(settingsSvc))

	handler := auth.NewHandler(svc, auth.HandlerOptions{
		RenderError:  httpapi.WriteError,
		CookieSecure: cfg.Auth.CookieSecure,
	})
	return svc, handler, nil
}

// run starts srv and blocks until SIGINT/SIGTERM, then shuts it down
// gracefully.
func run(srv *http.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
