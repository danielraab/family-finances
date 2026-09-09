package cli

import (
	"bytes"
	"context"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/entry"
	"at.draab/familyfinances/internal/storage/memory"
	"at.draab/familyfinances/internal/tag"
)

// testEmails stands in for what parseSeedEmails would produce from a
// CLI-supplied comma-separated list, for tests that don't exercise parsing
// itself.
var testEmails = []string{"tester1@draab.at", "tester2@draab.at", "tester3@draab.at"}

// newFixtureDeps builds the same shape of services Seed builds over
// Postgres, but over storage/memory, so seedTesters/generateFixtures are
// testable without a database. auth.Params carries real SessionTTL/
// SessionMaxTTL values (a zero-value Params mints a session that expires
// the instant it's created — Seed had exactly this bug until
// TestSeedTestersTokensAreImmediatelyUsable below caught it against a real
// server).
func newFixtureDeps() (auth.Store, *auth.Service, *account.Service, *category.Service, *tag.Service, *entry.Service) {
	authStore := memory.NewAuthStore()
	authSvc := auth.NewService(authStore, nil, nil, auth.Params{
		SessionTTL:    720 * time.Hour,
		SessionMaxTTL: 2160 * time.Hour,
	})
	accountSvc := account.NewService(memory.NewAccountStore())
	categorySvc := category.NewService(memory.NewCategoryStore())
	tagSvc := tag.NewService(memory.NewTagStore())
	entrySvc := entry.NewService(memory.NewEntryStore(), accountSvc, categorySvc, tagSvc)
	return authStore, authSvc, accountSvc, categorySvc, tagSvc, entrySvc
}

func TestSeedTestersCreatesAllThreeWithTester1Admin(t *testing.T) {
	authStore, authSvc, accountSvc, categorySvc, tagSvc, entrySvc := newFixtureDeps()
	rng := rand.New(rand.NewPCG(seedRNGSeed1, seedRNGSeed2))
	var stdout, stderr bytes.Buffer

	code := seedTesters(context.Background(), testEmails, authStore, authSvc, accountSvc, categorySvc, tagSvc, entrySvc, rng, 0, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("seedTesters exit = %d, stderr = %q", code, stderr.String())
	}

	for i, email := range testEmails {
		u, err := authStore.UserByEmail(context.Background(), email)
		if err != nil {
			t.Fatalf("UserByEmail(%q): %v", email, err)
		}
		wantAdmin := i == 0
		if u.IsAdmin != wantAdmin {
			t.Fatalf("%s.IsAdmin = %v, want %v", email, u.IsAdmin, wantAdmin)
		}
	}
}

