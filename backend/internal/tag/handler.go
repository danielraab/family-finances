package tag

import (
	"encoding/json"
	"net/http"

	"at.draab/familyfinances/internal/auth"
)

// RenderError writes a JSON error response with the right status for a
// (possibly wrapped) sentinel error. package main wires httpapi.WriteError in
// so tag's sentinels are mapped to status codes in the one place that owns
// that table.
type RenderError func(w http.ResponseWriter, r *http.Request, err error)

// HandlerOptions configures NewHandler.
type HandlerOptions struct {
	// RenderError is required.
	RenderError RenderError
}

// Handler is the tag HTTP surface, mounted by internal/httpapi at /api/tags.
type Handler struct {
	svc         *Service
	renderError RenderError
	mux         *http.ServeMux
}

// NewHandler builds the tag handler. opts.RenderError must be non-nil.
func NewHandler(svc *Service, opts HandlerOptions) *Handler {
	if opts.RenderError == nil {
		panic("tag: HandlerOptions.RenderError is required")
	}
	h := &Handler{svc: svc, renderError: opts.RenderError, mux: http.NewServeMux()}

	h.mux.HandleFunc("GET /api/tags", h.list)
	h.mux.HandleFunc("POST /api/tags", h.create)
	h.mux.HandleFunc("GET /api/tags/{id}", h.get)
	h.mux.HandleFunc("PATCH /api/tags/{id}", h.update)
	h.mux.HandleFunc("DELETE /api/tags/{id}", h.delete)
	h.mux.HandleFunc("POST /api/tags/{id}/disable", h.disable)
	h.mux.HandleFunc("POST /api/tags/{id}/enable", h.enable)

	h.mux.HandleFunc("GET /api/tags/{id}/shares", h.listShares)
	h.mux.HandleFunc("POST /api/tags/{id}/shares", h.inviteShare)
	h.mux.HandleFunc("PATCH /api/tags/{id}/shares/{userId}", h.updateShare)
	h.mux.HandleFunc("DELETE /api/tags/{id}/shares/{userId}", h.revokeShare)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

type tagBody struct {
	Name string `json:"name"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	tags, err := h.svc.List(r.Context(), user.ID)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	if tags == nil {
		tags = []Tag{}
	}
	writeJSON(w, http.StatusOK, tags)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	t, err := h.svc.Get(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body tagBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	t, err := h.svc.Create(r.Context(), user.ID, body.Name)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body tagBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	t, err := h.svc.Update(r.Context(), user.ID, r.PathValue("id"), body.Name)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
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
	t, err := h.svc.Disable(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) enable(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	t, err := h.svc.Enable(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// --- sharing -----------------------------------------------------------

func (h *Handler) listShares(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	shares, err := h.svc.ListShares(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	if shares == nil {
		shares = []TagShare{}
	}
	writeJSON(w, http.StatusOK, shares)
}

type tagShareInviteBody struct {
	Email      *string `json:"email"`
	Permission *string `json:"permission"`
}

type tagShareInviteResponse struct {
	Matched       bool      `json:"matched"`
	InviteAllowed bool      `json:"invite_allowed"`
	Share         *TagShare `json:"share,omitempty"`
}

// callerDisplayName is what a share-notification email's "X shared this
// with you" line names the granter as — the caller's own display name,
// falling back to their email when unset.
func callerDisplayName(u auth.User) string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Email
}

func (h *Handler) inviteShare(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body tagShareInviteBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	var email, permission string
	if body.Email != nil {
		email = *body.Email
	}
	if body.Permission != nil {
		permission = *body.Permission
	}
	result, err := h.svc.InviteShare(r.Context(), user.ID, callerDisplayName(user), r.PathValue("id"), email, Permission(permission))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	if !result.Matched {
		writeJSON(w, http.StatusOK, tagShareInviteResponse{Matched: false, InviteAllowed: result.InviteAllowed})
		return
	}
	writeJSON(w, http.StatusCreated, tagShareInviteResponse{Matched: true, Share: result.Share})
}

type tagSharePermissionBody struct {
	Permission *string `json:"permission"`
}

func (h *Handler) updateShare(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body tagSharePermissionBody
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidValue)
		return
	}
	var permission string
	if body.Permission != nil {
		permission = *body.Permission
	}
	share, err := h.svc.UpdateSharePermission(r.Context(), user.ID, r.PathValue("id"), r.PathValue("userId"), Permission(permission))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, share)
}

func (h *Handler) revokeShare(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	if err := h.svc.RevokeShare(r.Context(), user.ID, r.PathValue("id"), r.PathValue("userId")); err != nil {
		h.renderError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
