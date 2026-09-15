package cli

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"sort"
	"time"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/dashboard"
	"at.draab/familyfinances/internal/entry"
	"at.draab/familyfinances/internal/recurringtransaction"
	"at.draab/familyfinances/internal/tag"
)

// accountCurrencies is the small pool of currencies a generated account
// randomly draws from.
var accountCurrencies = []string{"EUR", "USD", "GBP"}

// fixtureAccountTypes is the pool of free-text type labels a generated
// account randomly draws from — the former seeded default set, now just
// strings written straight onto the account.
var fixtureAccountTypes = []string{
	"Checking", "Savings", "Cash", "Credit Card", "Loan", "Investment",
}

// financialInstitutePool is the small pool of bank names a generated
// account's optional financial_institute randomly draws from.
var financialInstitutePool = []string{
	"N26", "Sparkasse", "Deutsche Bank", "Revolut", "ING", "Erste Bank", "Raiffeisen",
}

// tagNamePool is the fixed set of tags created for each seeded user, then
// randomly attached to a subset of their generated entries.
var tagNamePool = []string{"recurring", "shared", "reimbursable", "business", "vacation", "gift"}

// categoryTitlePools gives each seeded default category (category.DefaultNames)
// a few plausible entry titles, so a generated listing doesn't read as
// obviously synthetic. A category outside this set (there shouldn't be one,
// since fixtures only ever draw from a fresh user's seeded categories) falls
// back to its own name.
// recurringFixture is one template in recurringFixturePool. categoryName
// must be one of category.DefaultNames — every seeded user has that
// category already (via categorySvc.SeedDefaults). amount is signed and
// unscaled (e.g. -15.99); startsOnDaysOffset is relative to "today" —
// negative means the template has no linked entries yet and its
// starts_on has already passed (an overdue next_suggested_date, with no
// action needed to reproduce it), positive means it's still upcoming —
// so a freshly seeded dashboard exercises both the tinted-overdue-row and
// the ordinary-upcoming cases of the recurring-transaction preview out of
// the box.
type recurringFixture struct {
	title              string
	categoryName       string
	unit               recurringtransaction.Unit
	intervalCount      int
	amount             float64
	startsOnDaysOffset int
}

// recurringFixturePool is the pool of recurring-transaction templates a
// seeded user draws a random subset from — plausible bills/income shaped
// the same way categoryTitlePools' entries are.
var recurringFixturePool = []recurringFixture{
	{"Rent", "Rent", recurringtransaction.UnitMonth, 1, -1200, -4},
	{"Netflix", "Entertainment", recurringtransaction.UnitMonth, 1, -15.99, 9},
	{"Spotify", "Entertainment", recurringtransaction.UnitMonth, 1, -9.99, 3},
	{"Gym membership", "Health", recurringtransaction.UnitMonth, 1, -34.90, -2},
	{"Salary", "Salary", recurringtransaction.UnitMonth, 1, 3200, 22},
	{"Car insurance", "Transportation", recurringtransaction.UnitYear, 1, -540, 40},
	{"Internet bill", "Utilities", recurringtransaction.UnitMonth, 1, -39.90, 7},
	{"Phone plan", "Utilities", recurringtransaction.UnitMonth, 1, -24.90, -1},
}

var categoryTitlePools = map[string][]string{
	"Salary":         {"Monthly salary", "Paycheck"},
	"Groceries":      {"Supermarket", "Grocery run", "Farmers market"},
	"Rent":           {"Monthly rent"},
	"Utilities":      {"Electric bill", "Water bill", "Internet bill"},
	"Transportation": {"Fuel", "Train ticket", "Parking"},
	"Entertainment":  {"Cinema", "Streaming subscription", "Concert tickets"},
	"Health":         {"Pharmacy", "Doctor visit"},
	"Other":          {"Misc purchase", "Cash withdrawal"},
}

