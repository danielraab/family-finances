package cli

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"os"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/config"
	"at.draab/familyfinances/internal/entry"
	"at.draab/familyfinances/internal/storage/postgres"
	"at.draab/familyfinances/internal/tag"
)

// seedEmails is the fixed set of testers `seed` creates, in this order —
// the first lands as the bootstrap admin (see fixtures.go).
var seedEmails = []string{"tester1@draab.at", "tester2@draab.at", "tester3@draab.at"}

// Fixed RNG seed: `seed --yes` is meant to be reproducible across runs, not
// a fresh random fixture every time.
const seedRNGSeed1, seedRNGSeed2 = 0x5EED5EED, 0xFACADE42

const seedUsage = "usage: server seed [--yes]"

// Seed runs the `seed` subcommand. Bare (`server seed`) prints what a reset
// would wipe and exits without touching the database; `server seed --yes`
// truncates every data table and (re)creates the three testers with
// randomly generated fixtures. It builds its own config, pool, and stores,
// the same self-contained shape Admin uses.
func Seed(ctx context.Context, args []string) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed: load config:", err)
		return 1
	}
	if cfg.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "seed: DATABASE_URL is required and unset")
		return 1
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed: connect database:", err)
		return 1
	}
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool); err != nil {
		fmt.Fprintln(os.Stderr, "seed: apply migrations:", err)
		return 1
	}

	if len(args) != 1 || args[0] != "--yes" {
		return printResetPlan(ctx, pool, os.Stdout, os.Stderr)
	}

	if err := postgres.ResetAll(ctx, pool); err != nil {
		fmt.Fprintln(os.Stderr, "seed: reset database:", err)
		return 1
	}

	authStore := postgres.NewAuthStore(pool)
	// SessionTTL/SessionMaxTTL matter here even though nothing else in
	// auth.Params does: IssueSession computes each token's expiry from
	// them, and a zero-value Params would mint sessions that expire
	// instantly (SessionTTL 0 ⇒ ExpiresAt == now).
	authSvc := auth.NewService(authStore, nil, nil, auth.Params{
		SessionTTL:    cfg.Auth.SessionTTL,
		SessionMaxTTL: cfg.Auth.SessionMaxTTL,
	})
	accountSvc := account.NewService(postgres.NewAccountStore(pool))
	categorySvc := category.NewService(postgres.NewCategoryStore(pool))
	tagSvc := tag.NewService(postgres.NewTagStore(pool))
	entrySvc := entry.NewService(postgres.NewEntryStore(pool), accountSvc, categorySvc, tagSvc)

	rng := rand.New(rand.NewPCG(seedRNGSeed1, seedRNGSeed2))

	return seedTesters(ctx, authStore, authSvc, accountSvc, categorySvc, entrySvc, rng, os.Stdout, os.Stderr)
}

// printResetPlan shows what `--yes` would wipe, without touching anything.
// Any invocation other than exactly "--yes" (no args, a typo, "-y", …)
// lands here rather than being treated as a usage error — safer to default
// to inert than to guess at what the caller meant.
func printResetPlan(ctx context.Context, pool *postgres.Pool, stdout, stderr io.Writer) int {
	counts, err := postgres.TableCounts(ctx, pool)
	if err != nil {
		fmt.Fprintln(stderr, "seed: list tables:", err)
		return 1
	}
	fmt.Fprintln(stdout, seedUsage)
	fmt.Fprintln(stdout, "server seed --yes would TRUNCATE every table below (schema_migrations excluded),")
	fmt.Fprintln(stdout, "then create tester1/2/3@draab.at with random fixtures:")
	for _, c := range counts {
		fmt.Fprintf(stdout, "  %-20s %d rows\n", c.Table, c.Rows)
	}
	return 2
}
