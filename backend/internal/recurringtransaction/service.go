package recurringtransaction

import (
	"context"
	"errors"
	"sort"
	"time"
)

// Permission mirrors internal/account's four-tier model as plain values so
// AccountLookup can be satisfied structurally by *account.Service without
// this package importing internal/account — the same reason
// internal/entry mirrors its own local Permission type.
type Permission string

const (
	PermissionView       Permission = "view"
	PermissionAppend     Permission = "append"
	PermissionEntryAdmin Permission = "entry_admin"
	PermissionOwner      Permission = "owner"
)

var permissionRank = map[Permission]int{
	PermissionView:       1,
	PermissionAppend:     2,
	PermissionEntryAdmin: 3,
	PermissionOwner:      4,
}

// AtLeast reports whether p is other or a stronger tier. An empty
// Permission (no access at all) is never AtLeast anything.
func (p Permission) AtLeast(other Permission) bool {
	pr, ok := permissionRank[p]
	if !ok {
		return false
	}
	or, ok := permissionRank[other]
	if !ok {
		return false
	}
	return pr >= or
}

// Service is the recurring transaction use-case layer. It depends on Store
// plus the narrow lookup interfaces it declares — the same shape
// internal/entry uses, with EntryLookup added for the two computed fields
// that depend on linked entries.
type Service struct {
	store      Store
	accounts   AccountLookup
	categories CategoryLookup
	tags       TagLookup
	entries    EntryLookup    // nil until SetEntryLookup is called
	timezones  TimezoneLookup // nil when not wired — Ended/Summary then use UTC
}

// Option customizes a Service (used by main.go and tests).
type Option func(*Service)

// WithTimezoneLookup wires the caller-timezone source used to decide
// "today" for Ended/Summary. Optional — without it, every caller is
// resolved in UTC.
func WithTimezoneLookup(l TimezoneLookup) Option { return func(s *Service) { s.timezones = l } }

// NewService builds the recurring transaction service. entries (EntryLookup)
// is wired separately via SetEntryLookup, mirroring account.Service's
// SetUserLookup — package main builds entry.Service and this Service in
// either order, then wires this one's EntryLookup dependency once both
// exist.
func NewService(store Store, accounts AccountLookup, categories CategoryLookup, tags TagLookup, opts ...Option) *Service {
	s := &Service{store: store, accounts: accounts, categories: categories, tags: tags}
	for _, o := range opts {
		o(s)
	}
	return s
}

// SetEntryLookup wires the EntryLookup dependency after construction.
func (s *Service) SetEntryLookup(e EntryLookup) { s.entries = e }

// checkAccount confirms callerID holds at least append permission on
// accountID and that it is not disabled — mirrors internal/entry's own
// checkAccount exactly.
func (s *Service) checkAccount(ctx context.Context, callerID, accountID string) error {
	_, disabled, permission, err := s.accounts.Access(ctx, accountID, callerID)
	if err != nil || !Permission(permission).AtLeast(PermissionAppend) {
		return ErrInvalidValue
	}
	if disabled {
		return ErrAccountDisabled
	}
	return nil
}

// authorizeWrite fetches id and resolves callerID's write access to it,
// mirroring internal/entry's authorizeEntryWrite: ErrNotFound when callerID
// has no permission at all on its parent account; ErrForbidden when
// callerID has some permission but not enough — below append, or exactly
// append on a recurring transaction a different user created.
func (s *Service) authorizeWrite(ctx context.Context, callerID, id string) (RecurringTransaction, error) {
	current, err := s.store.Get(ctx, id)
	if err != nil {
		return RecurringTransaction{}, err
	}
	_, _, permStr, err := s.accounts.Access(ctx, current.AccountID, callerID)
	if err != nil {
		return RecurringTransaction{}, err
	}
	permission := Permission(permStr)
	if !permission.AtLeast(PermissionView) {
		return RecurringTransaction{}, ErrNotFound
	}
	if !permission.AtLeast(PermissionAppend) {
		return RecurringTransaction{}, ErrForbidden
	}
	if permission == PermissionAppend && current.CreatedBy != callerID {
		return RecurringTransaction{}, ErrForbidden
	}
	return current, nil
}

// today resolves callerID's current calendar date in their resolved
// timezone (default UTC when no TimezoneLookup is wired, the lookup fails,
// or it reports empty).
func (s *Service) today(ctx context.Context, callerID string) Date {
	loc := time.UTC
	if s.timezones != nil {
		if tz, err := s.timezones.Timezone(ctx, callerID); err == nil && tz != "" {
			if l, err := time.LoadLocation(tz); err == nil {
				loc = l
			}
		}
	}
	return NewDate(time.Now().In(loc))
}

