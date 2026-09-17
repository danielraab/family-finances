package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"at.draab/familyfinances/internal/entry"
)

// entryRow wraps an entry.Entry with its insertion sequence number, used as
// a tie-break for entries sharing an identical booking_timestamp (mirrors
// entries.id bigserial in the real backend). native distinguishes a
// self_transfer's two synthesized "view" rows (see legsLocked) — true for
// the row oriented as stored (AccountID unchanged), false for the flipped
// one (AccountID/ToAccountID swapped, Amount negated) — used as a final
// tie-break in sortLess/isPastCursor so the two never compare equal; every
// other kind only ever produces one (native) row. A raw entryRow taken
// directly from s.rows (rather than legsLocked's output) always has
// native's zero value (false) and must never be compared on it — the
// pre-leg-expansion map is keyed by real entry id, not by (id, leg).
type entryRow struct {
	e      entry.Entry
	seq    int64
	native bool
}

// legsLocked returns every non-deleted entry as one entryRow (native) plus,
// for a self_transfer, a second one (not native) for ToAccountID with
// AccountID/ToAccountID swapped and Amount negated — mirroring the
// entry_legs view in storage/postgres exactly, so List/Sum/FlowSummary see
// the same "once per account touched" shape in both stores. Callers hold
// s.mu.
func (s *EntryStore) legsLocked() []entryRow {
	legs := make([]entryRow, 0, len(s.rows))
	for _, row := range s.rows {
		if row.e.DeletedAt != nil {
			continue
		}
		legs = append(legs, entryRow{e: row.e, seq: row.seq, native: true})
		if row.e.Kind == entry.KindSelfTransfer {
			flipped := row.e
			sender := row.e.AccountID
			flipped.AccountID = *row.e.ToAccountID
			flipped.ToAccountID = &sender
			flipped.Amount = -row.e.Amount
			legs = append(legs, entryRow{e: flipped, seq: row.seq, native: false})
		}
	}
	return legs
}

// EntryStore is the in-memory implementation of entry.Store — the default
// for domain and handler tests and for local runs without a database. Safe
// for concurrent use.
type EntryStore struct {
	mu   sync.Mutex
	rows map[string]entryRow
	seq  int64
}

// NewEntryStore returns an empty EntryStore.
func NewEntryStore() *EntryStore {
	return &EntryStore{rows: map[string]entryRow{}}
}

func (s *EntryStore) Create(_ context.Context, createdBy string, in entry.New) (entry.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	now := time.Now().UTC()
	tagIDs := make([]string, len(in.TagIDs))
	copy(tagIDs, in.TagIDs)
	// A balance adjustment's amount is never client-supplied — it starts as
	// a 0 placeholder, immediately overwritten below by recomputeFrom
	// (which finds this very row, since nothing else can share its
	// just-assigned id) using its Balance reading. A transaction or
	// self_transfer stores in.Amount as-is, with Balance left nil.
	var amount int64
	if in.Kind != entry.KindBalanceAdjustment && in.Amount != nil {
		amount = *in.Amount
	}
	e := entry.Entry{
		ID:                     strconv.FormatInt(s.seq, 10),
		CreatedBy:              createdBy,
		AccountID:              in.AccountID,
		ToAccountID:            in.ToAccountID,
		Kind:                   in.Kind,
		Amount:                 amount,
		Balance:                in.Balance,
		BookingTimestamp:       in.BookingTimestamp,
		Title:                  in.Title,
		Description:            in.Description,
		CategoryID:             in.CategoryID,
		Counterparty:           in.Counterparty,
		Location:               in.Location,
		TagIDs:                 tagIDs,
		CreatedAt:              now,
		UpdatedAt:              now,
		RecurringTransactionID: in.RecurringTransactionID,
	}
	s.rows[e.ID] = entryRow{e: e, seq: s.seq}
	s.recomputeFromLocked(e.AccountID, e.BookingTimestamp, s.seq, 0)
	if e.Kind == entry.KindSelfTransfer {
		s.recomputeFromLocked(*e.ToAccountID, e.BookingTimestamp, s.seq, 0)
	}
	return s.rows[e.ID].e, nil
}

