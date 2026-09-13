package recurringtransaction_test

import (
	"testing"
	"time"

	rt "at.draab/familyfinances/internal/recurringtransaction"
)

func mustDate(t *testing.T, s string) rt.Date {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatal(err)
	}
	return rt.NewDate(parsed)
}

func TestPerYearAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		unit   rt.Unit
		count  int
		want   int64
	}{
		{"monthly", -80000, rt.UnitMonth, 1, -960000},
		{"quarterly (every 3 months)", 300000, rt.UnitMonth, 3, 1200000},
		{"every 2 months", 100000, rt.UnitMonth, 2, 600000},
		{"yearly", 500000, rt.UnitYear, 1, 500000},
		{"every 2 years", 500000, rt.UnitYear, 2, 250000},
		{"weekly", 1000, rt.UnitWeek, 1, 52179}, // 1000 * 365.25/7 = 52178.57...
		{"every day", 100, rt.UnitDay, 1, 36525},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rt.PerYearAmount(tt.amount, tt.unit, tt.count)
			if got != tt.want {
				t.Errorf("PerYearAmount(%d, %s, %d) = %d, want %d", tt.amount, tt.unit, tt.count, got, tt.want)
			}
		})
	}
}

func TestEnded(t *testing.T) {
	today := mustDate(t, "2026-06-15")

	if rt.Ended(nil, today) {
		t.Error("nil ends_on should never be ended")
	}
	past := mustDate(t, "2026-06-14")
	if !rt.Ended(&past, today) {
		t.Error("an ends_on before today should be ended")
	}
	exactly := mustDate(t, "2026-06-15")
	if !rt.Ended(&exactly, today) {
		t.Error("an ends_on equal to today should be ended")
	}
	future := mustDate(t, "2026-06-16")
	if rt.Ended(&future, today) {
		t.Error("an ends_on after today should not be ended")
	}
}

func TestAdvanceMonthlyClampsToMonthEnd(t *testing.T) {
	start := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	got := rt.Advance(start, rt.UnitMonth, 1)
	want := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Advance(Jan 31, +1 month) = %v, want %v", got, want)
	}
}

func TestAdvanceMonthlyNoClampNeeded(t *testing.T) {
	start := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	got := rt.Advance(start, rt.UnitMonth, 1)
	want := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Advance(Jan 15, +1 month) = %v, want %v", got, want)
	}
}

func TestAdvanceQuarterly(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := rt.Advance(start, rt.UnitMonth, 3)
	want := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Advance(Jan 1, +3 months) = %v, want %v", got, want)
	}
}

func TestAdvanceYearlyClampsLeapDay(t *testing.T) {
	start := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
	got := rt.Advance(start, rt.UnitYear, 1)
	want := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Advance(Feb 29 2024, +1 year) = %v, want %v", got, want)
	}
}

func TestAdvanceWeekly(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := rt.Advance(start, rt.UnitWeek, 2)
	want := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Advance(Jan 1, +2 weeks) = %v, want %v", got, want)
	}
}

func TestAdvanceDaily(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := rt.Advance(start, rt.UnitDay, 10)
	want := time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Advance(Jan 1, +10 days) = %v, want %v", got, want)
	}
}
