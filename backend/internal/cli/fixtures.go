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
	"at.draab/familyfinances/internal/entry"
)

// accountCurrencies is the small pool of currencies a generated account
// randomly draws from.
var accountCurrencies = []string{"EUR", "USD", "GBP"}

// categoryTitlePools gives each seeded default category (category.DefaultNames)
// a few plausible entry titles, so a generated listing doesn't read as
// obviously synthetic. A category outside this set (there shouldn't be one,
// since fixtures only ever draw from a fresh user's seeded categories) falls
// back to its own name.
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

// seedTesters creates the fixed set of testers (seedEmails) directly via
// authStore.CreateUserWithIdentity — a pre-verified email identity each,
// the same mechanism this package's own tests already use, bypassing the
// magic-link/OIDC flows entirely. Because that goes straight to the store,
// internal/auth's NewUserHooks (wired only inside auth.Service) never fire
// here; internal/cli already imports account/category directly (the same
// way main.go's builders do), so it calls each user's SeedDefaults itself
// instead of routing through that indirection — NewUserHook exists to let
// internal/auth notify account/category without importing them, a problem
// internal/cli doesn't have. authSvc is used only to mint each tester's
// session token via IssueSession. Prints (email, admin?, session token) per
// tester to stdout as it goes; returns 0 on success, or 1 with a stderr
// message on the first failure.
func seedTesters(
	ctx context.Context,
	authStore auth.Store,
	authSvc *auth.Service,
	accountSvc *account.Service,
	categorySvc *category.Service,
	entrySvc *entry.Service,
	rng *rand.Rand,
	stdout, stderr io.Writer,
) int {
	for _, email := range seedEmails {
		user, _, err := authStore.CreateUserWithIdentity(ctx, auth.NewUser{Email: email}, auth.Identity{
			Kind:          auth.IdentityEmail,
			Email:         email,
			EmailVerified: true,
		})
		if err != nil {
			fmt.Fprintf(stderr, "seed: create %s: %v\n", email, err)
			return 1
		}

		if err := accountSvc.SeedDefaults(ctx, user.ID); err != nil {
			fmt.Fprintf(stderr, "seed: seed account types for %s: %v\n", email, err)
			return 1
		}
		if err := categorySvc.SeedDefaults(ctx, user.ID); err != nil {
			fmt.Fprintf(stderr, "seed: seed categories for %s: %v\n", email, err)
			return 1
		}
		if err := generateFixtures(ctx, user.ID, accountSvc, categorySvc, entrySvc, rng); err != nil {
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
		fmt.Fprintf(stdout, "%s%s session=%s\n", email, admin, token)
	}
	return 0
}

// generateFixtures creates 2-4 random accounts for ownerID (each a random
// seeded type + a random currency + a random opening date within the past
// two years), then 15-60 random transaction entries per account (random
// seeded category, booking date within the past year, and an
// amount/title shaped by that category — see randomAmount/randomTitle).
func generateFixtures(
	ctx context.Context,
	ownerID string,
	accountSvc *account.Service,
	categorySvc *category.Service,
	entrySvc *entry.Service,
	rng *rand.Rand,
) error {
	types, err := accountSvc.ListTypes(ctx, ownerID)
	if err != nil {
		return err
	}
	cats, err := categorySvc.List(ctx, ownerID)
	if err != nil {
		return err
	}
	if len(types) == 0 || len(cats) == 0 {
		return nil
	}
	// Sort before indexing by rng draw: a Store is only contracted to
	// return "every type/category", not in any particular order (memory's
	// ListTypes, for one, ties on CreatedAt for a freshly seeded set and
	// doesn't sort further) — sorting here keeps the fixed-seed RNG
	// reproducible regardless of what order the store happens to return.
	sort.Slice(types, func(i, j int) bool { return types[i].Title < types[j].Title })
	sort.Slice(cats, func(i, j int) bool { return cats[i].Name < cats[j].Name })

	numAccounts := 2 + rng.IntN(3) // 2..4
	for i := 0; i < numAccounts; i++ {
		typ := types[rng.IntN(len(types))]
		currency := accountCurrencies[rng.IntN(len(accountCurrencies))]

		acc, err := accountSvc.Create(ctx, ownerID, account.New{
			Title:       fmt.Sprintf("%s %s", currency, typ.Title),
			TypeID:      typ.ID,
			Currency:    currency,
			OpeningDate: account.NewDate(randomPastDate(rng, 2*365)),
		})
		if err != nil {
			return err
		}

		numEntries := 15 + rng.IntN(46) // 15..60
		for j := 0; j < numEntries; j++ {
			cat := cats[rng.IntN(len(cats))]
			catID := cat.ID

			_, err := entrySvc.Create(ctx, ownerID, entry.New{
				AccountID:        acc.ID,
				Kind:             entry.KindTransaction,
				Amount:           randomAmount(rng, cat.Name),
				BookingTimestamp: randomPastDate(rng, 365),
				Title:            randomTitle(rng, cat.Name),
				CategoryID:       &catID,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
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
