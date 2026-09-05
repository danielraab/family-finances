package settings

import (
	"encoding/json"
	"net/http"

	"at.draab/familyfinances/internal/auth"
)

// RenderError writes a JSON error response with the right status for a
// (possibly wrapped) sentinel error. package main wires httpapi.WriteError in
// so settings' sentinels are mapped to status codes in the one place that
// owns that table.
type RenderError func(w http.ResponseWriter, r *http.Request, err error)

// HandlerOptions configures NewHandler.
type HandlerOptions struct {
	// RenderError is required.
	RenderError RenderError
}

// Handler is the settings HTTP surface, mounted by internal/httpapi at
// /api/settings.
type Handler struct {
	svc         *Service
	renderError RenderError
	mux         *http.ServeMux
}

// NewHandler builds the settings handler. opts.RenderError must be non-nil.
func NewHandler(svc *Service, opts HandlerOptions) *Handler {
	if opts.RenderError == nil {
		panic("settings: HandlerOptions.RenderError is required")
	}
	h := &Handler{svc: svc, renderError: opts.RenderError, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /api/settings", h.get)
	h.mux.HandleFunc("PUT /api/settings", h.update)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	s, err := h.svc.Get(r.Context(), user.ID)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body struct {
		Language               *string `json:"language"`
		Timezone               *string `json:"timezone"`
		DefaultCurrency        *string `json:"default_currency"`
		DisplayedDecimalPlaces *int    `json:"displayed_decimal_places"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	s, err := h.svc.Update(r.Context(), user.ID, Update{
		Language:               body.Language,
		Timezone:               body.Timezone,
		DefaultCurrency:        body.DefaultCurrency,
		DisplayedDecimalPlaces: body.DisplayedDecimalPlaces,
	})
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
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
