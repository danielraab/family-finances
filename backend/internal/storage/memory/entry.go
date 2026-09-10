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
// the tie-break for entries sharing an identical booking_timestamp (mirrors
// entries.id bigserial in the real backend).
type entryRow struct {
	e   entry.Entry
	seq int64
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
	// just-assigned id) using its Balance reading. A transaction stores
	// in.Amount as-is, with Balance left nil.
	var amount int64
	if in.Kind == entry.KindTransaction && in.Amount != nil {
		amount = *in.Amount
	}
	e := entry.Entry{
		ID:               strconv.FormatInt(s.seq, 10),
		CreatedBy:        createdBy,
		AccountID:        in.AccountID,
		Kind:             in.Kind,
		Amount:           amount,
		Balance:          in.Balance,
		BookingTimestamp: in.BookingTimestamp,
		Title:            in.Title,
		Description:      in.Description,
		CategoryID:       in.CategoryID,
		TagIDs:           tagIDs,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.rows[e.ID] = entryRow{e: e, seq: s.seq}
	s.recomputeFromLocked(e.AccountID, e.BookingTimestamp, s.seq, 0)
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
	if upd.TagIDs != nil {
		tagIDs := make([]string, len(*upd.TagIDs))
		copy(tagIDs, *upd.TagIDs)
		e.TagIDs = tagIDs
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
	var before int64
	for _, row := range s.rows {
		e := row.e
		if e.AccountID != accountID || e.DeletedAt != nil {
			continue
		}
		if comparePos(e.BookingTimestamp, row.seq, ts, seq) < 0 {
			before += e.Amount
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

// matchingRows returns every non-deleted entry matching f's
// account/category/tag/kind/date-range/query filters, in no particular
// order. Scoping to the caller happens entirely through f.AccountIDs —
// already narrowed by Service to the caller's visible (owned or shared)
// accounts before it reaches here, per design.md's "entry read/write
// authorization moves from the Store layer into the Service layer"
// decision — never by an owner/creator equality check, so a viewer sees
// every entry on a visible account regardless of who created it. List and
// Sum both build on this so their filtering logic can never diverge.
// Callers hold s.mu.
func (s *EntryStore) matchingRows(f entry.Filter) []entryRow {
	if len(f.AccountIDs) == 0 {
		return nil
	}
	accountSet := toSet(f.AccountIDs)
	var categorySet map[string]bool
	if f.CategoryID != nil {
		categorySet = toSet(f.CategoryIDs)
	}

	var rows []entryRow
	for _, row := range s.rows {
		e := row.e
		if e.DeletedAt != nil {
			continue
		}
		if !accountSet[e.AccountID] {
			continue
		}
		if categorySet != nil && (e.CategoryID == nil || !categorySet[*e.CategoryID]) {
			continue
		}
		if f.TagID != nil && !containsString(e.TagIDs, *f.TagID) {
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
			if !strings.Contains(strings.ToLower(e.Title), q) && !strings.Contains(strings.ToLower(e.Description), q) {
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
		next = &entry.Cursor{BookingTimestamp: last.e.BookingTimestamp, Amount: last.e.Amount, ID: last.e.ID}
		rows = rows[:f.Limit]
	}

	out := make([]entry.Entry, len(rows))
	for i, row := range rows {
		out[i] = row.e
	}
	return out, next, nil
}

// sortLess reports whether a sorts before b in ascending order of field,
// breaking ties by insertion sequence (ascending).
func sortLess(a, b entryRow, field entry.SortField) bool {
	if field == entry.SortAmount {
		if a.e.Amount != b.e.Amount {
			return a.e.Amount < b.e.Amount
		}
		return a.seq < b.seq
	}
	if !a.e.BookingTimestamp.Equal(b.e.BookingTimestamp) {
		return a.e.BookingTimestamp.Before(b.e.BookingTimestamp)
	}
	return a.seq < b.seq
}

// isPastCursor reports whether row comes strictly after cursor in the
// listing's order (asc or desc) — i.e. it belongs on the next page.
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
	if asc {
		return cmp > 0
	}
	return cmp < 0
}

// Balance implements design.md's live computation: the sum of every
// non-deleted entry's amount up to asOf. A balance adjustment's amount is
// kept, by the recompute helpers above (run inside Create/Update/
// SoftDelete), equal to its own Balance reading minus the balance strictly
// before it — so this plain sum reproduces exactly the same result as
// always resetting to the latest adjustment and summing only the
// transactions after it, with no kind-branching needed here.
func (s *EntryStore) Balance(_ context.Context, accountID string, asOf time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var sum int64
	for _, row := range s.rows {
		e := row.e
		if e.AccountID != accountID || e.DeletedAt != nil {
			continue
		}
		if e.BookingTimestamp.After(asOf) {
			continue
		}
		sum += e.Amount
	}
	return sum, nil
}

// Sum implements entry.Store's Sum: f's matching entries, restricted to
// kind: transaction regardless of f.Kind, summed per account id.
func (s *EntryStore) Sum(_ context.Context, f entry.Filter) (map[string]int64, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	txKind := entry.KindTransaction
	f.Kind = &txKind
	rows := s.matchingRows(f)

	perAccount := make(map[string]int64, len(rows))
	for _, row := range rows {
		perAccount[row.e.AccountID] += row.e.Amount
	}
	return perAccount, len(rows), nil
}

// FlowSummary implements entry.Store's FlowSummary: one row per (account,
// period) combination with at least one matching entry, bucketed by
// booking_timestamp converted into f.Timezone (Service.FlowSummary already
// resolved it, default "UTC") and truncated to f.Unit.
func (s *EntryStore) FlowSummary(_ context.Context, f entry.FlowFilter) ([]entry.FlowRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	loc, err := time.LoadLocation(f.Timezone)
	if err != nil {
		return nil, err
	}
	accountSet := toSet(f.AccountIDs)

	type key struct{ period, accountID string }
	totals := map[key]*entry.FlowRow{}

	for _, row := range s.rows {
		e := row.e
		if e.DeletedAt != nil || !accountSet[e.AccountID] {
			continue
		}
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
