package memory

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	rt "at.draab/familyfinances/internal/recurringtransaction"
)

// RecurringTransactionStore is the in-memory implementation of
// recurringtransaction.Store — the default for domain and handler tests
// and for local runs without a database. Safe for concurrent use. Mirrors
// EntryStore's shape.
type RecurringTransactionStore struct {
	mu   sync.Mutex
	rows map[string]rt.RecurringTransaction
	seq  int64
}

// NewRecurringTransactionStore returns an empty RecurringTransactionStore.
func NewRecurringTransactionStore() *RecurringTransactionStore {
	return &RecurringTransactionStore{rows: map[string]rt.RecurringTransaction{}}
}

func (s *RecurringTransactionStore) Create(_ context.Context, createdBy string, in rt.New) (rt.RecurringTransaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	now := time.Now().UTC()
	tagIDs := make([]string, len(in.TagIDs))
	copy(tagIDs, in.TagIDs)

	row := rt.RecurringTransaction{
		ID:            strconv.FormatInt(s.seq, 10),
		CreatedBy:     createdBy,
		AccountID:     in.AccountID,
		Title:         in.Title,
		Description:   in.Description,
		CategoryID:    in.CategoryID,
		Counterparty:  in.Counterparty,
		Location:      in.Location,
		TagIDs:        tagIDs,
		Amount:        in.Amount,
		IntervalUnit:  in.IntervalUnit,
		IntervalCount: in.IntervalCount,
		StartsOn:      in.StartsOn,
		EndsOn:        in.EndsOn,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.rows[row.ID] = row
	return row, nil
}

func (s *RecurringTransactionStore) Get(_ context.Context, id string) (rt.RecurringTransaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok || row.DeletedAt != nil {
		return rt.RecurringTransaction{}, rt.ErrNotFound
	}
	return row, nil
}

func (s *RecurringTransactionStore) Update(_ context.Context, id string, upd rt.Update) (rt.RecurringTransaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok || row.DeletedAt != nil {
		return rt.RecurringTransaction{}, rt.ErrNotFound
	}

	if upd.AccountID != nil {
		row.AccountID = *upd.AccountID
	}
	if upd.Title != nil {
		row.Title = *upd.Title
	}
	if upd.Description != nil {
		row.Description = *upd.Description
	}
	if upd.CategoryID != nil {
		row.CategoryID = upd.CategoryID
	}
	if upd.Counterparty != nil {
		row.Counterparty = *upd.Counterparty
	}
	if upd.Location != nil {
		row.Location = *upd.Location
	}
	if upd.TagIDs != nil {
		tagIDs := make([]string, len(*upd.TagIDs))
		copy(tagIDs, *upd.TagIDs)
		row.TagIDs = tagIDs
	}
	if upd.Amount != nil {
		row.Amount = *upd.Amount
	}
	if upd.IntervalUnit != nil {
		row.IntervalUnit = *upd.IntervalUnit
	}
	if upd.IntervalCount != nil {
		row.IntervalCount = *upd.IntervalCount
	}
	if upd.StartsOn != nil {
		row.StartsOn = *upd.StartsOn
	}
	if upd.EndsOn.Set {
		row.EndsOn = upd.EndsOn.Value
	}
	row.UpdatedAt = time.Now().UTC()
	s.rows[id] = row
	return row, nil
}

func (s *RecurringTransactionStore) SoftDelete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.rows[id]
	if !ok || row.DeletedAt != nil {
		return rt.ErrNotFound
	}
	now := time.Now().UTC()
	row.DeletedAt = &now
	s.rows[id] = row
	return nil
}

func (s *RecurringTransactionStore) List(_ context.Context, f rt.Filter) ([]rt.RecurringTransaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	accountSet := toSet(f.AccountIDs)
	var out []rt.RecurringTransaction
	for _, row := range s.rows {
		if row.DeletedAt != nil || !accountSet[row.AccountID] {
			continue
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}
