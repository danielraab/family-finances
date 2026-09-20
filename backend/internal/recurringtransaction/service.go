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

// checkSelfTransferAccounts runs checkAccount against both of a
// self_transfer template's accounts and confirms they share a currency —
// the same rule internal/entry already applies when creating a
// self_transfer entry, applied here at template time so a template can
// never describe a movement its author could not book (see design.md).
func (s *Service) checkSelfTransferAccounts(ctx context.Context, callerID, accountID, toAccountID string) error {
	if err := s.checkAccount(ctx, callerID, accountID); err != nil {
		return err
	}
	if err := s.checkAccount(ctx, callerID, toAccountID); err != nil {
		return err
	}
	from, _, _, err := s.accounts.Access(ctx, accountID, callerID)
	if err != nil {
		return err
	}
	to, _, _, err := s.accounts.Access(ctx, toAccountID, callerID)
	if err != nil {
		return err
	}
	if from != to {
		return ErrInvalidValue
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
	// Editing a self_transfer template additionally requires append+ on its
	// far account, still held now — mirroring account-entries' identical
	// rule for a self_transfer entry, and for the same reason: the amount
	// this template carries governs a movement into an account the editor
	// must still be trusted with. Delete does not go through here.
	if current.Kind == KindSelfTransfer && current.ToAccountID != nil {
		_, _, farStr, err := s.accounts.Access(ctx, *current.ToAccountID, callerID)
		if err != nil {
			return RecurringTransaction{}, err
		}
		if !Permission(farStr).AtLeast(PermissionAppend) {
			return RecurringTransaction{}, ErrForbidden
		}
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

	nextSuggested, err := s.nextSuggestedDate(ctx, rt)
	if err != nil {
		return RecurringTransaction{}, err
	}
	rt.NextSuggestedDate = nextSuggested

	if s.entries != nil {
		count, err := s.entries.LinkedCount(ctx, rt.ID)
		if err != nil {
			return RecurringTransaction{}, err
		}
		rt.LinkedEntryCount = count
	}
	return rt, nil
}

// nextSuggestedDate computes rt's NextSuggestedDate — factored out of
// decorate so Preview can anchor its occurrence generation at the same
// value without duplicating the "latest linked entry, or starts_on" logic.
func (s *Service) nextSuggestedDate(ctx context.Context, rt RecurringTransaction) (Date, error) {
	if s.entries == nil {
		return rt.StartsOn, nil
	}
	latest, err := s.entries.LatestLinkedBookingTime(ctx, rt.ID)
	if err != nil {
		return Date{}, err
	}
	if latest == nil {
		return rt.StartsOn, nil
	}
	return NewDate(Advance(*latest, rt.IntervalUnit, rt.IntervalCount)), nil
}

// Create validates in, confirms callerID holds at least append permission
// on the target account and that it is not disabled — on both accounts,
// sharing a currency, for a self_transfer — confirms the category/tags are
// usable by callerID, and creates the recurring transaction with CreatedBy
// set to callerID.
func (s *Service) Create(ctx context.Context, callerID string, in New) (RecurringTransaction, error) {
	if err := validateNew(in); err != nil {
		return RecurringTransaction{}, err
	}
	if in.Kind == KindSelfTransfer {
		if err := s.checkSelfTransferAccounts(ctx, callerID, in.AccountID, *in.ToAccountID); err != nil {
			return RecurringTransaction{}, err
		}
	} else if err := s.checkAccount(ctx, callerID, in.AccountID); err != nil {
		return RecurringTransaction{}, err
	}
	// A category is required for a transaction and optional for a
	// self_transfer (validateNew enforces which); when one is supplied it is
	// checked the same way for both kinds.
	if in.CategoryID != nil {
		if ok, err := s.categories.Usable(ctx, callerID, *in.CategoryID); err != nil {
			return RecurringTransaction{}, err
		} else if !ok {
			return RecurringTransaction{}, ErrInvalidValue
		}
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
	visible, err := s.readable(ctx, callerID, rt)
	if err != nil {
		return RecurringTransaction{}, err
	}
	if !visible {
		return RecurringTransaction{}, ErrNotFound
	}
	return s.decorate(ctx, callerID, rt)
}

// readable reports whether callerID may read rt: view+ on its account, or —
// for a self_transfer, whose two accounts are both parents — on either of
// them, mirroring account-entries' "either parent account is sufficient"
// read rule for a self_transfer entry.
func (s *Service) readable(ctx context.Context, callerID string, rt RecurringTransaction) (bool, error) {
	_, _, permStr, err := s.accounts.Access(ctx, rt.AccountID, callerID)
	if err != nil {
		return false, err
	}
	if Permission(permStr).AtLeast(PermissionView) {
		return true, nil
	}
	if rt.Kind != KindSelfTransfer || rt.ToAccountID == nil {
		return false, nil
	}
	_, _, farStr, err := s.accounts.Access(ctx, *rt.ToAccountID, callerID)
	if err != nil {
		return false, err
	}
	return Permission(farStr).AtLeast(PermissionView), nil
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
		// A self_transfer template's accounts are fixed at creation, the
		// same way its entry counterpart's are — moving one side alone
		// would silently repoint a pair that was validated together.
		if current.Kind == KindSelfTransfer {
			return RecurringTransaction{}, ErrInvalidValue
		}
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

// authorizeDelete is authorizeWrite's rule evaluated against AccountID
// alone — deleting a self_transfer template removes a suggestion and moves
// no money, so unlike editing it needs nothing on the far account. Mirrors
// the same edit-needs-both/delete-needs-one asymmetry add-self-transfer
// settled on for entries.
func (s *Service) authorizeDelete(ctx context.Context, callerID, id string) error {
	current, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	_, _, permStr, err := s.accounts.Access(ctx, current.AccountID, callerID)
	if err != nil {
		return err
	}
	permission := Permission(permStr)
	if !permission.AtLeast(PermissionView) {
		return ErrNotFound
	}
	if !permission.AtLeast(PermissionAppend) {
		return ErrForbidden
	}
	if permission == PermissionAppend && current.CreatedBy != callerID {
		return ErrForbidden
	}
	return nil
}

// HasSelfTransferAccount satisfies internal/account's lookup for the
// currency-immutability rule: it reports whether accountID is named on
// either side of any non-deleted self_transfer recurring transaction.
// Deliberately unauthenticated, like SameAccount — it answers a question
// about the account being edited, whose own permission check has already
// run in internal/account.
func (s *Service) HasSelfTransferAccount(ctx context.Context, accountID string) (bool, error) {
	return s.store.HasSelfTransferAccount(ctx, accountID)
}

// Delete soft-deletes id, authorized for callerID per authorizeDelete, and
// rejected (ErrInUse) while any non-deleted entry still links to it.
func (s *Service) Delete(ctx context.Context, callerID, id string) error {
	if err := s.authorizeDelete(ctx, callerID, id); err != nil {
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

// resolveFilter turns a caller-supplied Filter into the one a Store reads:
// f.AccountIDs narrowed to callerID's own visible accounts (owned or
// shared), intersected with any caller-supplied AccountIDs, and
// f.CategoryID expanded into f.CategoryIDs. It mirrors internal/entry's
// resolveFilter, with one deliberate omission — there is no AllAccounts
// equivalent here, so a category or tag filter narrows the account scope
// and never widens it (see Filter's doc comment and design.md of
// add-recurring-list-filters).
//
// CategoryLookup.Subtree doubles as the category's permission check: it
// returns nothing for a category callerID neither owns nor holds a share
// on. That case must match no recurring transaction rather than every one
// in scope, so CategoryIDs is left nil only when there is no category
// filter at all, and is otherwise a non-nil — possibly empty — set.
func (s *Service) resolveFilter(ctx context.Context, callerID string, f Filter) (Filter, error) {
	if !f.CategoryMode.valid() {
		return Filter{}, ErrInvalidValue
	}

	visible, err := s.accounts.VisibleIDs(ctx, callerID)
	if err != nil {
		return Filter{}, err
	}
	if len(f.AccountIDs) > 0 {
		f.AccountIDs = intersect(f.AccountIDs, visible)
	} else {
		f.AccountIDs = visible
	}

	if f.CategoryID != nil {
		if f.CategoryMode == CategoryModeExact {
			f.CategoryIDs = []string{*f.CategoryID}
		} else {
			permitted, err := s.categories.Subtree(ctx, callerID, *f.CategoryID)
			if err != nil {
				return Filter{}, err
			}
			// Never nil past here: an empty subtree is a filter that
			// matches nothing, not the absence of one.
			f.CategoryIDs = append([]string{}, permitted...)
		}
	}

	return f, nil
}

// List resolves f's caller-supplied AccountIDs and CategoryID against
// callerID's visible accounts and category tree, and returns every
// matching recurring transaction, decorated.
func (s *Service) List(ctx context.Context, callerID string, f Filter) ([]RecurringTransaction, error) {
	f, err := s.resolveFilter(ctx, callerID, f)
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
	f, err := s.resolveFilter(ctx, callerID, f)
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

// Preview resolves f's caller-supplied AccountIDs the same way List/Summary
// do, then for each matching, non-deleted recurring transaction whose
// category/tag also match f (when supplied), generates its projected future
// occurrences up to min(f.To, that template's own EndsOn) via
// previewOccurrences, and returns every template's occurrences merged and
// sorted by BookingTimestamp ascending. f.To is required (ErrInvalidValue
// when zero) — Preview never computes an unbounded result. See design.md's
// "one new endpoint, no persistence" and "exactly one overdue row per
// template" decisions.
func (s *Service) Preview(ctx context.Context, callerID string, f PreviewFilter) ([]PreviewItem, error) {
	if f.To.IsZero() {
		return nil, ErrInvalidValue
	}

	// Preview always projects both legs of a self_transfer, with no
	// equivalent of List/Summary's caller-chosen mode — see PreviewItem's
	// doc comment and design.md. Everything else it narrows by is the
	// listing's own Filter, resolved by the listing's own rule.
	resolved, err := s.resolveFilter(ctx, callerID, Filter{
		AccountIDs:    f.AccountIDs,
		SelfTransfers: SelfTransferBothLegs,
		CategoryID:    f.CategoryID,
		CategoryMode:  f.CategoryMode,
		TagID:         f.TagID,
	})
	if err != nil {
		return nil, err
	}
	rows, err := s.store.List(ctx, resolved)
	if err != nil {
		return nil, err
	}

	today := s.today(ctx, callerID)
	cutoff := NewDate(f.To)

	var items []PreviewItem
	for _, rt := range rows {
		// A template already ended (ends_on at or before today) never
		// contributes, regardless of how stale its next_suggested_date is
		// — mirrors Summary's own ended-exclusion above.
		if Ended(rt.EndsOn, today) {
			continue
		}

		anchor, err := s.nextSuggestedDate(ctx, rt)
		if err != nil {
			return nil, err
		}
		currency, _, _, err := s.accounts.Access(ctx, rt.AccountID, callerID)
		if err != nil {
			return nil, err
		}

		for _, d := range previewOccurrences(anchor, rt.IntervalUnit, rt.IntervalCount, cutoff, today, rt.EndsOn) {
			items = append(items, PreviewItem{
				RecurringTransactionID: rt.ID,
				AccountID:              rt.AccountID,
				AccountCurrency:        currency,
				ToAccountID:            rt.ToAccountID,
				ToAccountName:          rt.ToAccountName,
				Kind:                   rt.Kind,
				Title:                  rt.Title,
				Description:            rt.Description,
				CategoryID:             rt.CategoryID,
				Counterparty:           rt.Counterparty,
				Location:               rt.Location,
				TagIDs:                 rt.TagIDs,
				Amount:                 rt.Amount,
				BookingTimestamp:       d,
				Overdue:                d.Before(today),
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].BookingTimestamp.Time.Equal(items[j].BookingTimestamp.Time) {
			return items[i].RecurringTransactionID < items[j].RecurringTransactionID
		}
		return items[i].BookingTimestamp.Before(items[j].BookingTimestamp)
	})
	return items, nil
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