func TestSeedTestersSeedsStarterAccountTypesAndCategories(t *testing.T) {
	authStore, authSvc, accountSvc, categorySvc, tagSvc, entrySvc := newFixtureDeps()
	rng := rand.New(rand.NewPCG(seedRNGSeed1, seedRNGSeed2))
	var stdout, stderr bytes.Buffer

	if code := seedTesters(context.Background(), testEmails, authStore, authSvc, accountSvc, categorySvc, tagSvc, entrySvc, rng, 0, &stdout, &stderr); code != 0 {
		t.Fatalf("seedTesters exit = %d, stderr = %q", code, stderr.String())
	}

	u, err := authStore.UserByEmail(context.Background(), testEmails[0])
	if err != nil {
		t.Fatal(err)
	}
	types, err := accountSvc.ListTypes(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != len(account.DefaultTypeTitles) {
		t.Fatalf("ListTypes = %d, want %d default types", len(types), len(account.DefaultTypeTitles))
	}
	cats, err := categorySvc.List(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != len(category.DefaultNames) {
		t.Fatalf("List categories = %d, want %d default categories", len(cats), len(category.DefaultNames))
	}
}

func TestSeedTestersPrintsSessionTokens(t *testing.T) {
	authStore, authSvc, accountSvc, categorySvc, tagSvc, entrySvc := newFixtureDeps()
	rng := rand.New(rand.NewPCG(seedRNGSeed1, seedRNGSeed2))
	var stdout, stderr bytes.Buffer

	if code := seedTesters(context.Background(), testEmails, authStore, authSvc, accountSvc, categorySvc, tagSvc, entrySvc, rng, 0, &stdout, &stderr); code != 0 {
		t.Fatalf("seedTesters exit = %d, stderr = %q", code, stderr.String())
	}

	out := stdout.String()
	for _, email := range testEmails {
		if !bytes.Contains([]byte(out), []byte(email)) {
			t.Fatalf("stdout missing %q: %s", email, out)
		}
	}
	if !bytes.Contains([]byte(out), []byte("session=")) {
		t.Fatalf("stdout missing session tokens: %s", out)
	}
}

// TestSeedTestersTokensAreImmediatelyUsable guards against the bug a
// zero-value auth.Params caused: IssueSession minted a token whose
// ExpiresAt was already in the past (SessionTTL 0 ⇒ ExpiresAt == the
// instant it was created), so it printed a token that Authenticate
// rejected right away. Confirms each printed token actually resolves back
// to its tester via authSvc.Authenticate, not just that stdout contains
// the string "session=".
func TestSeedTestersTokensAreImmediatelyUsable(t *testing.T) {
	authStore, authSvc, accountSvc, categorySvc, tagSvc, entrySvc := newFixtureDeps()
	rng := rand.New(rand.NewPCG(seedRNGSeed1, seedRNGSeed2))
	var stdout, stderr bytes.Buffer

	if code := seedTesters(context.Background(), testEmails, authStore, authSvc, accountSvc, categorySvc, tagSvc, entrySvc, rng, 0, &stdout, &stderr); code != 0 {
		t.Fatalf("seedTesters exit = %d, stderr = %q", code, stderr.String())
	}

	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		fields := strings.Fields(line)
		email := fields[0]
		token := strings.TrimPrefix(fields[len(fields)-1], "session=")

		got, err := authSvc.Authenticate(context.Background(), token)
		if err != nil {
			t.Fatalf("Authenticate(%s's token): %v", email, err)
		}
		if got.Email != email {
			t.Fatalf("Authenticate(%s's token) resolved to %s", email, got.Email)
		}
	}
}

func TestGenerateFixturesCreatesAccountsAndEntriesInRange(t *testing.T) {
	_, _, accountSvc, categorySvc, tagSvc, entrySvc := newFixtureDeps()
	ctx := context.Background()
	const ownerID = "u1"

	if err := accountSvc.SeedDefaults(ctx, ownerID); err != nil {
		t.Fatal(err)
	}
	if err := categorySvc.SeedDefaults(ctx, ownerID); err != nil {
		t.Fatal(err)
	}

	rng := rand.New(rand.NewPCG(seedRNGSeed1, seedRNGSeed2))
	if _, err := generateFixtures(ctx, ownerID, accountSvc, categorySvc, tagSvc, entrySvc, rng, 0); err != nil {
		t.Fatalf("generateFixtures: %v", err)
	}

	accounts, err := accountSvc.List(ctx, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) < 2 || len(accounts) > 4 {
		t.Fatalf("accounts = %d, want 2..4", len(accounts))
	}

	total := 0
	for _, acc := range accounts {
		list, _, err := entrySvc.List(ctx, ownerID, entry.Filter{AccountIDs: []string{acc.ID}, Limit: 200})
		if err != nil {
			t.Fatal(err)
		}
		if len(list) < 15 || len(list) > 60 {
			t.Fatalf("entries for account %s = %d, want 15..60", acc.ID, len(list))
		}
		total += len(list)
	}
	if total == 0 {
		t.Fatal("no entries generated at all")
	}
}

func TestGenerateFixturesCreatesTagsAndAttachesSome(t *testing.T) {
	_, _, accountSvc, categorySvc, tagSvc, entrySvc := newFixtureDeps()
	ctx := context.Background()
	const ownerID = "u1"

	if err := accountSvc.SeedDefaults(ctx, ownerID); err != nil {
		t.Fatal(err)
	}
	if err := categorySvc.SeedDefaults(ctx, ownerID); err != nil {
		t.Fatal(err)
	}

	rng := rand.New(rand.NewPCG(seedRNGSeed1, seedRNGSeed2))
	if _, err := generateFixtures(ctx, ownerID, accountSvc, categorySvc, tagSvc, entrySvc, rng, 0); err != nil {
		t.Fatalf("generateFixtures: %v", err)
	}

	tags, err := tagSvc.List(ctx, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != len(tagNamePool) {
		t.Fatalf("tags = %d, want %d (one per tagNamePool entry)", len(tags), len(tagNamePool))
	}

	accounts, err := accountSvc.List(ctx, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	taggedEntries := 0
	for _, acc := range accounts {
		list, _, err := entrySvc.List(ctx, ownerID, entry.Filter{AccountIDs: []string{acc.ID}, Limit: 200})
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range list {
			if len(e.TagIDs) > 2 {
				t.Fatalf("entry %s has %d tags, want at most 2", e.ID, len(e.TagIDs))
			}
			if len(e.TagIDs) > 0 {
				taggedEntries++
			}
		}
	}
	if taggedEntries == 0 {
		t.Fatal("no entry got any tag attached")
	}
}

func TestGenerateFixturesSetsFinancialInstitute(t *testing.T) {
	_, _, accountSvc, categorySvc, tagSvc, entrySvc := newFixtureDeps()
	ctx := context.Background()
	const ownerID = "u1"

	if err := accountSvc.SeedDefaults(ctx, ownerID); err != nil {
		t.Fatal(err)
	}
	if err := categorySvc.SeedDefaults(ctx, ownerID); err != nil {
		t.Fatal(err)
	}

	rng := rand.New(rand.NewPCG(seedRNGSeed1, seedRNGSeed2))
	if _, err := generateFixtures(ctx, ownerID, accountSvc, categorySvc, tagSvc, entrySvc, rng, 0); err != nil {
		t.Fatalf("generateFixtures: %v", err)
	}

	accounts, err := accountSvc.List(ctx, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	for _, acc := range accounts {
		if acc.FinancialInstitute == "" {
			t.Fatalf("account %s has empty FinancialInstitute", acc.ID)
		}
	}
}

func TestGenerateFixturesIsDeterministic(t *testing.T) {
	ctx := context.Background()

	run := func() []string {
		_, _, accountSvc, categorySvc, tagSvc, entrySvc := newFixtureDeps()
		const ownerID = "u1"
		if err := accountSvc.SeedDefaults(ctx, ownerID); err != nil {
			t.Fatal(err)
		}
		if err := categorySvc.SeedDefaults(ctx, ownerID); err != nil {
			t.Fatal(err)
		}
		rng := rand.New(rand.NewPCG(seedRNGSeed1, seedRNGSeed2))
		if _, err := generateFixtures(ctx, ownerID, accountSvc, categorySvc, tagSvc, entrySvc, rng, 0); err != nil {
			t.Fatal(err)
		}
		accounts, err := accountSvc.List(ctx, ownerID)
		if err != nil {
			t.Fatal(err)
		}
		titles := make([]string, len(accounts))
		for i, a := range accounts {
			titles[i] = a.Title
		}
		return titles
	}

	first := run()
	second := run()
	if len(first) != len(second) {
		t.Fatalf("account count differs across runs: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("account %d title differs across runs: %q vs %q", i, first[i], second[i])
		}
	}
}

func TestParseSeedEmailsSplitsTrimsAndNormalizes(t *testing.T) {
	got, err := parseSeedEmails(" Tester1@Draab.at ,tester2@draab.at,  tester3@draab.at")
	if err != nil {
		t.Fatalf("parseSeedEmails: %v", err)
	}
	want := []string{"tester1@draab.at", "tester2@draab.at", "tester3@draab.at"}
	if len(got) != len(want) {
		t.Fatalf("parseSeedEmails = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseSeedEmails[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseSeedEmailsRejectsEmpty(t *testing.T) {
	if _, err := parseSeedEmails(""); err == nil {
		t.Fatal("parseSeedEmails(\"\") = nil error, want an error")
	}
	if _, err := parseSeedEmails(" , , "); err == nil {
		t.Fatal("parseSeedEmails(\" , , \") = nil error, want an error")
	}
}

func TestParseSeedEmailsRejectsInvalidAddress(t *testing.T) {
	if _, err := parseSeedEmails("tester1@draab.at,not-an-email"); err == nil {
		t.Fatal("parseSeedEmails with an invalid address = nil error, want an error")
	}
}

func TestParseSeedEmailsRejectsDuplicates(t *testing.T) {
	if _, err := parseSeedEmails("tester1@draab.at,Tester1@Draab.at"); err == nil {
		t.Fatal("parseSeedEmails with a case-insensitive duplicate = nil error, want an error")
	}
}

func TestRandomAmountSalaryPositiveOthersNegative(t *testing.T) {
	rng := rand.New(rand.NewPCG(seedRNGSeed1, seedRNGSeed2))
	for i := 0; i < 50; i++ {
		if v := randomAmount(rng, "Salary"); v <= 0 {
			t.Fatalf("Salary amount = %d, want positive", v)
		}
	}
	for _, name := range category.DefaultNames[1:] {
		for i := 0; i < 20; i++ {
			if v := randomAmount(rng, name); v >= 0 {
				t.Fatalf("%s amount = %d, want negative", name, v)
			}
		}
	}
}

func TestParseSeedFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantYes    bool
		wantEmails string
		wantEnt    int
	}{
		{name: "yes plus emails", args: []string{"--yes", "a@b.com,c@d.com"}, wantYes: true, wantEmails: "a@b.com,c@d.com"},
		{name: "entries flag, space form", args: []string{"--yes", "--entries", "10000", "a@b.com"}, wantYes: true, wantEmails: "a@b.com", wantEnt: 10000},
		{name: "entries flag, equals form", args: []string{"--yes", "--entries=500", "a@b.com"}, wantYes: true, wantEmails: "a@b.com", wantEnt: 500},
		{name: "flags after nothing else", args: []string{"a@b.com"}, wantYes: false, wantEmails: "a@b.com"},
		{name: "bare", args: nil},
		{name: "negative entries", args: []string{"--yes", "--entries", "-1", "a@b.com"}, wantErr: true},
		{name: "unknown flag", args: []string{"--nope", "a@b.com"}, wantErr: true},
		{name: "extra positional", args: []string{"--yes", "a@b.com", "surprise"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSeedFlags(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseSeedFlags(%v) = %+v, want error", tc.args, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSeedFlags(%v): %v", tc.args, err)
			}
			if got.yes != tc.wantYes || got.emails != tc.wantEmails || got.entries != tc.wantEnt {
				t.Fatalf("parseSeedFlags(%v) = %+v, want {yes:%v emails:%q entries:%d}",
					tc.args, got, tc.wantYes, tc.wantEmails, tc.wantEnt)
			}
		})
	}
}

func TestGenerateFixturesEntriesPerUserHitsTargetExactly(t *testing.T) {
	_, _, accountSvc, categorySvc, tagSvc, entrySvc := newFixtureDeps()
	ctx := context.Background()
	const ownerID = "u1"
	const target = 317 // deliberately not divisible by any likely account count

	if err := accountSvc.SeedDefaults(ctx, ownerID); err != nil {
		t.Fatal(err)
	}
	if err := categorySvc.SeedDefaults(ctx, ownerID); err != nil {
		t.Fatal(err)
	}

	rng := rand.New(rand.NewPCG(seedRNGSeed1, seedRNGSeed2))
	created, err := generateFixtures(ctx, ownerID, accountSvc, categorySvc, tagSvc, entrySvc, rng, target)
	if err != nil {
		t.Fatalf("generateFixtures: %v", err)
	}
	if created != target {
		t.Fatalf("generateFixtures returned %d, want %d", created, target)
	}

	// Count every persisted entry, paging through with the keyset cursor
	// (List clamps a page to 200 rows).
	total := 0
	var after *entry.Cursor
	for {
		page, cursor, err := entrySvc.List(ctx, ownerID, entry.Filter{Limit: 200, After: after})
		if err != nil {
			t.Fatal(err)
		}
		total += len(page)
		if cursor == nil || len(page) == 0 {
			break
		}
		after = cursor
	}
	if total != target {
		t.Fatalf("entries persisted = %d, want %d", total, target)
	}
}