// seedTesters creates one user per email (in the given order — the first
// lands as the bootstrap admin, since the reset just emptied users)
// directly via authStore.CreateUserWithIdentity — a pre-verified email
// identity each, the same mechanism this package's own tests already use,
// bypassing the magic-link/OIDC flows entirely. Because that goes straight
// to the store, internal/auth's NewUserHooks (wired only inside
// auth.Service) never fire here; internal/cli already imports
// category directly (the same way main.go's builders do), so it calls each
// user's categorySvc.SeedDefaults itself instead of routing through that
// indirection — NewUserHook exists to let internal/auth notify category
// without importing it, a problem internal/cli doesn't have. Account types
// are no longer seeded (a type is just a text label on the account now).
// authSvc is used only to mint each tester's session token via
// IssueSession. Prints (email, admin?, session token) per tester to stdout
// as it goes; returns 0 on success, or 1 with a stderr message on the
// first failure.
func seedTesters(
	ctx context.Context,
	emails []string,
	authStore auth.Store,
	authSvc *auth.Service,
	accountSvc *account.Service,
	categorySvc *category.Service,
	tagSvc *tag.Service,
	entrySvc *entry.Service,
	recurringSvc *recurringtransaction.Service,
	dashboardSvc *dashboard.Service,
	rng *rand.Rand,
	entriesPerUser int,
	stdout, stderr io.Writer,
) int {
	for _, email := range emails {
		user, _, err := authStore.CreateUserWithIdentity(ctx, auth.NewUser{Email: email}, auth.Identity{
			Kind:          auth.IdentityEmail,
			Email:         email,
			EmailVerified: true,
		})
		if err != nil {
			fmt.Fprintf(stderr, "seed: create %s: %v\n", email, err)
			return 1
		}

		if err := categorySvc.SeedDefaults(ctx, user.ID); err != nil {
			fmt.Fprintf(stderr, "seed: seed categories for %s: %v\n", email, err)
			return 1
		}
		counts, err := generateFixtures(ctx, user.ID, accountSvc, categorySvc, tagSvc, entrySvc, recurringSvc, dashboardSvc, rng, entriesPerUser)
		if err != nil {
			fmt.Fprintf(stderr, "seed: generate fixtures for %s: %v\n", email, err)
			return 1
		}

		token, err := authSvc.IssueSession(ctx, user.ID)
		if err != nil {
			fmt.Fprintf(stderr, "seed: issue session for %s: %v\n", email, err)
			return 1
		}

		admin := ""
		if user.IsAdmin {
			admin = " (admin)"
		}
		fmt.Fprintf(stdout, "%s%s entries=%d recurring=%d cards=%d session=%s\n",
			email, admin, counts.entries, counts.recurring, counts.cards, token)
	}
	return 0
}

// fixtureCounts tallies what generateFixtures created for one user, for
// seedTesters' summary line.
type fixtureCounts struct {
	entries   int
	recurring int
	cards     int
}

