package entry

import (
	"context"
	"sort"
	"strings"
	"time"
)

// Service is the entry use-case layer. It depends on Store plus the three
// narrow lookup interfaces it declares.
type Service struct {
	store      Store
	accounts   AccountLookup
	categories CategoryLookup
	tags       TagLookup
	timezones  TimezoneLookup // nil when not wired — FlowSummary then buckets in UTC
}

// Option customizes a Service (used by main.go and tests).
type Option func(*Service)

// WithTimezoneLookup wires the caller-timezone source FlowSummary uses to
// bucket entries. package main passes internal/settings' Service. Optional —
// without it, FlowSummary buckets every caller in UTC.
func WithTimezoneLookup(l TimezoneLookup) Option { return func(s *Service) { s.timezones = l } }

// NewService builds the entry service.
func NewService(store Store, accounts AccountLookup, categories CategoryLookup, tags TagLookup, opts ...Option) *Service {
	s := &Service{store: store, accounts: accounts, categories: categories, tags: tags}
	for _, o := range opts {
		o(s)
	}
	return s
}

// checkAccount confirms ownerID owns accountID and that it is not disabled —
// the rule Create applies to the account an entry is created against, and
// Update applies identically to a new account_id an entry is being moved to.
func (s *Service) checkAccount(ctx context.Context, ownerID, accountID string) error {
	accOwner, _, disabled, err := s.accounts.Owner(ctx, accountID)
	if err != nil || accOwner != ownerID {
		return ErrInvalidValue
	}
	if disabled {
		return ErrAccountDisabled
	}
	return nil
}

// Create validates in, confirms ownerID owns the target account and that it
// is not disabled, confirms any category/tags are usable, and creates the
// entry.
func (s *Service) Create(ctx context.Context, ownerID string, in New) (Entry, error) {
	if err := validateNew(in); err != nil {
		return Entry{}, err
	}

	if err := s.checkAccount(ctx, ownerID, in.AccountID); err != nil {
		return Entry{}, err
	}

	if in.CategoryID != nil {
		ok, err := s.categories.Usable(ctx, ownerID, *in.CategoryID)
		if err != nil {
			return Entry{}, err
		}
		if !ok {
			return Entry{}, ErrInvalidValue
		}
	}
	if len(in.TagIDs) > 0 {
		// Every tag id is new at creation, so Usable (owned + not disabled)
		// applies to the whole set.
		ok, err := s.tags.Usable(ctx, ownerID, in.TagIDs)
		if err != nil {
			return Entry{}, err
		}
		if !ok {
			return Entry{}, ErrInvalidValue
		}
	}

	return s.store.Create(ctx, ownerID, in)
}

// Get returns ownerID's entry with this id.
func (s *Service) Get(ctx context.Context, ownerID, id string) (Entry, error) {
	return s.store.Get(ctx, ownerID, id)
}

// Update validates and applies a partial change to ownerID's entry.
// AccountID and Kind cannot be changed — there is no field for them on
// Update at all (see its doc comment).
func (s *Service) Update(ctx context.Context, ownerID, id string, upd Update) (Entry, error) {
	current, err := s.store.Get(ctx, ownerID, id)
	if err != nil {
		return Entry{}, err
	}

	if upd.AccountID != nil {
		if err := s.checkAccount(ctx, ownerID, *upd.AccountID); err != nil {
			return Entry{}, err
		}
	}
	if upd.Title != nil && strings.TrimSpace(*upd.Title) == "" {
		return Entry{}, ErrInvalidValue
	}
	if upd.BookingTimestamp != nil && upd.BookingTimestamp.IsZero() {
		return Entry{}, ErrInvalidValue
	}
	// Amount is only settable on a transaction, Balance only on a balance
	// adjustment — kind is immutable, so this checks against the entry's
	// existing kind rather than anything in the request body.
	if current.Kind == KindTransaction && upd.Balance != nil {
		return Entry{}, ErrInvalidValue
	}
	if current.Kind == KindBalanceAdjustment && upd.Amount != nil {
		return Entry{}, ErrInvalidValue
	}

	newCategoryID := current.CategoryID
	if upd.CategoryID.Set {
		newCategoryID = upd.CategoryID.Value
	}
	if current.Kind == KindTransaction && newCategoryID == nil {
		return Entry{}, ErrInvalidValue
	}
	// Only a category explicitly supplied in this request is validated —
	// an update that doesn't touch category_id never re-checks the
	// entry's existing (possibly since-disabled) category.
	if upd.CategoryID.Set && upd.CategoryID.Value != nil {
		ok, err := s.categories.Usable(ctx, ownerID, *upd.CategoryID.Value)
		if err != nil {
			return Entry{}, err
		}
		if !ok {
			return Entry{}, ErrInvalidValue
		}
	}

	if upd.TagIDs != nil && len(*upd.TagIDs) > 0 {
		ok, err := s.tags.OwnedBy(ctx, ownerID, *upd.TagIDs)
		if err != nil {
			return Entry{}, err
		}
		if !ok {
			return Entry{}, ErrInvalidValue
		}
		// tag_ids is always a full replacement array, so a resubmitted id
		// the entry already carried is not "newly assigned" — only ids the
		// caller is actually adding need to pass the not-disabled check.
		newTagIDs := diffStrings(*upd.TagIDs, current.TagIDs)
		if len(newTagIDs) > 0 {
			ok, err := s.tags.Usable(ctx, ownerID, newTagIDs)
			if err != nil {
				return Entry{}, err
			}
			if !ok {
				return Entry{}, ErrInvalidValue
			}
		}
	}

	return s.store.Update(ctx, ownerID, id, upd)
}

