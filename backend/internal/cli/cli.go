// Package cli implements the backend's CLI subcommands other than the Docker
// healthcheck probe. main.go dispatches "admin …" here; the probe stays in
// package main (healthcheck.go) so the root package keeps its
// main.go/healthcheck.go/embed.go shape.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/config"
	"at.draab/familyfinances/internal/storage/postgres"
)

const adminUsage = "usage: server admin grant <email> | revoke <email> | list"

// Admin runs the `admin` subcommand. args is everything after "admin" (e.g.
// {"grant", "a@b.com"}). It builds config, a Postgres pool, and the auth store
// itself, then returns a process exit code.
func Admin(ctx context.Context, args []string) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "admin: load config:", err)
		return 1
	}
	if cfg.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "admin: DATABASE_URL is required and unset")
		return 1
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "admin: connect database:", err)
		return 1
	}
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool); err != nil {
		fmt.Fprintln(os.Stderr, "admin: apply migrations:", err)
		return 1
	}

	return run(ctx, args, postgres.NewAuthStore(pool), os.Stdout, os.Stderr)
}

// run is the store-agnostic command logic, split out so tests drive it with an
// in-memory store.
func run(ctx context.Context, args []string, store auth.Store, stdout, stderr io.Writer) int {
	svc := auth.NewService(store, nil, nil, auth.Params{})

	if len(args) == 0 {
		fmt.Fprintln(stderr, adminUsage)
		return 2
	}

	switch args[0] {
	case "grant", "revoke":
		if len(args) != 2 {
			fmt.Fprintln(stderr, adminUsage)
			return 2
		}
		grant := args[0] == "grant"
		email := auth.NormalizeEmail(args[1])
		if err := svc.SetAdmin(ctx, email, grant); err != nil {
			if errors.Is(err, auth.ErrNotFound) {
				fmt.Fprintf(stderr, "admin: no user with email %q\n", email)
				return 1
			}
			fmt.Fprintln(stderr, "admin:", err)
			return 1
		}
		verb := "granted to"
		if !grant {
			verb = "revoked from"
		}
		fmt.Fprintf(stdout, "admin %s %s\n", verb, email)
		return 0

	case "list":
		admins, err := svc.ListAdmins(ctx)
		if err != nil {
			fmt.Fprintln(stderr, "admin:", err)
			return 1
		}
		for _, e := range admins {
			fmt.Fprintln(stdout, e)
		}
		return 0

	default:
		fmt.Fprintln(stderr, adminUsage)
		return 2
	}
}