func (s *EntryStore) Get(_ context.Context, id string) (entry.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok || row.e.DeletedAt != nil {
		return entry.Entry{}, entry.ErrNotFound
	}
	return row.e, nil
}

func (s *EntryStore) Update(_ context.Context, id string, upd entry.Update) (entry.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok || row.e.DeletedAt != nil {
		return entry.Entry{}, entry.ErrNotFound
	}
	// Captured before the update so both the position this entry is
	// leaving and the one it lands on (if either its account or its
	// booking_timestamp changes) can have their nearest balance adjustment
	// recomputed — see design.md's algorithm.
	oldAccountID := row.e.AccountID
	oldTS := row.e.BookingTimestamp
	seq := row.seq

	e := row.e
	if upd.AccountID != nil {
		e.AccountID = *upd.AccountID
	}
	if upd.Amount != nil {
		e.Amount = *upd.Amount
	}
	if upd.Balance != nil {
		e.Balance = upd.Balance
	}
	if upd.BookingTimestamp != nil {
		e.BookingTimestamp = *upd.BookingTimestamp
	}
	if upd.Title != nil {
		e.Title = *upd.Title
	}
	if upd.Description != nil {
		e.Description = *upd.Description
	}
	if upd.CategoryID.Set {
		e.CategoryID = upd.CategoryID.Value
	}
	if upd.Counterparty != nil {
		e.Counterparty = *upd.Counterparty
	}
	if upd.Location != nil {
		e.Location = *upd.Location
	}
	if upd.TagIDs != nil {
		tagIDs := make([]string, len(*upd.TagIDs))
		copy(tagIDs, *upd.TagIDs)
		e.TagIDs = tagIDs
	}
	if upd.RecurringTransactionID.Set {
		e.RecurringTransactionID = upd.RecurringTransactionID.Value
	}
	e.UpdatedAt = time.Now().UTC()
	row.e = e
	s.rows[id] = row

	if e.AccountID != oldAccountID || !e.BookingTimestamp.Equal(oldTS) {
		// This row already carries its new account/booking_timestamp, so
		// the old-position search must exclude it explicitly (seq) —
		// otherwise, if only the timestamp moved later within the same
		// account, its own (already-updated) row could still satisfy the
		// old position's "at or after" comparison and be mistaken for what
		// is still there.
		s.recomputeFromLocked(oldAccountID, oldTS, seq, seq)
		s.recomputeFromLocked(e.AccountID, e.BookingTimestamp, seq, 0)
	} else {
		s.recomputeFromLocked(oldAccountID, oldTS, seq, 0)
	}
	// A self_transfer's to_account_id never changes via Update — Service
	// rejects an AccountID change on this kind, and there is no
	// ToAccountID field on Update at all — so its own recompute only ever
	// needs to react to a booking_timestamp change, on the same fixed
	// account.
	if e.Kind == entry.KindSelfTransfer {
		if !e.BookingTimestamp.Equal(oldTS) {
			s.recomputeFromLocked(*e.ToAccountID, oldTS, seq, seq)
			s.recomputeFromLocked(*e.ToAccountID, e.BookingTimestamp, seq, 0)
		} else {
			s.recomputeFromLocked(*e.ToAccountID, oldTS, seq, 0)
		}
	}

	return s.rows[id].e, nil
}

func (s *EntryStore) SoftDelete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok || row.e.DeletedAt != nil {
		return entry.ErrNotFound
	}
	now := time.Now().UTC()
	row.e.DeletedAt = &now
	s.rows[id] = row
	// DeletedAt is already set, so findAdjustmentLocked's own DeletedAt
	// filter already excludes this row — no explicit exclusion needed the
	// way a moved (but still live) entry needs in Update.
	s.recomputeFromLocked(row.e.AccountID, row.e.BookingTimestamp, row.seq, 0)
	if row.e.Kind == entry.KindSelfTransfer {
		s.recomputeFromLocked(*row.e.ToAccountID, row.e.BookingTimestamp, row.seq, 0)
	}
	return nil
}