// diffStrings returns the elements of next not present in current.
func diffStrings(next, current []string) []string {
	existing := make(map[string]bool, len(current))
	for _, id := range current {
		existing[id] = true
	}
	var out []string
	for _, id := range next {
		if !existing[id] {
			out = append(out, id)
		}
	}
	return out
}

// Delete soft-deletes ownerID's entry.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	return s.store.SoftDelete(ctx, ownerID, id)
}

// resolveFilter applies the account-visibility and category resolution
// shared by List and Sum: narrows f.AccountIDs to ownerID's own visible
// accounts (intersected with any caller-supplied AccountIDs), and — when
// f.CategoryID is set — resolves f.CategoryIDs to either the category's
// full subtree (the default) or the category alone (CategoryMode:
// ModeExact).
func (s *Service) resolveFilter(ctx context.Context, ownerID string, f Filter) (Filter, error) {
	if !f.CategoryMode.valid() {
		return Filter{}, ErrInvalidValue
	}

	visible, err := s.accounts.VisibleIDs(ctx, ownerID)
	if err != nil {
		return Filter{}, err
	}
	if len(f.AccountIDs) > 0 {
		f.AccountIDs = intersect(f.AccountIDs, visible)
	} else {
		f.AccountIDs = visible
	}

	if f.CategoryID != nil {
		if f.CategoryMode == ModeExact {
			f.CategoryIDs = []string{*f.CategoryID}
		} else {
			ids, err := s.categories.Subtree(ctx, ownerID, *f.CategoryID)
			if err != nil {
				return Filter{}, err
			}
			f.CategoryIDs = ids
		}
	}

	return f, nil
}

// List resolves f's caller-supplied AccountIDs/CategoryID against the
// caller's visible accounts and the category tree, applies defaults for
// Sort/Dir/Limit, and returns a page of ownerID's matching entries.
func (s *Service) List(ctx context.Context, ownerID string, f Filter) ([]Entry, *Cursor, error) {
	if f.Sort == "" {
		f.Sort = SortBookingTimestamp
	} else if !f.Sort.valid() {
		return nil, nil, ErrInvalidValue
	}
	if f.Dir == "" {
		f.Dir = DirDesc
	} else if !f.Dir.valid() {
		return nil, nil, ErrInvalidValue
	}
	if f.Limit <= 0 || f.Limit > maxPageSize {
		f.Limit = defaultPageSize
	}
	if f.Kind != nil && !f.Kind.valid() {
		return nil, nil, ErrInvalidValue
	}

	f, err := s.resolveFilter(ctx, ownerID, f)
	if err != nil {
		return nil, nil, err
	}

	return s.store.List(ctx, ownerID, f)
}

// Sum resolves f the same way List does (account visibility, category
// exact/subtree resolution), then returns the matching kind: transaction
// entries' amounts summed per account currency — Store.Sum forces the
// transaction-only restriction regardless of f.Kind. Currencies are
// resolved per matching account via AccountLookup.Owner, since Store has
// no notion of an account's currency.
func (s *Service) Sum(ctx context.Context, ownerID string, f Filter) (Summary, error) {
	f, err := s.resolveFilter(ctx, ownerID, f)
	if err != nil {
		return Summary{}, err
	}

	perAccount, count, err := s.store.Sum(ctx, ownerID, f)
	if err != nil {
		return Summary{}, err
	}

	totals := make(map[string]int64, len(perAccount))
	for accountID, amount := range perAccount {
		_, currency, _, err := s.accounts.Owner(ctx, accountID)
		if err != nil {
			return Summary{}, err
		}
		totals[currency] += amount
	}

	currencies := make([]string, 0, len(totals))
	for currency := range totals {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)

	sums := make([]CurrencySum, 0, len(currencies))
	for _, currency := range currencies {
		sums = append(sums, CurrencySum{Currency: currency, Amount: totals[currency]})
	}

	return Summary{Sums: sums, Count: count}, nil
}

