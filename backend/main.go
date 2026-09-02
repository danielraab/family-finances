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

	"at.draab/familyfinances/internal/config"
	"at.draab/familyfinances/internal/httpapi"
	"at.draab/familyfinances/internal/storage/postgres"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}

	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required and unset")
		os.Exit(1)
	}

	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := postgres.Migrate(context.Background(), pool); err != nil {
		slog.Error("apply migrations", "error", err)
		os.Exit(1)
	}

	staticFS, err := fs.Sub(staticFiles, "static/out")
	if err != nil {
		slog.Error("sub static fs", "error", err)
		os.Exit(1)
	}

	srv := httpapi.New(cfg, httpapi.Deps{Static: staticFS, DB: pool})

	if err := run(srv); err != nil {
		slog.Error("server", "error", err)
		os.Exit(1)
	}
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
