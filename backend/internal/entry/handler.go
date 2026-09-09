package entry

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"at.draab/familyfinances/internal/auth"
)

// RenderError writes a JSON error response with the right status for a
// (possibly wrapped) sentinel error. package main wires httpapi.WriteError in
// so entry's sentinels are mapped to status codes in the one place that
// owns that table.
type RenderError func(w http.ResponseWriter, r *http.Request, err error)

// HandlerOptions configures NewHandler.
type HandlerOptions struct {
	// RenderError is required.
	RenderError RenderError
}

// Handler is the entry HTTP surface, mounted by internal/httpapi at
// /api/entries and (for the balance read) GET /api/accounts/{id}/balance.
type Handler struct {
	svc         *Service
	renderError RenderError
	mux         *http.ServeMux
}

// NewHandler builds the entry handler. opts.RenderError must be non-nil.
func NewHandler(svc *Service, opts HandlerOptions) *Handler {
	if opts.RenderError == nil {
		panic("entry: HandlerOptions.RenderError is required")
	}
	h := &Handler{svc: svc, renderError: opts.RenderError, mux: http.NewServeMux()}

	h.mux.HandleFunc("GET /api/entries", h.list)
	h.mux.HandleFunc("POST /api/entries", h.create)
	h.mux.HandleFunc("GET /api/entries/summary", h.summary)
	h.mux.HandleFunc("GET /api/entries/flow-summary", h.flowSummary)
	h.mux.HandleFunc("GET /api/entries/balance-series", h.balanceSeries)
	h.mux.HandleFunc("GET /api/entries/{id}", h.get)
	h.mux.HandleFunc("PATCH /api/entries/{id}", h.update)
	h.mux.HandleFunc("DELETE /api/entries/{id}", h.delete)
	h.mux.HandleFunc("GET /api/accounts/{id}/balance", h.balance)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

type entryCreateBody struct {
	AccountID        *string    `json:"account_id"`
	Kind             *string    `json:"kind"`
	Amount           *int64     `json:"amount"`
	Balance          *int64     `json:"balance"`
	BookingTimestamp *time.Time `json:"booking_timestamp"`
	Title            *string    `json:"title"`
	Description      *string    `json:"description"`
	CategoryID       *string    `json:"category_id"`
	TagIDs           []string   `json:"tag_ids"`
}

type entryUpdateBody struct {
	AccountID        *string    `json:"account_id"`
	Amount           *int64     `json:"amount"`
	Balance          *int64     `json:"balance"`
	BookingTimestamp *time.Time `json:"booking_timestamp"`
	Title            *string    `json:"title"`
	Description      *string    `json:"description"`
	CategoryID       OptionalID `json:"category_id"`
	TagIDs           *[]string  `json:"tag_ids"`
}

type entryPage struct {
	Items      []Entry `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body entryCreateBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	in := New{CategoryID: body.CategoryID, TagIDs: body.TagIDs, Amount: body.Amount, Balance: body.Balance}
	if body.AccountID != nil {
		in.AccountID = *body.AccountID
	}
	if body.Kind != nil {
		in.Kind = Kind(*body.Kind)
	}
	if body.BookingTimestamp != nil {
		in.BookingTimestamp = *body.BookingTimestamp
	}
	if body.Title != nil {
		in.Title = *body.Title
	}
	if body.Description != nil {
		in.Description = *body.Description
	}

	e, err := h.svc.Create(r.Context(), user.ID, in)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	e, err := h.svc.Get(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body entryUpdateBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	upd := Update{
		AccountID:        body.AccountID,
		Amount:           body.Amount,
		Balance:          body.Balance,
		BookingTimestamp: body.BookingTimestamp,
		Title:            body.Title,
		Description:      body.Description,
		CategoryID:       body.CategoryID,
		TagIDs:           body.TagIDs,
	}
	e, err := h.svc.Update(r.Context(), user.ID, r.PathValue("id"), upd)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	if err := h.svc.Delete(r.Context(), user.ID, r.PathValue("id")); err != nil {
		h.renderError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseCommonFilter parses the account_id, category_id/category_mode,
// tag_id, from, to, and q query parameters shared by GET /api/entries and
// GET /api/entries/summary. list additionally parses sort, dir, kind,
// limit, and after; summary uses no other parameters — it always sums
// kind: transaction entries regardless of what the caller passes.
func parseCommonFilter(q url.Values) (Filter, error) {
	f := Filter{
		AccountIDs: q["account_id"],
		Query:      q.Get("q"),
	}
	if v := q.Get("category_id"); v != "" {
		f.CategoryID = &v
	}
	if v := q.Get("category_mode"); v != "" {
		f.CategoryMode = CategoryMode(v)
	}
	if v := q.Get("tag_id"); v != "" {
		f.TagID = &v
	}
	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return Filter{}, ErrInvalidValue
		}
		f.From = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return Filter{}, ErrInvalidValue
		}
		f.To = &t
	}
	return f, nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	q := r.URL.Query()

	f, err := parseCommonFilter(q)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	f.Sort = SortField(q.Get("sort"))
	f.Dir = SortDir(q.Get("dir"))
	if v := q.Get("kind"); v != "" {
		k := Kind(v)
		f.Kind = &k
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			h.renderError(w, r, ErrInvalidValue)
			return
		}
		f.Limit = n
	}
	if v := q.Get("after"); v != "" {
		c, err := decodeCursor(v)
		if err != nil {
			h.renderError(w, r, ErrInvalidValue)
			return
		}
		f.After = c
	}

	items, next, err := h.svc.List(r.Context(), user.ID, f)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	page := entryPage{Items: items}
	if next != nil {
		s := encodeCursor(*next)
		page.NextCursor = &s
	}
	if page.Items == nil {
		page.Items = []Entry{}
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	f, err := parseCommonFilter(r.URL.Query())
	if err != nil {
		h.renderError(w, r, err)
		return
	}

	sum, err := h.svc.Sum(r.Context(), user.ID, f)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

type flowSummaryResponse struct {
	Buckets []FlowBucket `json:"buckets"`
}

func (h *Handler) flowSummary(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	q := r.URL.Query()

	f := FlowFilter{
		AccountIDs: q["account_id"],
		Unit:       FlowUnit(q.Get("unit")),
	}
	year, err := strconv.Atoi(q.Get("year"))
	if err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	f.Year = year
	if v := q.Get("month"); v != "" {
		month, err := strconv.Atoi(v)
		if err != nil {
			h.renderError(w, r, ErrInvalidValue)
			return
		}
		f.Month = month
	}

	buckets, err := h.svc.FlowSummary(r.Context(), user.ID, f)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, flowSummaryResponse{Buckets: buckets})
}

type balanceSeriesResponse struct {
	Points []BalancePoint `json:"points"`
}

func (h *Handler) balanceSeries(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	q := r.URL.Query()

	// A balance is defined only by the account and the instant. A
	// category-, tag-, text-, or time-range filter would drop the
	// balance_adjustment anchors the running sum depends on, so it is
	// rejected outright rather than silently changing what the line means.
	for _, k := range []string{"category_id", "category_mode", "tag_id", "from", "to", "q"} {
		if q.Has(k) {
			h.renderError(w, r, ErrInvalidValue)
			return
		}
	}

	f := BalanceFilter{
		AccountIDs: q["account_id"],
		Unit:       FlowUnit(q.Get("unit")),
	}
	year, err := strconv.Atoi(q.Get("year"))
	if err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	f.Year = year
	month, err := strconv.Atoi(q.Get("month"))
	if err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	f.Month = month

	points, err := h.svc.BalanceSeries(r.Context(), user.ID, f)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, balanceSeriesResponse{Points: points})
}

func (h *Handler) balance(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	asOf := time.Now()
	if v := r.URL.Query().Get("as_of"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			h.renderError(w, r, ErrInvalidValue)
			return
		}
		asOf = t
	}
	balance, err := h.svc.Balance(r.Context(), user.ID, r.PathValue("id"), asOf)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"balance": balance})
}

// encodeCursor/decodeCursor make a Cursor opaque to the client — a
// base64url-encoded JSON blob, per design.md.
func encodeCursor(c Cursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (*Cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var c Cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeUnauthorized(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}
