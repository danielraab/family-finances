package dashboard

import (
	"encoding/json"
	"net/http"

	"at.draab/familyfinances/internal/auth"
)

// RenderError writes a JSON error response with the right status for a
// (possibly wrapped) sentinel error. package main wires httpapi.WriteError
// in so dashboard's sentinels are mapped to status codes in the one place
// that owns that table.
type RenderError func(w http.ResponseWriter, r *http.Request, err error)

// HandlerOptions configures NewHandler.
type HandlerOptions struct {
	// RenderError is required.
	RenderError RenderError
}

// Handler is the dashboard HTTP surface, mounted by internal/httpapi at
// /api/dashboard/cards.
type Handler struct {
	svc         *Service
	renderError RenderError
	mux         *http.ServeMux
}

// NewHandler builds the dashboard handler. opts.RenderError must be
// non-nil.
func NewHandler(svc *Service, opts HandlerOptions) *Handler {
	if opts.RenderError == nil {
		panic("dashboard: HandlerOptions.RenderError is required")
	}
	h := &Handler{svc: svc, renderError: opts.RenderError, mux: http.NewServeMux()}

	h.mux.HandleFunc("GET /api/dashboard/cards", h.list)
	h.mux.HandleFunc("POST /api/dashboard/cards", h.create)
	h.mux.HandleFunc("PATCH /api/dashboard/cards/{id}", h.update)
	h.mux.HandleFunc("DELETE /api/dashboard/cards/{id}", h.delete)
	h.mux.HandleFunc("POST /api/dashboard/cards/{id}/move-up", h.moveUp)
	h.mux.HandleFunc("POST /api/dashboard/cards/{id}/move-down", h.moveDown)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

type cardCreateBody struct {
	Type   *string `json:"type"`
	Config Config  `json:"config"`
}

type cardUpdateBody struct {
	// Type is accepted but ignored — a card's type is immutable after
	// creation (see Service.Update's doc comment).
	Type   *string `json:"type"`
	Config Config  `json:"config"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	cards, err := h.svc.List(r.Context(), user.ID)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	if cards == nil {
		cards = []Card{}
	}
	writeJSON(w, http.StatusOK, cards)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body cardCreateBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	var typ CardType
	if body.Type != nil {
		typ = CardType(*body.Type)
	}
	card, err := h.svc.Create(r.Context(), user.ID, New{Type: typ, Config: body.Config})
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, card)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body cardUpdateBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	card, err := h.svc.Update(r.Context(), user.ID, r.PathValue("id"), Update{Config: body.Config})
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, card)
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

func (h *Handler) moveUp(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	card, err := h.svc.MoveUp(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, card)
}

func (h *Handler) moveDown(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	card, err := h.svc.MoveDown(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, card)
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