// comparePos orders two (booking_timestamp, seq) positions: -1 if the first
// sorts before the second, 1 if after, 0 if equal. seq is the insertion
// sequence tie-break, mirroring entries.id in the real backend.
func comparePos(ts1 time.Time, seq1 int64, ts2 time.Time, seq2 int64) int {
	if ts1.Before(ts2) {
		return -1
	}
	if ts1.After(ts2) {
		return 1
	}
	switch {
	case seq1 < seq2:
		return -1
	case seq1 > seq2:
		return 1
	default:
		return 0
	}
}

// findAdjustmentLocked locates the earliest non-deleted balance_adjustment
// on accountID at (inclusive=true) or strictly after (inclusive=false)
// position (ts, seq), optionally excluding one row by its seq (0 = exclude
// none) — mirrors postgres.findAdjustment's role in the same algorithm.
// Callers hold s.mu.
func (s *EntryStore) findAdjustmentLocked(accountID string, ts time.Time, seq int64, inclusive bool, excludeSeq int64) (id string, foundSeq int64, foundTS time.Time, reading int64, ok bool) {
	for candID, row := range s.rows {
		e := row.e
		if e.AccountID != accountID || e.Kind != entry.KindBalanceAdjustment || e.DeletedAt != nil {
			continue
		}
		if excludeSeq != 0 && row.seq == excludeSeq {
			continue
		}
		cmp := comparePos(e.BookingTimestamp, row.seq, ts, seq)
		if inclusive && cmp < 0 {
			continue
		}
		if !inclusive && cmp <= 0 {
			continue
		}
		if !ok || comparePos(e.BookingTimestamp, row.seq, foundTS, foundSeq) < 0 {
			ok = true
			id = candID
			foundSeq = row.seq
			foundTS = e.BookingTimestamp
			reading = 0
			if e.Balance != nil {
				reading = *e.Balance
			}
		}
	}
	return
}

// setAmountLocked recomputes and stores the amount for the balance
// adjustment (id, seq, ts) on accountID: its Balance reading minus the
// balance strictly before its own position. Callers hold s.mu.
func (s *EntryStore) setAmountLocked(accountID, id string, ts time.Time, seq int64, reading int64) {
	// Mirrors Balance's own AccountID/ToAccountID handling — a self_transfer
	// whose ToAccountID is accountID contributes -Amount to the balance
	// strictly before this adjustment's position, exactly as it does to
	// Balance() itself; omitting it here left a receiving account's
	// adjustments permanently under-recomputed (a real bug the postgres
	// store's self-transfer tests caught, mirrored here for parity).
	var before int64
	for _, row := range s.rows {
		e := row.e
		if e.DeletedAt != nil {
			continue
		}
		var contribution int64
		switch {
		case e.AccountID == accountID:
			contribution = e.Amount
		case e.Kind == entry.KindSelfTransfer && *e.ToAccountID == accountID:
			contribution = -e.Amount
		default:
			continue
		}
		if comparePos(e.BookingTimestamp, row.seq, ts, seq) < 0 {
			before += contribution
		}
	}
	row := s.rows[id]
	row.e.Amount = reading - before
	s.rows[id] = row
}

// recomputeFromLocked keeps every balance adjustment's stored amount
// correct after a mutation at (accountID, ts, seq) — see design.md's
// algorithm and worked example: only the earliest non-deleted adjustment
// at or after that position (A1), and the one immediately after it (A2),
// can possibly need a new amount; nothing further downstream ever changes.
// excludeSeq (0 = none) is forwarded to the A1 search only — see
// findAdjustmentLocked. Callers hold s.mu.
func (s *EntryStore) recomputeFromLocked(accountID string, ts time.Time, seq int64, excludeSeq int64) {
	a1ID, a1Seq, a1TS, a1Reading, ok := s.findAdjustmentLocked(accountID, ts, seq, true, excludeSeq)
	if !ok {
		return
	}
	s.setAmountLocked(accountID, a1ID, a1TS, a1Seq, a1Reading)

	a2ID, a2Seq, a2TS, a2Reading, ok := s.findAdjustmentLocked(accountID, a1TS, a1Seq, false, 0)
	if !ok {
		return
	}
	s.setAmountLocked(accountID, a2ID, a2TS, a2Seq, a2Reading)
}