// decorate fills in a recurring transaction's computed, never-stored
// fields: PerYearAmount and Ended (pure), and NextSuggestedDate and
// LinkedEntryCount (both need EntryLookup). LinkedEntryCount lets the
// client disable its delete action up front, the same
// know-before-you-try precedent category.Category's entry_count already
// establishes for categories' own in-use delete guard.
func (s *Service) decorate(ctx context.Context, callerID string, rt RecurringTransaction) (RecurringTransaction, error) {
	rt.PerYearAmount = PerYearAmount(rt.Amount, rt.IntervalUnit, rt.IntervalCount)
	rt.Ended = Ended(rt.EndsOn, s.today(ctx, callerID))

	if s.entries != nil {
		latest, err := s.entries.LatestLinkedBookingTime(ctx, rt.ID)
		if err != nil {
			return RecurringTransaction{}, err
		}
		if latest == nil {
			rt.NextSuggestedDate = rt.StartsOn
		} else {
			rt.NextSuggestedDate = NewDate(Advance(*latest, rt.IntervalUnit, rt.IntervalCount))
		}
		count, err := s.entries.LinkedCount(ctx, rt.ID)
		if err != nil {
			return RecurringTransaction{}, err
		}
		rt.LinkedEntryCount = count
	} else {
		rt.NextSuggestedDate = rt.StartsOn
	}
	return rt, nil
}

// Create validates in, confirms callerID holds at least append permission
// on the target account and that it is not disabled, confirms the category/
// tags are usable by callerID, and creates the recurring transaction with
// CreatedBy set to callerID.
func (s *Service) Create(ctx context.Context, callerID string, in New) (RecurringTransaction, error) {
	if err := validateNew(in); err != nil {
		return RecurringTransaction{}, err
	}
	if err := s.checkAccount(ctx, callerID, in.AccountID); err != nil {
		return RecurringTransaction{}, err
	}
	if ok, err := s.categories.Usable(ctx, callerID, *in.CategoryID); err != nil {
		return RecurringTransaction{}, err
	} else if !ok {
		return RecurringTransaction{}, ErrInvalidValue
	}
	if len(in.TagIDs) > 0 {
		ok, err := s.tags.Usable(ctx, callerID, in.TagIDs)
		if err != nil {
			return RecurringTransaction{}, err
		}
		if !ok {
			return RecurringTransaction{}, ErrInvalidValue
		}
	}

	rt, err := s.store.Create(ctx, callerID, in)
	if err != nil {
		return RecurringTransaction{}, err
	}
	return s.decorate(ctx, callerID, rt)
}

// Get returns id as seen by callerID: ErrNotFound when they hold no
// permission on its parent account at all.
func (s *Service) Get(ctx context.Context, callerID, id string) (RecurringTransaction, error) {
	rt, err := s.store.Get(ctx, id)
	if err != nil {
		return RecurringTransaction{}, err
	}
	_, _, permStr, err := s.accounts.Access(ctx, rt.AccountID, callerID)
	if err != nil {
		return RecurringTransaction{}, err
	}
	if !Permission(permStr).AtLeast(PermissionView) {
		return RecurringTransaction{}, ErrNotFound
	}
	return s.decorate(ctx, callerID, rt)
}