// generateFixtures creates a fixed pool of tags for ownerID (tagNamePool),
// then 2-4 random accounts (each a random seeded type + a random currency +
// a random financial institute + a random opening date within the past two
// years), then transaction entries per account (random seeded category,
// booking date within the past year, an amount/title shaped by that
// category — see randomAmount/randomTitle — and 0-2 random tags drawn from
// that user's tag pool). When entriesPerUser is 0 each account gets 15-60
// entries at random; when it is > 0 exactly that many entries are spread
// as evenly as possible across the accounts. Once accounts and entries
// exist, it also creates a handful of recurring-transaction templates
// (generateRecurringTransactions) and a starter set of dashboard cards
// (generateDashboardCards) so a freshly seeded user's /home isn't empty
// and exercises the recurring-transaction preview feature out of the box.
func generateFixtures(
	ctx context.Context,
	ownerID string,
	accountSvc *account.Service,
	categorySvc *category.Service,
	tagSvc *tag.Service,
	entrySvc *entry.Service,
	recurringSvc *recurringtransaction.Service,
	dashboardSvc *dashboard.Service,
	rng *rand.Rand,
	entriesPerUser int,
) (fixtureCounts, error) {
	cats, err := categorySvc.List(ctx, ownerID)
	if err != nil {
		return fixtureCounts{}, err
	}
	if len(cats) == 0 {
		return fixtureCounts{}, nil
	}
	// Sort before indexing by rng draw: a Store is only contracted to
	// return "every category", not in any particular order — sorting here
	// keeps the fixed-seed RNG reproducible regardless of what order the
	// store happens to return. Account types are a fixed local slice
	// (fixtureAccountTypes), already in a stable order.
	sort.Slice(cats, func(i, j int) bool { return cats[i].Name < cats[j].Name })

	// Created directly from tagNamePool's fixed order, not read back via
	// List — so, unlike types/cats above, there's no store-ordering
	// nondeterminism to sort away.
	tags := make([]tag.Tag, 0, len(tagNamePool))
	for _, name := range tagNamePool {
		t, err := tagSvc.Create(ctx, ownerID, name)
		if err != nil {
			return fixtureCounts{}, err
		}
		tags = append(tags, t)
	}

	numAccounts := 2 + rng.IntN(3) // 2..4
	accountIDs := make([]string, 0, numAccounts)
	created := 0
	for i := 0; i < numAccounts; i++ {
		typ := fixtureAccountTypes[rng.IntN(len(fixtureAccountTypes))]
		currency := accountCurrencies[rng.IntN(len(accountCurrencies))]
		institute := financialInstitutePool[rng.IntN(len(financialInstitutePool))]

		acc, err := accountSvc.Create(ctx, ownerID, account.New{
			Title:              fmt.Sprintf("%s %s", currency, typ),
			Type:               typ,
			Currency:           currency,
			FinancialInstitute: institute,
			OpeningDate:        account.NewDate(randomPastDate(rng, 2*365)),
		})
		if err != nil {
			return fixtureCounts{entries: created}, err
		}
		accountIDs = append(accountIDs, acc.ID)

		numEntries := 15 + rng.IntN(46) // 15..60
		if entriesPerUser > 0 {
			// Spread the requested total across the accounts as evenly as
			// possible; the first (entriesPerUser % numAccounts) accounts
			// take one extra so the sum is exact.
			numEntries = entriesPerUser / numAccounts
			if i < entriesPerUser%numAccounts {
				numEntries++
			}
		}
		for j := 0; j < numEntries; j++ {
			cat := cats[rng.IntN(len(cats))]
			catID := cat.ID

			amount := randomAmount(rng, cat.Name)
			_, err := entrySvc.Create(ctx, ownerID, entry.New{
				AccountID:        acc.ID,
				Kind:             entry.KindTransaction,
				Amount:           &amount,
				BookingTimestamp: randomPastDate(rng, 1000),
				Title:            randomTitle(rng, cat.Name),
				CategoryID:       &catID,
				TagIDs:           randomTagIDs(rng, tags, rng.IntN(3)), // 0..2
			})
			if err != nil {
				return fixtureCounts{entries: created}, err
			}
			created++
		}
	}

	recurringCreated, err := generateRecurringTransactions(ctx, ownerID, accountIDs, cats, recurringSvc, rng)
	if err != nil {
		return fixtureCounts{entries: created}, err
	}

	cardsCreated, err := generateDashboardCards(ctx, ownerID, accountIDs, dashboardSvc)
	if err != nil {
		return fixtureCounts{entries: created, recurring: recurringCreated}, err
	}

	return fixtureCounts{entries: created, recurring: recurringCreated, cards: cardsCreated}, nil
}

