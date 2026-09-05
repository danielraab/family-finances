package category

import (
	"encoding/json"
	"net/http"

	"at.draab/familyfinances/internal/auth"
)

// RenderError writes a JSON error response with the right status for a
// (possibly wrapped) sentinel error. package main wires httpapi.WriteError in
// so category's sentinels are mapped to status codes in the one place that
// owns that table.
type RenderError func(w http.ResponseWriter, r *http.Request, err error)

// HandlerOptions configures NewHandler.
type HandlerOptions struct {
	// RenderError is required.
	RenderError RenderError
}

// Handler is the category HTTP surface, mounted by internal/httpapi at
// /api/categories.
type Handler struct {
	svc         *Service
	renderError RenderError
	mux         *http.ServeMux
}

// NewHandler builds the category handler. opts.RenderError must be non-nil.
func NewHandler(svc *Service, opts HandlerOptions) *Handler {
	if opts.RenderError == nil {
		panic("category: HandlerOptions.RenderError is required")
	}
	h := &Handler{svc: svc, renderError: opts.RenderError, mux: http.NewServeMux()}

	h.mux.HandleFunc("GET /api/categories", h.list)
	h.mux.HandleFunc("POST /api/categories", h.create)
	h.mux.HandleFunc("PATCH /api/categories/{id}", h.update)
	h.mux.HandleFunc("DELETE /api/categories/{id}", h.delete)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

type categoryBody struct {
	Name     *string    `json:"name"`
	ParentID OptionalID `json:"parent_id"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserFromContext(r.Context()); !ok {
		writeUnauthorized(w)
		return
	}
	cats, err := h.svc.List(r.Context())
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	if cats == nil {
		cats = []Category{}
	}
	writeJSON(w, http.StatusOK, cats)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r); !ok {
		return
	}
	var body categoryBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	in := New{ParentID: body.ParentID.Value}
	if body.Name != nil {
		in.Name = *body.Name
	}
	cat, err := h.svc.Create(r.Context(), in)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, cat)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r); !ok {
		return
	}
	var body categoryBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	cat, err := h.svc.Update(r.Context(), r.PathValue("id"), Update{Name: body.Name, ParentID: body.ParentID})
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, cat)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireAdmin(w, r); !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		h.renderError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// requireAdmin returns the authenticated admin user, or writes 401/403 and
// returns ok=false — the same gate internal/auth established.
func requireAdmin(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return auth.User{}, false
	}
	if !user.IsAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return auth.User{}, false
	}
	return user, true
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