// Update validates and applies a partial change to id, authorized for
// callerID per authorizeWrite. A non-nil AccountID moves the recurring
// transaction, subject to the same append+/disabled checks Create applies.
func (s *Service) Update(ctx context.Context, callerID, id string, upd Update) (RecurringTransaction, error) {
	current, err := s.authorizeWrite(ctx, callerID, id)
	if err != nil {
		return RecurringTransaction{}, err
	}
	if err := validateUpdate(current, upd); err != nil {
		return RecurringTransaction{}, err
	}

	if upd.AccountID != nil {
		if err := s.checkAccount(ctx, callerID, *upd.AccountID); err != nil {
			return RecurringTransaction{}, err
		}
	}
	if upd.CategoryID != nil {
		ok, err := s.categories.Usable(ctx, callerID, *upd.CategoryID)
		if err != nil {
			return RecurringTransaction{}, err
		}
		if !ok {
			return RecurringTransaction{}, ErrInvalidValue
		}
	}
	if upd.TagIDs != nil && len(*upd.TagIDs) > 0 {
		ok, err := s.tags.OwnedBy(ctx, callerID, *upd.TagIDs)
		if err != nil {
			return RecurringTransaction{}, err
		}
		if !ok {
			return RecurringTransaction{}, ErrInvalidValue
		}
		newTagIDs := diffStrings(*upd.TagIDs, current.TagIDs)
		if len(newTagIDs) > 0 {
			ok, err := s.tags.Usable(ctx, callerID, newTagIDs)
			if err != nil {
				return RecurringTransaction{}, err
			}
			if !ok {
				return RecurringTransaction{}, ErrInvalidValue
			}
		}
	}

	rt, err := s.store.Update(ctx, id, upd)
	if err != nil {
		return RecurringTransaction{}, err
	}
	return s.decorate(ctx, callerID, rt)
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

// SameAccount satisfies internal/entry's RecurringTransactionLookup: it
// reports whether id names a non-deleted recurring transaction whose
// AccountID equals accountID. Deliberately unauthenticated (no permission
// check) — the caller linking an entry to it already holds append+ on
// accountID itself, which is what actually gates the write; this only
// confirms the two records agree on which account they belong to.
func (s *Service) SameAccount(ctx context.Context, id, accountID string) (bool, error) {
	current, err := s.store.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return current.AccountID == accountID, nil
}

// Delete soft-deletes id, authorized for callerID per authorizeWrite, and
// rejected (ErrInUse) while any non-deleted entry still links to it.
func (s *Service) Delete(ctx context.Context, callerID, id string) error {
	if _, err := s.authorizeWrite(ctx, callerID, id); err != nil {
		return err
	}
	if s.entries != nil {
		count, err := s.entries.LinkedCount(ctx, id)
		if err != nil {
			return err
		}
		if count > 0 {
			return ErrInUse
		}
	}
	return s.store.SoftDelete(ctx, id)
}

// resolveAccountIDs narrows f.AccountIDs to callerID's own visible accounts
// (owned or shared), intersected with any caller-supplied AccountIDs —
// mirrors internal/entry's own account-visibility resolution.
func (s *Service) resolveAccountIDs(ctx context.Context, callerID string, f Filter) (Filter, error) {
	visible, err := s.accounts.VisibleIDs(ctx, callerID)
	if err != nil {
		return Filter{}, err
	}
	if len(f.AccountIDs) > 0 {
		f.AccountIDs = intersect(f.AccountIDs, visible)
	} else {
		f.AccountIDs = visible
	}
	return f, nil
}

// List resolves f's caller-supplied AccountIDs against callerID's visible
// accounts and returns every matching recurring transaction, decorated.
func (s *Service) List(ctx context.Context, callerID string, f Filter) ([]RecurringTransaction, error) {
	f, err := s.resolveAccountIDs(ctx, callerID, f)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.List(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]RecurringTransaction, len(rows))
	for i, rt := range rows {
		decorated, err := s.decorate(ctx, callerID, rt)
		if err != nil {
			return nil, err
		}
		out[i] = decorated
	}
	return out, nil
}

// Summary resolves f the same way List does, then sums each matching,
// non-ended recurring transaction's PerYearAmount grouped by its account's
// currency — mirrors internal/entry's Sum, with the ended-exclusion added.
func (s *Service) Summary(ctx context.Context, callerID string, f Filter) (Summary, error) {
	f, err := s.resolveAccountIDs(ctx, callerID, f)
	if err != nil {
		return Summary{}, err
	}
	rows, err := s.store.List(ctx, f)
	if err != nil {
		return Summary{}, err
	}

	today := s.today(ctx, callerID)
	totals := map[string]int64{}
	count := 0
	for _, rt := range rows {
		if Ended(rt.EndsOn, today) {
			continue
		}
		currency, _, _, err := s.accounts.Access(ctx, rt.AccountID, callerID)
		if err != nil {
			return Summary{}, err
		}
		totals[currency] += PerYearAmount(rt.Amount, rt.IntervalUnit, rt.IntervalCount)
		count++
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

// CurrencySum is one currency's total within a Summary — same shape as
// internal/entry's own CurrencySum (kept local to avoid a cross-package
// import for a two-field struct).
type CurrencySum struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
}

// Summary is the result of summing a Filter's matching, non-ended recurring
// transactions' PerYearAmount, grouped by account currency.
type Summary struct {
	Sums  []CurrencySum `json:"sums"`
	Count int           `json:"count"`
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
