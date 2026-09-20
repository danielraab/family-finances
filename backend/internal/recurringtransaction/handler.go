package recurringtransaction

import (
	"encoding/json"
	"net/http"
	"time"

	"at.draab/familyfinances/internal/auth"
)

// RenderError writes a JSON error response with the right status for a
// (possibly wrapped) sentinel error. package main wires httpapi.WriteError
// in so this package's sentinels are mapped to status codes in the one
// place that owns that table.
type RenderError func(w http.ResponseWriter, r *http.Request, err error)

// HandlerOptions configures NewHandler.
type HandlerOptions struct {
	// RenderError is required.
	RenderError RenderError
}

// Handler is the recurring transaction HTTP surface, mounted by
// internal/httpapi at /api/recurring-transactions.
type Handler struct {
	svc         *Service
	renderError RenderError
	mux         *http.ServeMux
}

// NewHandler builds the recurring transaction handler. opts.RenderError
// must be non-nil.
func NewHandler(svc *Service, opts HandlerOptions) *Handler {
	if opts.RenderError == nil {
		panic("recurringtransaction: HandlerOptions.RenderError is required")
	}
	h := &Handler{svc: svc, renderError: opts.RenderError, mux: http.NewServeMux()}

	h.mux.HandleFunc("GET /api/recurring-transactions", h.list)
	h.mux.HandleFunc("POST /api/recurring-transactions", h.create)
	h.mux.HandleFunc("GET /api/recurring-transactions/summary", h.summary)
	h.mux.HandleFunc("GET /api/recurring-transactions/preview", h.preview)
	h.mux.HandleFunc("GET /api/recurring-transactions/{id}", h.get)
	h.mux.HandleFunc("PATCH /api/recurring-transactions/{id}", h.update)
	h.mux.HandleFunc("DELETE /api/recurring-transactions/{id}", h.delete)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

type recurringCreateBody struct {
	AccountID     *string  `json:"account_id"`
	ToAccountID   *string  `json:"to_account_id"`
	Kind          *string  `json:"kind"`
	Title         *string  `json:"title"`
	Description   *string  `json:"description"`
	CategoryID    *string  `json:"category_id"`
	Counterparty  *string  `json:"counterparty"`
	Location      *string  `json:"location"`
	TagIDs        []string `json:"tag_ids"`
	Amount        *int64   `json:"amount"`
	IntervalUnit  *string  `json:"interval_unit"`
	IntervalCount *int     `json:"interval_count"`
	StartsOn      *Date    `json:"starts_on"`
	EndsOn        *Date    `json:"ends_on"`
}

// recurringUpdateBody deliberately carries neither kind nor to_account_id:
// both are immutable after creation, so decodeJSON's DisallowUnknownFields
// rejects an attempt to supply either.
type recurringUpdateBody struct {
	AccountID     *string      `json:"account_id"`
	Title         *string      `json:"title"`
	Description   *string      `json:"description"`
	CategoryID    *string      `json:"category_id"`
	Counterparty  *string      `json:"counterparty"`
	Location      *string      `json:"location"`
	TagIDs        *[]string    `json:"tag_ids"`
	Amount        *int64       `json:"amount"`
	IntervalUnit  *string      `json:"interval_unit"`
	IntervalCount *int         `json:"interval_count"`
	StartsOn      *Date        `json:"starts_on"`
	EndsOn        OptionalDate `json:"ends_on"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body recurringCreateBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	in := New{
		CategoryID:  body.CategoryID,
		TagIDs:      body.TagIDs,
		ToAccountID: body.ToAccountID,
		// An omitted kind is a transaction — what every recurring
		// transaction was before this field existed, so an older client's
		// body keeps working unchanged.
		Kind: KindTransaction,
	}
	if body.Kind != nil {
		in.Kind = Kind(*body.Kind)
	}
	if body.AccountID != nil {
		in.AccountID = *body.AccountID
	}
	if body.Title != nil {
		in.Title = *body.Title
	}
	if body.Description != nil {
		in.Description = *body.Description
	}
	if body.Counterparty != nil {
		in.Counterparty = *body.Counterparty
	}
	if body.Location != nil {
		in.Location = *body.Location
	}
	if body.Amount != nil {
		in.Amount = *body.Amount
	}
	if body.IntervalUnit != nil {
		in.IntervalUnit = Unit(*body.IntervalUnit)
	}
	if body.IntervalCount != nil {
		in.IntervalCount = *body.IntervalCount
	}
	if body.StartsOn != nil {
		in.StartsOn = *body.StartsOn
	}
	in.EndsOn = body.EndsOn

	rt, err := h.svc.Create(r.Context(), user.ID, in)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, rt)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	rt, err := h.svc.Get(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, rt)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body recurringUpdateBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	upd := Update{
		AccountID:     body.AccountID,
		Title:         body.Title,
		Description:   body.Description,
		CategoryID:    body.CategoryID,
		Counterparty:  body.Counterparty,
		Location:      body.Location,
		TagIDs:        body.TagIDs,
		Amount:        body.Amount,
		IntervalCount: body.IntervalCount,
		StartsOn:      body.StartsOn,
		EndsOn:        body.EndsOn,
	}
	if body.IntervalUnit != nil {
		u := Unit(*body.IntervalUnit)
		upd.IntervalUnit = &u
	}

	rt, err := h.svc.Update(r.Context(), user.ID, r.PathValue("id"), upd)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, rt)
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

// selfTransferMode resolves the include_self_transfer and
// self_transfer_both_legs query parameters. Anything other than "true" —
// including an absent parameter — is false, and both_legs without include
// is ignored (see ResolveSelfTransferMode).
func selfTransferMode(r *http.Request) SelfTransferMode {
	q := r.URL.Query()
	return ResolveSelfTransferMode(
		q.Get("include_self_transfer") == "true",
		q.Get("self_transfer_both_legs") == "true",
	)
}

// contentFilter reads the account, category and tag parameters every
// recurring read endpoint accepts, in one place so the listing, the
// summary and the preview all parse them identically. An invalid
// category_mode is not rejected here — Service.resolveFilter owns that
// check, for the listing and the preview alike.
func contentFilter(r *http.Request) Filter {
	q := r.URL.Query()
	f := Filter{
		AccountIDs:   q["account_id"],
		CategoryMode: CategoryMode(q.Get("category_mode")),
	}
	if v := q.Get("category_id"); v != "" {
		f.CategoryID = &v
	}
	if v := q.Get("tag_id"); v != "" {
		f.TagID = &v
	}
	return f
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	f := contentFilter(r)
	f.SelfTransfers = selfTransferMode(r)
	items, err := h.svc.List(r.Context(), user.ID, f)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	if items == nil {
		items = []RecurringTransaction{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	// The summary takes the same parameters as the listing, resolved the
	// same way, so the total it returns is always the total of the rows
	// the listing under those parameters would return.
	f := contentFilter(r)
	f.SelfTransfers = selfTransferMode(r)
	sum, err := h.svc.Summary(r.Context(), user.ID, f)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// previewResponse wraps Preview's result the way every other list endpoint
// in this codebase wraps its items, under an "items" key.
type previewResponse struct {
	Items []PreviewItem `json:"items"`
}

func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	q := r.URL.Query()
	toStr := q.Get("to")
	if toStr == "" {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	content := contentFilter(r)
	f := PreviewFilter{
		AccountIDs:   content.AccountIDs,
		CategoryID:   content.CategoryID,
		CategoryMode: content.CategoryMode,
		TagID:        content.TagID,
		To:           to,
	}

	items, err := h.svc.Preview(r.Context(), user.ID, f)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	if items == nil {
		items = []PreviewItem{}
	}
	writeJSON(w, http.StatusOK, previewResponse{Items: items})
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
