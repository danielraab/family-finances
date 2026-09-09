package account

import (
	"encoding/json"
	"net/http"

	"at.draab/familyfinances/internal/auth"
)

// RenderError writes a JSON error response with the right status for a
// (possibly wrapped) sentinel error. package main wires httpapi.WriteError in
// so account's sentinels are mapped to status codes in the one place that
// owns that table.
type RenderError func(w http.ResponseWriter, r *http.Request, err error)

// HandlerOptions configures NewHandler.
type HandlerOptions struct {
	// RenderError is required.
	RenderError RenderError
}

// Handler is the account HTTP surface, mounted by internal/httpapi at
// /api/accounts and /api/account-types.
type Handler struct {
	svc         *Service
	renderError RenderError
	mux         *http.ServeMux
}

// NewHandler builds the account handler. opts.RenderError must be non-nil.
func NewHandler(svc *Service, opts HandlerOptions) *Handler {
	if opts.RenderError == nil {
		panic("account: HandlerOptions.RenderError is required")
	}
	h := &Handler{svc: svc, renderError: opts.RenderError, mux: http.NewServeMux()}

	h.mux.HandleFunc("GET /api/accounts", h.list)
	h.mux.HandleFunc("POST /api/accounts", h.create)
	h.mux.HandleFunc("GET /api/accounts/{id}", h.get)
	h.mux.HandleFunc("PATCH /api/accounts/{id}", h.update)
	h.mux.HandleFunc("DELETE /api/accounts/{id}", h.delete)
	h.mux.HandleFunc("POST /api/accounts/{id}/disable", h.disable)
	h.mux.HandleFunc("POST /api/accounts/{id}/enable", h.enable)

	h.mux.HandleFunc("GET /api/account-types", h.listTypes)
	h.mux.HandleFunc("POST /api/account-types", h.createType)
	h.mux.HandleFunc("PATCH /api/account-types/{id}", h.updateType)
	h.mux.HandleFunc("DELETE /api/account-types/{id}", h.deleteType)
	h.mux.HandleFunc("POST /api/account-types/{id}/disable", h.disableType)
	h.mux.HandleFunc("POST /api/account-types/{id}/enable", h.enableType)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

// --- accounts ------------------------------------------------------------

type accountBody struct {
	Title              *string      `json:"title"`
	Description        *string      `json:"description"`
	Icon               *string      `json:"icon"`
	Color              *string      `json:"color"`
	TypeID             *string      `json:"type_id"`
	Currency           *string      `json:"currency"`
	FinancialInstitute *string      `json:"financial_institute"`
	OpeningDate        *Date        `json:"opening_date"`
	ClosingDate        OptionalDate `json:"closing_date"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	accounts, err := h.svc.List(r.Context(), user.ID)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	if accounts == nil {
		accounts = []Account{}
	}
	writeJSON(w, http.StatusOK, accounts)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body accountBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	in := New{ClosingDate: body.ClosingDate.Value}
	if body.Title != nil {
		in.Title = *body.Title
	}
	if body.Description != nil {
		in.Description = *body.Description
	}
	if body.Icon != nil {
		in.Icon = *body.Icon
	}
	if body.Color != nil {
		in.Color = *body.Color
	}
	if body.TypeID != nil {
		in.TypeID = *body.TypeID
	}
	if body.Currency != nil {
		in.Currency = *body.Currency
	}
	if body.FinancialInstitute != nil {
		in.FinancialInstitute = *body.FinancialInstitute
	}
	if body.OpeningDate != nil {
		in.OpeningDate = *body.OpeningDate
	}

	acc, err := h.svc.Create(r.Context(), user.ID, in)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, acc)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	acc, err := h.svc.Get(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body accountBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	upd := Update{
		Title:              body.Title,
		Description:        body.Description,
		Icon:               body.Icon,
		Color:              body.Color,
		TypeID:             body.TypeID,
		Currency:           body.Currency,
		FinancialInstitute: body.FinancialInstitute,
		OpeningDate:        body.OpeningDate,
		ClosingDate:        body.ClosingDate,
	}
	acc, err := h.svc.Update(r.Context(), user.ID, r.PathValue("id"), upd)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, acc)
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

func (h *Handler) disable(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	acc, err := h.svc.Disable(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

func (h *Handler) enable(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	acc, err := h.svc.Enable(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

// --- account types ---------------------------------------------------

func (h *Handler) listTypes(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	types, err := h.svc.ListTypes(r.Context(), user.ID)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	if types == nil {
		types = []Type{}
	}
	writeJSON(w, http.StatusOK, types)
}

type accountTypeBody struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (h *Handler) createType(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body accountTypeBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	t, err := h.svc.CreateType(r.Context(), user.ID, body.Title, body.Description)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) updateType(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body accountTypeBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	t, err := h.svc.UpdateType(r.Context(), user.ID, r.PathValue("id"), body.Title, body.Description)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteType(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	if err := h.svc.DeleteType(r.Context(), user.ID, r.PathValue("id")); err != nil {
		h.renderError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) disableType(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	t, err := h.svc.DisableType(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) enableType(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	t, err := h.svc.EnableType(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
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
