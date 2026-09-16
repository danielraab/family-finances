// Package dashboard owns each user's own, ordered set of /home dashboard
// cards. It follows the repo's four-file shape (dashboard.go, store.go,
// service.go, handler.go). It imports internal/auth only for
// auth.UserFromContext — never its Store or a database driver.
//
// A card's filter is a plain, inline value on the card itself — there is
// no separate saved-query entity, and this package has no notion of
// /reports. A dashboard is always private to its owner: a card may
// reference an account/category/tag shared with the caller, but the card
// itself, and the layout it belongs to, is never visible to anyone else.
package dashboard

import "strings"

// CardType is immutable after a card is created — changing a card's
// purpose is delete-and-recreate, not an edit, mirroring how an entry's
// Kind is fixed at creation.
type CardType string

const (
	// CardTypeAccountStat shows one account's live balance.
	CardTypeAccountStat CardType = "account_stat"
	// CardTypeQueryStat shows a per-currency sum for an inline filter.
	CardTypeQueryStat CardType = "query_stat"
	// CardTypeEntryList shows the most recent entries matching an inline
	// filter (a fixed page size — the client renders up to 10).
	CardTypeEntryList CardType = "entry_list"
	// CardTypeBarChart shows an income/outcome bar chart for an account
	// or an inline-filtered query.
	CardTypeBarChart CardType = "bar_chart"
	// CardTypeLineChart shows a running-balance line chart for an
	// account or every account.
	CardTypeLineChart CardType = "line_chart"
)

func (t CardType) valid() bool {
	switch t {
	case CardTypeAccountStat, CardTypeQueryStat, CardTypeEntryList, CardTypeBarChart, CardTypeLineChart:
		return true
	default:
		return false
	}
}

// Range mirrors the web client's own /reports filter shape verbatim.
// Preset is a frontend-owned relative-range key (e.g. "this_month"),
// opaque to this package — never interpreted server-side. Only
// meaningful on query_stat/entry_list.
type Range struct {
	Preset string `json:"preset,omitempty"`
	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
}

// Unit selects a bar_chart card's bucketing — mirrors
// GET /api/entries/flow-summary's own unit parameter.
type Unit string

const (
	UnitMonth Unit = "month"
	UnitDay   Unit = "day"
)

// MinColumns/MaxColumns bound an entry_list card's configurable grid
// span — see Config.Columns.
const (
	MinColumns = 2
	MaxColumns = 4
)

// Config is a card's inline filter/target — a plain object whose
// meaningful fields depend on its card's Type (see the package doc and
// design.md's per-type field table):
//
//   - account_stat requires AccountID and accepts no other field.
//   - query_stat and entry_list accept AccountID, CategoryID,
//     IncludeSubcategories, TagID, Range, and Title, all optional, and no
//     other field.
//   - entry_list additionally accepts Columns (2-4, default 2 when
//     unset) — how many columns of the responsive dashboard grid it
//     spans. No other type may set it: account_stat/query_stat always
//     occupy exactly one column, and bar_chart is always full width
//     regardless of any configured span.
//   - bar_chart requires Unit and accepts AccountID, CategoryID,
//     IncludeSubcategories, TagID, and Title besides it, but no Range.
//   - line_chart accepts only AccountID, Title, and ShowRecurringPreview,
//     all optional, and no other field — its running balance
//     (GET /api/entries/balance-series) has no category/tag filter or
//     bucketing choice to expose.
//   - entry_list, bar_chart, and line_chart additionally accept
//     ShowRecurringPreview (default false) — whether the card additionally
//     previews upcoming recurring-transaction occurrences via
//     GET /api/recurring-transactions/preview (an Upcoming block on
//     entry_list, a stacked projected bar segment on bar_chart, a
//     projected balance line on line_chart). No other type may set it.
//
// Title is a free-text label the web client shows in place of its own
// generated filter summary and makes clickable through to /reports with
// this card's filter prefilled — never settable on account_stat, whose
// heading is always its account's own name and which already links to
// that account's own details page.
//
// AccountID/CategoryID/TagID, when present, must each name an entity the
// caller has at least view permission on — checked by Service, not here,
// since it requires the lookup interfaces in store.go.
type Config struct {
	AccountID            *string `json:"account_id,omitempty"`
	CategoryID           *string `json:"category_id,omitempty"`
	IncludeSubcategories *bool   `json:"include_subcategories,omitempty"`
	TagID                *string `json:"tag_id,omitempty"`
	Range                *Range  `json:"range,omitempty"`
	Unit                 *string `json:"unit,omitempty"`
	Columns              *int    `json:"columns,omitempty"`
	Title                *string `json:"title,omitempty"`
	ShowRecurringPreview *bool   `json:"show_recurring_preview,omitempty"`
}

// Card is one node in a user's own, ordered dashboard. It has exactly one
// owner and is never shared.
type Card struct {
	ID        string   `json:"id"`
	Type      CardType `json:"type"`
	Config    Config   `json:"config"`
	SortOrder int      `json:"sort_order"`
	OwnerID   string   `json:"-"`
}

// New is the input to creating a card.
type New struct {
	Type   CardType
	Config Config
}

// Update is the input to updating a card. A card's Type can never change
// after creation — Service.Update accepts only a replacement Config.
type Update struct {
	Config Config
}

// validateShape checks Config's required/allowed field set for typ,
// without consulting any external access — that half (an
// account_id/category_id/tag_id must actually be visible to the caller)
// is Service.validateReferences, which needs the lookup interfaces.
func validateShape(typ CardType, c Config) error {
	if !typ.valid() {
		return ErrInvalidValue
	}

	switch typ {
	case CardTypeAccountStat:
		if c.AccountID == nil || strings.TrimSpace(*c.AccountID) == "" {
			return ErrInvalidValue
		}
		if c.CategoryID != nil || c.IncludeSubcategories != nil || c.TagID != nil || c.Range != nil || c.Unit != nil || c.Columns != nil || c.Title != nil || c.ShowRecurringPreview != nil {
			return ErrInvalidValue
		}
	case CardTypeQueryStat:
		if c.Unit != nil || c.Columns != nil || c.ShowRecurringPreview != nil {
			return ErrInvalidValue
		}
	case CardTypeEntryList:
		if c.Unit != nil {
			return ErrInvalidValue
		}
		if c.Columns != nil && (*c.Columns < MinColumns || *c.Columns > MaxColumns) {
			return ErrInvalidValue
		}
	case CardTypeBarChart:
		if c.Range != nil || c.Columns != nil {
			return ErrInvalidValue
		}
		if c.Unit == nil {
			return ErrInvalidValue
		}
		if *c.Unit != string(UnitMonth) && *c.Unit != string(UnitDay) {
			return ErrInvalidValue
		}
	case CardTypeLineChart:
		if c.CategoryID != nil || c.IncludeSubcategories != nil || c.TagID != nil || c.Range != nil || c.Unit != nil || c.Columns != nil {
			return ErrInvalidValue
		}
	}
	return nil
}