// matchingRows returns every non-deleted entry leg (see legsLocked)
// matching f's account/category/tag/kind/date-range/query filters, in no
// particular order. Scoping to the caller happens through f.AccountIDs —
// already narrowed by Service to the caller's visible (owned or shared)
// accounts before it reaches here, per design.md's "entry read/write
// authorization moves from the Store layer into the Service layer"
// decision — never by an owner/creator equality check, so a viewer sees
// every entry on a visible account regardless of who created it. When
// f.AllAccounts is set, the account membership check is skipped entirely:
// the caller's permission on the filtered category (f.CategoryIDs,
// resolved by entry.Service) is what authorizes those rows instead.
// Filtering leg.e.AccountID (each leg already oriented per legsLocked) is
// what makes a self_transfer match once per account it touches — twice
// when both are in f.AccountIDs at once, mirroring the entry_legs view in
// storage/postgres. List and Sum both build on this so their filtering
// logic can never diverge. Callers hold s.mu.
func (s *EntryStore) matchingRows(f entry.Filter) []entryRow {
	if !f.AllAccounts && len(f.AccountIDs) == 0 {
		return nil
	}
	accountSet := toSet(f.AccountIDs)
	var categorySet map[string]bool
	if f.CategoryID != nil {
		categorySet = toSet(f.CategoryIDs)
	}

	var rows []entryRow
	for _, row := range s.legsLocked() {
		e := row.e
		if !f.AllAccounts && !accountSet[e.AccountID] {
			continue
		}
		if categorySet != nil && (e.CategoryID == nil || !categorySet[*e.CategoryID]) {
			continue
		}
		if f.TagID != nil && !containsString(e.TagIDs, *f.TagID) {
			continue
		}
		if f.RecurringTransactionID != nil &&
			(e.RecurringTransactionID == nil || *e.RecurringTransactionID != *f.RecurringTransactionID) {
			continue
		}
		if f.Kind != nil && e.Kind != *f.Kind {
			continue
		}
		if f.From != nil && e.BookingTimestamp.Before(*f.From) {
			continue
		}
		if f.To != nil && e.BookingTimestamp.After(*f.To) {
			continue
		}
		if f.Query != "" {
			q := strings.ToLower(f.Query)
			if !strings.Contains(strings.ToLower(e.Title), q) &&
				!strings.Contains(strings.ToLower(e.Description), q) &&
				!strings.Contains(strings.ToLower(e.Counterparty), q) {
				continue
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func (s *EntryStore) List(_ context.Context, f entry.Filter) ([]entry.Entry, *entry.Cursor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows := s.matchingRows(f)

	asc := f.Dir == entry.DirAsc
	sort.Slice(rows, func(i, j int) bool {
		less := sortLess(rows[i], rows[j], f.Sort)
		if asc {
			return less
		}
		return sortLess(rows[j], rows[i], f.Sort)
	})

	if f.After != nil {
		cut := 0
		for i, row := range rows {
			if isPastCursor(row, *f.After, f.Sort, asc) {
				cut = i
				break
			}
			cut = i + 1
		}
		rows = rows[cut:]
	}

	var next *entry.Cursor
	if len(rows) > f.Limit {
		last := rows[f.Limit-1]
		next = &entry.Cursor{BookingTimestamp: last.e.BookingTimestamp, Amount: last.e.Amount, ID: last.e.ID, Native: last.native}
		rows = rows[:f.Limit]
	}

	out := make([]entry.Entry, len(rows))
	for i, row := range rows {
		out[i] = row.e
	}
	return out, next, nil
}

// sortLess reports whether a sorts before b in ascending order of field,
// breaking ties by insertion sequence (ascending), then finally by native
// (false before true) — the last tie-break only ever matters between a
// self_transfer's two legs, which always share the same seq (see
// entryRow's doc comment); every other pair of rows has a distinct seq
// long before native is ever consulted.
func sortLess(a, b entryRow, field entry.SortField) bool {
	if field == entry.SortAmount {
		if a.e.Amount != b.e.Amount {
			return a.e.Amount < b.e.Amount
		}
	} else if !a.e.BookingTimestamp.Equal(b.e.BookingTimestamp) {
		return a.e.BookingTimestamp.Before(b.e.BookingTimestamp)
	}
	if a.seq != b.seq {
		return a.seq < b.seq
	}
	return !a.native && b.native
}

// isPastCursor reports whether row comes strictly after cursor in the
// listing's order (asc or desc) — i.e. it belongs on the next page. native
// is the final tie-break, mirroring sortLess/the postgres store's keyset
// tuple — without it, a self_transfer's two legs (identical sortCol and
// seq) would compare equal and a page boundary landing between them could
// silently drop whichever one didn't make the previous page.
func isPastCursor(row entryRow, cursor entry.Cursor, field entry.SortField, asc bool) bool {
	var cmp int
	if field == entry.SortAmount {
		switch {
		case row.e.Amount < cursor.Amount:
			cmp = -1
		case row.e.Amount > cursor.Amount:
			cmp = 1
		}
	} else {
		switch {
		case row.e.BookingTimestamp.Before(cursor.BookingTimestamp):
			cmp = -1
		case row.e.BookingTimestamp.After(cursor.BookingTimestamp):
			cmp = 1
		}
	}
	if cmp == 0 {
		rowSeq := row.seq
		cursorSeq, _ := strconv.ParseInt(cursor.ID, 10, 64)
		switch {
		case rowSeq < cursorSeq:
			cmp = -1
		case rowSeq > cursorSeq:
			cmp = 1
		}
	}
	if cmp == 0 {
		switch {
		case !row.native && cursor.Native:
			cmp = -1
		case row.native && !cursor.Native:
			cmp = 1
		}
	}
	if asc {
		return cmp > 0
	}
	return cmp < 0
}

// Balance implements design.md's live computation: the sum of every
// non-deleted entry's amount up to asOf, from accountID's own point of
// view — a plain row it's the AccountID of contributes Amount as stored; a
// self_transfer it's instead the ToAccountID of contributes -Amount (the
// receiving side's effective delta — see entry.Entry's ToAccountID doc
// comment). A balance adjustment's amount is kept, by the recompute
// helpers above (run inside Create/Update/SoftDelete, on both accounts for
// a self_transfer), equal to its own Balance reading minus the balance
// strictly before it — so this reproduces exactly the same result as
// always resetting to the latest adjustment and summing only the
// transactions after it, with no further kind-branching needed here.
func (s *EntryStore) Balance(_ context.Context, accountID string, asOf time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var sum int64
	for _, row := range s.rows {
		e := row.e
		if e.DeletedAt != nil {
			continue
		}
		if e.BookingTimestamp.After(asOf) {
			continue
		}
		switch {
		case e.AccountID == accountID:
			sum += e.Amount
		case e.Kind == entry.KindSelfTransfer && *e.ToAccountID == accountID:
			sum -= e.Amount
		}
	}
	return sum, nil
}

// Sum implements entry.Store's Sum: f's matching entry legs, restricted to
// kind: transaction or self_transfer regardless of f.Kind, summed per
// (leg) account id — a self_transfer contributes twice, once per account,
// when both are within f.AccountIDs, mirroring List.
func (s *EntryStore) Sum(_ context.Context, f entry.Filter) (map[string]int64, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f.Kind = nil
	rows := s.matchingRows(f)

	perAccount := make(map[string]int64, len(rows))
	count := 0
	for _, row := range rows {
		if row.e.Kind != entry.KindTransaction && row.e.Kind != entry.KindSelfTransfer {
			continue
		}
		perAccount[row.e.AccountID] += row.e.Amount
		count++
	}
	return perAccount, count, nil
}

// FlowSummary implements entry.Store's FlowSummary: one row per (account,
// period) combination with at least one matching entry, bucketed by
// booking_timestamp converted into f.Timezone (Service.FlowSummary already
// resolved it, default "UTC") and truncated to f.Unit. Account/category/tag
// scoping reuses matchingRows (the same filtering List/Sum apply), passing
// no Kind so both transaction and balance_adjustment entries are included.
func (s *EntryStore) FlowSummary(_ context.Context, f entry.FlowFilter) ([]entry.FlowRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	loc, err := time.LoadLocation(f.Timezone)
	if err != nil {
		return nil, err
	}

	rows := s.matchingRows(entry.Filter{
		AccountIDs:  f.AccountIDs,
		AllAccounts: f.AllAccounts,
		CategoryID:  f.CategoryID,
		CategoryIDs: f.CategoryIDs,
		TagID:       f.TagID,
	})

	type key struct{ period, accountID string }
	totals := map[key]*entry.FlowRow{}

	for _, row := range rows {
		e := row.e
		local := e.BookingTimestamp.In(loc)
		if local.Year() != f.Year {
			continue
		}
		if f.Unit == entry.FlowUnitDay && int(local.Month()) != f.Month {
			continue
		}

		var period string
		if f.Unit == entry.FlowUnitDay {
			period = local.Format("2006-01-02")
		} else {
			period = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		}

		k := key{period: period, accountID: e.AccountID}
		r, ok := totals[k]
		if !ok {
			r = &entry.FlowRow{AccountID: e.AccountID, Period: period}
			totals[k] = r
		}
		switch {
		case e.Amount > 0:
			r.Income += e.Amount
		case e.Amount < 0:
			r.Outcome += -e.Amount
		}
	}

	out := make([]entry.FlowRow, 0, len(totals))
	for _, r := range totals {
		out = append(out, *r)
	}
	return out, nil
}

// ListInUseCounterparties returns the distinct, non-empty counterparty
// values on ownerID's own non-deleted entries, sorted case-insensitively
// ascending — mirrors postgres.EntryStore.ListInUseCounterparties.
func (s *EntryStore) ListInUseCounterparties(_ context.Context, ownerID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	set := map[string]bool{}
	for _, row := range s.rows {
		e := row.e
		if e.DeletedAt != nil || e.CreatedBy != ownerID || strings.TrimSpace(e.Counterparty) == "" {
			continue
		}
		set[e.Counterparty] = true
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		li, lj := strings.ToLower(out[i]), strings.ToLower(out[j])
		if li != lj {
			return li < lj
		}
		return out[i] < out[j]
	})
	return out, nil
}

// LatestBookingTimeByRecurringTransaction implements entry.Store's
// LatestBookingTimeByRecurringTransaction: the highest booking_timestamp
// among recurringTransactionID's non-deleted linked entries, or nil.
func (s *EntryStore) LatestBookingTimeByRecurringTransaction(_ context.Context, recurringTransactionID string) (*time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var latest *time.Time
	for _, row := range s.rows {
		e := row.e
		if e.DeletedAt != nil || e.RecurringTransactionID == nil || *e.RecurringTransactionID != recurringTransactionID {
			continue
		}
		if latest == nil || e.BookingTimestamp.After(*latest) {
			ts := e.BookingTimestamp
			latest = &ts
		}
	}
	return latest, nil
}

// CountByRecurringTransaction implements entry.Store's
// CountByRecurringTransaction: the number of recurringTransactionID's
// non-deleted linked entries.
func (s *EntryStore) CountByRecurringTransaction(_ context.Context, recurringTransactionID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, row := range s.rows {
		e := row.e
		if e.DeletedAt != nil || e.RecurringTransactionID == nil || *e.RecurringTransactionID != recurringTransactionID {
			continue
		}
		count++
	}
	return count, nil
}

// HasEntriesForAccount implements entry.Store's HasEntriesForAccount: a
// cheap existence check across both sides accountID could appear on
// (AccountID, or a self_transfer's ToAccountID) — backs internal/
// account's currency-immutability rule.
func (s *EntryStore) HasEntriesForAccount(_ context.Context, accountID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, row := range s.rows {
		e := row.e
		if e.DeletedAt != nil {
			continue
		}
		if e.AccountID == accountID {
			return true, nil
		}
		if e.Kind == entry.KindSelfTransfer && *e.ToAccountID == accountID {
			return true, nil
		}
	}
	return false, nil
}

func toSet(ids []string) map[string]bool {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

func containsString(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
