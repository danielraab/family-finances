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

func (s *EntryStore) Create(_ context.Context, ownerID string, in entry.New) (entry.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	now := time.Now().UTC()
	tagIDs := make([]string, len(in.TagIDs))
	copy(tagIDs, in.TagIDs)
	e := entry.Entry{
		ID:               strconv.FormatInt(s.seq, 10),
		OwnerID:          ownerID,
		AccountID:        in.AccountID,
		Kind:             in.Kind,
		Amount:           in.Amount,
		BookingTimestamp: in.BookingTimestamp,
		Title:            in.Title,
		Description:      in.Description,
		CategoryID:       in.CategoryID,
		TagIDs:           tagIDs,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.rows[e.ID] = entryRow{e: e, seq: s.seq}
	return e, nil
}

func (s *EntryStore) Get(_ context.Context, ownerID, id string) (entry.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok || row.e.OwnerID != ownerID || row.e.DeletedAt != nil {
		return entry.Entry{}, entry.ErrNotFound
	}
	return row.e, nil
}

func (s *EntryStore) Update(_ context.Context, ownerID, id string, upd entry.Update) (entry.Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok || row.e.OwnerID != ownerID || row.e.DeletedAt != nil {
		return entry.Entry{}, entry.ErrNotFound
	}
	e := row.e
	if upd.Amount != nil {
		e.Amount = *upd.Amount
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
	return e, nil
}

func (s *EntryStore) SoftDelete(_ context.Context, ownerID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok || row.e.OwnerID != ownerID || row.e.DeletedAt != nil {
		return entry.ErrNotFound
	}
	now := time.Now().UTC()
	row.e.DeletedAt = &now
	s.rows[id] = row
	return nil
}

func (s *EntryStore) List(_ context.Context, ownerID string, f entry.Filter) ([]entry.Entry, *entry.Cursor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(f.AccountIDs) == 0 {
		return nil, nil, nil
	}
	accountSet := toSet(f.AccountIDs)
	var categorySet map[string]bool
	if f.CategoryID != nil {
		categorySet = toSet(f.CategoryIDs)
	}

	var rows []entryRow
	for _, row := range s.rows {
		e := row.e
		if e.OwnerID != ownerID || e.DeletedAt != nil {
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

func (s *EntryStore) Balance(_ context.Context, accountID string, asOf time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var base int64
	var baseAt time.Time // zero value = -infinity: every transaction counts
	var baseSeq int64 = -1
	found := false
	for _, row := range s.rows {
		e := row.e
		if e.AccountID != accountID || e.DeletedAt != nil || e.Kind != entry.KindBalanceAdjustment {
			continue
		}
		if e.BookingTimestamp.After(asOf) {
			continue
		}
		if !found || e.BookingTimestamp.After(baseAt) || (e.BookingTimestamp.Equal(baseAt) && row.seq > baseSeq) {
			found = true
			base = e.Amount
			baseAt = e.BookingTimestamp
			baseSeq = row.seq
		}
	}

	var sum int64
	for _, row := range s.rows {
		e := row.e
		if e.AccountID != accountID || e.DeletedAt != nil || e.Kind != entry.KindTransaction {
			continue
		}
		if e.BookingTimestamp.After(asOf) {
			continue
		}
		if e.BookingTimestamp.Before(baseAt) {
			continue
		}
		if e.BookingTimestamp.Equal(baseAt) && row.seq <= baseSeq {
			continue
		}
		sum += e.Amount
	}
	return base + sum, nil
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