// generateRecurringTransactions creates 3-5 distinct recurring-transaction
// templates for ownerID (drawn from recurringFixturePool), each on a
// random one of accountIDs, with its category resolved by name against
// cats (silently skipped if that name isn't present — it always is, since
// every fixture's categoryName is one of category.DefaultNames). Returns
// the number created.
func generateRecurringTransactions(
	ctx context.Context,
	ownerID string,
	accountIDs []string,
	cats []category.Category,
	recurringSvc *recurringtransaction.Service,
	rng *rand.Rand,
) (int, error) {
	if len(accountIDs) == 0 {
		return 0, nil
	}
	catIDByName := make(map[string]string, len(cats))
	for _, c := range cats {
		catIDByName[c.Name] = c.ID
	}

	n := 3 + rng.IntN(3) // 3..5
	if n > len(recurringFixturePool) {
		n = len(recurringFixturePool)
	}
	order := rng.Perm(len(recurringFixturePool))

	created := 0
	for _, i := range order[:n] {
		fx := recurringFixturePool[i]
		catID, ok := catIDByName[fx.categoryName]
		if !ok {
			continue
		}
		accountID := accountIDs[rng.IntN(len(accountIDs))]
		amount := int64(fx.amount * amountScaleFactor)
		startsOn := recurringtransaction.NewDate(randomOffsetDate(fx.startsOnDaysOffset))

		_, err := recurringSvc.Create(ctx, ownerID, recurringtransaction.New{
			AccountID:     accountID,
			Title:         fx.title,
			CategoryID:    &catID,
			Amount:        amount,
			IntervalUnit:  fx.unit,
			IntervalCount: fx.intervalCount,
			StartsOn:      startsOn,
		})
		if err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}

// generateDashboardCards seeds a starter /home layout for ownerID: one
// account_stat card per account in accountIDs (mirroring the app's old
// fixed dashboard), a query_stat card summing every account, and an
// entry_list plus a bar_chart card, both with their recurring-transaction
// preview toggle on — so a freshly seeded dashboard already demonstrates
// that feature rather than needing it turned on by hand. Returns the
// number of cards created.
func generateDashboardCards(
	ctx context.Context,
	ownerID string,
	accountIDs []string,
	dashboardSvc *dashboard.Service,
) (int, error) {
	created := 0
	for _, accountID := range accountIDs {
		cardAccountID := accountID
		if _, err := dashboardSvc.Create(ctx, ownerID, dashboard.New{
			Type:   dashboard.CardTypeAccountStat,
			Config: dashboard.Config{AccountID: &cardAccountID},
		}); err != nil {
			return created, err
		}
		created++
	}

	if _, err := dashboardSvc.Create(ctx, ownerID, dashboard.New{
		Type: dashboard.CardTypeQueryStat,
	}); err != nil {
		return created, err
	}
	created++

	showRecurringPreview := true
	if _, err := dashboardSvc.Create(ctx, ownerID, dashboard.New{
		Type: dashboard.CardTypeEntryList,
		Config: dashboard.Config{
			ShowRecurringPreview: &showRecurringPreview,
		},
	}); err != nil {
		return created, err
	}
	created++

	barChartUnit := string(dashboard.UnitMonth)
	if _, err := dashboardSvc.Create(ctx, ownerID, dashboard.New{
		Type: dashboard.CardTypeBarChart,
		Config: dashboard.Config{
			Unit:                 &barChartUnit,
			ShowRecurringPreview: &showRecurringPreview,
		},
	}); err != nil {
		return created, err
	}
	created++

	return created, nil
}

// randomOffsetDate returns the instant `days` calendar days from now
// (negative allowed, for an intentionally-overdue recurring-transaction
// fixture) — unlike randomPastDate, this is deterministic given days, not
// randomized further, since a fixture's offset is already the source of
// its variety.
func randomOffsetDate(days int) time.Time {
	return time.Now().UTC().AddDate(0, 0, days)
}

// randomTagIDs returns up to n distinct ids drawn from tags, in random
// order. n is clamped to len(tags); n <= 0 or an empty tags returns nil.
func randomTagIDs(rng *rand.Rand, tags []tag.Tag, n int) []string {
	if n <= 0 || len(tags) == 0 {
		return nil
	}
	if n > len(tags) {
		n = len(tags)
	}
	idx := rng.Perm(len(tags))
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = tags[idx[i]].ID
	}
	return out
}

// randomPastDate returns a random time between now and maxDaysAgo days ago.
func randomPastDate(rng *rand.Rand, maxDaysAgo int) time.Time {
	days := rng.IntN(maxDaysAgo)
	hours := rng.IntN(24)
	return time.Now().UTC().AddDate(0, 0, -days).Add(-time.Duration(hours) * time.Hour)
}

// amountScaleFactor is 10^entry.AmountScale — entries store amounts as an
// integer scaled by this factor (see entry.AmountScale's doc comment).
var amountScaleFactor = func() float64 {
	f := 1.0
	for range entry.AmountScale {
		f *= 10
	}
	return f
}()

// randomAmount returns a scaled entry.New.Amount: positive and
// salary-sized for the "Salary" category, negative and expense-sized for
// every other one — random either way, just shaped by which category it
// landed on, the way real transaction history actually looks.
func randomAmount(rng *rand.Rand, categoryName string) int64 {
	if categoryName == "Salary" {
		v := 2000 + rng.Float64()*3000 // 2000.00 .. 5000.00
		return int64(v * amountScaleFactor)
	}
	v := 5 + rng.Float64()*495 // 5.00 .. 500.00
	return -int64(v * amountScaleFactor)
}

// randomTitle picks a plausible title for categoryName from
// categoryTitlePools, falling back to the category's own name.
func randomTitle(rng *rand.Rand, categoryName string) string {
	pool := categoryTitlePools[categoryName]
	if len(pool) == 0 {
		return categoryName
	}
	return pool[rng.IntN(len(pool))]
}
