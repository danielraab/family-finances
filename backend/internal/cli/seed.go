package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"strings"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/config"
	"at.draab/familyfinances/internal/entry"
	"at.draab/familyfinances/internal/storage/postgres"
	"at.draab/familyfinances/internal/tag"
)

// Fixed RNG seed: `seed --yes` is meant to be reproducible across runs, not
// a fresh random fixture every time.
const seedRNGSeed1, seedRNGSeed2 = 0x5EED5EED, 0xFACADE42

const seedUsage = "usage: server seed --yes [--entries N] <email1,email2,...>"

// seedFlags is the parsed form of `seed`'s arguments. yes and emails must
// both be present for a run to touch the database; entries is optional
// (0 = the default per-account random count).
type seedFlags struct {
	yes     bool
	emails  string
	entries int
}

// parseSeedFlags parses `seed`'s args with the standard flag package:
// `--yes` (bool), `--entries N` (int, per-user transaction-entry target,
// spread across that user's accounts; 0 keeps the default random count per
// account), and one trailing positional — the comma-separated email list.
// A malformed flag or a negative --entries is an error; a missing --yes or
// email list is not (Seed treats that as "print the plan, change nothing").
func parseSeedFlags(args []string) (seedFlags, error) {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // Seed prints its own usage
	yes := fs.Bool("yes", false, "actually reset the database and seed")
	entries := fs.Int("entries", 0, "transaction entries per seeded user")
	if err := fs.Parse(args); err != nil {
		return seedFlags{}, err
	}
	if *entries < 0 {
		return seedFlags{}, fmt.Errorf("--entries must be >= 0")
	}
	rest := fs.Args()
	if len(rest) > 1 {
		return seedFlags{}, fmt.Errorf("unexpected extra arguments: %v", rest[1:])
	}
	f := seedFlags{yes: *yes, entries: *entries}
	if len(rest) == 1 {
		f.emails = rest[0]
	}
	return f, nil
}

// Seed runs the `seed` subcommand. Bare (`server seed`), or anything other
// than exactly "--yes" followed by a comma-separated email list, prints
// what a reset would wipe and exits without touching the database;
// `server seed --yes <emails>` truncates every data table and creates one
// user per email with randomly generated fixtures. It builds its own
// config, pool, and stores, the same self-contained shape Admin uses.
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

	flags, err := parseSeedFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		fmt.Fprintln(os.Stderr, seedUsage)
		return 2
	}
	if !flags.yes || flags.emails == "" {
		return printResetPlan(ctx, pool, os.Stdout, os.Stderr)
	}

	emails, err := parseSeedEmails(flags.emails)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		fmt.Fprintln(os.Stderr, seedUsage)
		return 2
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

	return seedTesters(ctx, emails, authStore, authSvc, accountSvc, categorySvc, tagSvc, entrySvc, rng, flags.entries, os.Stdout, os.Stderr)
}

// parseSeedEmails splits raw on commas, trims and normalizes each address
// (auth.NormalizeEmail — lower-cased, whitespace-trimmed, matching how
// `admin grant/revoke` already treat an email argument), validates each
// with auth.ValidateEmail, and rejects an empty list or a duplicate
// address (a duplicate would otherwise surface later as a confusing
// identity-conflict error from CreateUserWithIdentity instead of a clear
// one here).
func parseSeedEmails(raw string) ([]string, error) {
	var emails []string
	seen := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		email := auth.NormalizeEmail(part)
		if email == "" {
			continue
		}
		if err := auth.ValidateEmail(email); err != nil {
			return nil, fmt.Errorf("invalid email %q: %w", part, err)
		}
		if seen[email] {
			return nil, fmt.Errorf("duplicate email %q", email)
		}
		seen[email] = true
		emails = append(emails, email)
	}
	if len(emails) == 0 {
		return nil, fmt.Errorf("no emails given")
	}
	return emails, nil
}

// printResetPlan shows what `--yes <emails>` would wipe, without touching
// anything. Any invocation other than exactly "--yes" followed by a
// comma-separated email list (no args, a typo, an empty/invalid email
// list, …) lands here rather than being treated as a usage error — safer
// to default to inert than to guess at what the caller meant.
func printResetPlan(ctx context.Context, pool *postgres.Pool, stdout, stderr io.Writer) int {
	counts, err := postgres.TableCounts(ctx, pool)
	if err != nil {
		fmt.Fprintln(stderr, "seed: list tables:", err)
		return 1
	}
	fmt.Fprintln(stdout, seedUsage)
	fmt.Fprintln(stdout, "would TRUNCATE every table below (schema_migrations excluded),")
	fmt.Fprintln(stdout, "then create one user per given email with random fixtures:")
	for _, c := range counts {
		fmt.Fprintf(stdout, "  %-20s %d rows\n", c.Table, c.Rows)
	}
	return 2
}