// FlowSummary resolves f's caller-supplied AccountIDs the same way List/Sum
// do, resolves the caller's timezone (default UTC when no TimezoneLookup is
// wired, the lookup fails, or it reports empty), buckets ownerID's matching
// entries via Store.FlowSummary, and fills in every period in the
// requested year (or month, for FlowUnitDay) that had no matching entries
// at all — a bucket is never omitted for being empty. Each row's account is
// resolved to its currency the same way Sum resolves its per-account totals.
func (s *Service) FlowSummary(ctx context.Context, ownerID string, f FlowFilter) ([]FlowBucket, error) {
	if !f.Unit.valid() {
		return nil, ErrInvalidValue
	}
	if f.Unit == FlowUnitDay {
		if f.Month < 1 || f.Month > 12 {
			return nil, ErrInvalidValue
		}
	} else if f.Month != 0 {
		return nil, ErrInvalidValue
	}
	if f.Year < 1 {
		return nil, ErrInvalidValue
	}

	visible, err := s.accounts.VisibleIDs(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if len(f.AccountIDs) > 0 {
		f.AccountIDs = intersect(f.AccountIDs, visible)
	} else {
		f.AccountIDs = visible
	}

	f.Timezone = "UTC"
	if s.timezones != nil {
		if tz, err := s.timezones.Timezone(ctx, ownerID); err == nil && tz != "" {
			f.Timezone = tz
		}
	}

	rows, err := s.store.FlowSummary(ctx, ownerID, f)
	if err != nil {
		return nil, err
	}

	type totals struct{ income, outcome int64 }
	byPeriod := map[string]map[string]*totals{} // period -> currency -> totals
	currencyByAccount := map[string]string{}
	for _, row := range rows {
		currency, ok := currencyByAccount[row.AccountID]
		if !ok {
			_, cur, _, err := s.accounts.Owner(ctx, row.AccountID)
			if err != nil {
				return nil, err
			}
			currency = cur
			currencyByAccount[row.AccountID] = currency
		}
		byCurrency, ok := byPeriod[row.Period]
		if !ok {
			byCurrency = map[string]*totals{}
			byPeriod[row.Period] = byCurrency
		}
		t, ok := byCurrency[currency]
		if !ok {
			t = &totals{}
			byCurrency[currency] = t
		}
		t.income += row.Income
		t.outcome += row.Outcome
	}

	buckets := make([]FlowBucket, 0, len(flowPeriods(f)))
	for _, period := range flowPeriods(f) {
		b := FlowBucket{Period: period, Income: []CurrencySum{}, Outcome: []CurrencySum{}}
		if byCurrency, ok := byPeriod[period]; ok {
			currencies := make([]string, 0, len(byCurrency))
			for c := range byCurrency {
				currencies = append(currencies, c)
			}
			sort.Strings(currencies)
			for _, c := range currencies {
				t := byCurrency[c]
				// A currency present only as income (or only as outcome)
				// in this period is not also listed with a 0 on the other
				// side — matches Sum's "one entry per currency actually
				// present," now split per direction.
				if t.income != 0 {
					b.Income = append(b.Income, CurrencySum{Currency: c, Amount: t.income})
				}
				if t.outcome != 0 {
					b.Outcome = append(b.Outcome, CurrencySum{Currency: c, Amount: t.outcome})
				}
			}
		}
		buckets = append(buckets, b)
	}
	return buckets, nil
}

// flowPeriods returns every period label (the bucket's first calendar day,
// "YYYY-MM-DD") a FlowFilter's year (and, for FlowUnitDay, month) spans, in
// order — the skeleton FlowSummary fills gaps into so no period is ever
// omitted for being empty.
func flowPeriods(f FlowFilter) []string {
	if f.Unit == FlowUnitDay {
		days := time.Date(f.Year, time.Month(f.Month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
		periods := make([]string, days)
		for d := 1; d <= days; d++ {
			periods[d-1] = time.Date(f.Year, time.Month(f.Month), d, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		}
		return periods
	}
	periods := make([]string, 12)
	for m := 1; m <= 12; m++ {
		periods[m-1] = time.Date(f.Year, time.Month(m), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}
	return periods
}

// Balance confirms ownerID owns accountID, then returns its live balance as
// of asOf.
func (s *Service) Balance(ctx context.Context, ownerID, accountID string, asOf time.Time) (int64, error) {
	owner, _, _, err := s.accounts.Owner(ctx, accountID)
	if err != nil {
		return 0, err
	}
	if owner != ownerID {
		return 0, ErrNotFound
	}
	return s.store.Balance(ctx, accountID, asOf)
}

// BalanceSeries returns ownerID's running account balance sampled at every
// local midnight of f.Year/f.Month — one point at 00:00 on each calendar
// day of the month, plus a closing point at 00:00 on the first day of the
// following month — in the caller's resolved timezone (default UTC). Each
// point's value per currency is the sum, over the matching accounts of
// that currency, of Store.Balance strictly before the sample instant (so
// an entry booked exactly at a midnight belongs to that day, not the point
// that opens it) — the same anchor-aware computation
// GET /api/accounts/{id}/balance uses. Every currency present in the
// resolved accounts appears on every point, including with a 0 amount.
func (s *Service) BalanceSeries(ctx context.Context, ownerID string, f BalanceFilter) ([]BalancePoint, error) {
	if f.Unit != FlowUnitDay {
		return nil, ErrInvalidValue
	}
	if f.Month < 1 || f.Month > 12 {
		return nil, ErrInvalidValue
	}
	if f.Year < 1 {
		return nil, ErrInvalidValue
	}

	visible, err := s.accounts.VisibleIDs(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if len(f.AccountIDs) > 0 {
		f.AccountIDs = intersect(f.AccountIDs, visible)
	} else {
		f.AccountIDs = visible
	}

	f.Timezone = "UTC"
	if s.timezones != nil {
		if tz, err := s.timezones.Timezone(ctx, ownerID); err == nil && tz != "" {
			f.Timezone = tz
		}
	}
	loc, err := time.LoadLocation(f.Timezone)
	if err != nil {
		loc = time.UTC
	}

	// Resolve every matching account's currency up front, so a currency is
	// listed on every point even for a day it happens to sum to 0.
	currencyByAccount := make(map[string]string, len(f.AccountIDs))
	currencySet := map[string]struct{}{}
	for _, id := range f.AccountIDs {
		_, cur, _, err := s.accounts.Owner(ctx, id)
		if err != nil {
			return nil, err
		}
		currencyByAccount[id] = cur
		currencySet[cur] = struct{}{}
	}
	currencies := make([]string, 0, len(currencySet))
	for c := range currencySet {
		currencies = append(currencies, c)
	}
	sort.Strings(currencies)

	// 00:00 local on days 1..N of the month, then 00:00 local on the 1st of
	// the next month — the closing point that captures the last day's own
	// activity.
	days := time.Date(f.Year, time.Month(f.Month)+1, 0, 0, 0, 0, 0, loc).Day()
	boundaries := make([]time.Time, 0, days+1)
	for d := 1; d <= days; d++ {
		boundaries = append(boundaries, time.Date(f.Year, time.Month(f.Month), d, 0, 0, 0, 0, loc))
	}
	boundaries = append(boundaries, time.Date(f.Year, time.Month(f.Month)+1, 1, 0, 0, 0, 0, loc))

	points := make([]BalancePoint, 0, len(boundaries))
	for _, boundary := range boundaries {
		asOf := boundary.Add(-time.Nanosecond)
		byCurrency := make(map[string]int64, len(currencies))
		for _, c := range currencies {
			byCurrency[c] = 0
		}
		for _, id := range f.AccountIDs {
			bal, err := s.store.Balance(ctx, id, asOf)
			if err != nil {
				return nil, err
			}
			byCurrency[currencyByAccount[id]] += bal
		}
		p := BalancePoint{
			Period:   boundary.Format("2006-01-02"),
			Balances: make([]CurrencySum, 0, len(currencies)),
		}
		for _, c := range currencies {
			p.Balances = append(p.Balances, CurrencySum{Currency: c, Amount: byCurrency[c]})
		}
		points = append(points, p)
	}
	return points, nil
}

// intersect returns the elements of a that also appear in b.
func intersect(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, id := range b {
		set[id] = true
	}
	var out []string
	for _, id := range a {
		if set[id] {
			out = append(out, id)
		}
	}
	return out
}
