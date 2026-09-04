package auth

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// RenderError writes a JSON error response with the right status for a
// (possibly wrapped) sentinel error. package main wires httpapi.WriteError in
// so auth's sentinels are mapped to status codes in the one place that owns
// that table.
type RenderError func(w http.ResponseWriter, r *http.Request, err error)

// HandlerOptions configures NewHandler.
type HandlerOptions struct {
	// RenderError is required.
	RenderError RenderError
	// CookieSecure sets the Secure attribute on the session cookie.
	CookieSecure bool
}

// Handler is the auth HTTP surface, mounted by internal/httpapi at
// /api/auth/. It maps requests to Service calls and Service results (or
// sentinel errors) to responses; it holds no logic of its own.
type Handler struct {
	svc          *Service
	renderError  RenderError
	cookieSecure bool
	mux          *http.ServeMux
}

// NewHandler builds the auth handler. opts.RenderError must be non-nil.
func NewHandler(svc *Service, opts HandlerOptions) *Handler {
	if opts.RenderError == nil {
		panic("auth: HandlerOptions.RenderError is required")
	}
	h := &Handler{
		svc:          svc,
		renderError:  opts.RenderError,
		cookieSecure: opts.CookieSecure,
		mux:          http.NewServeMux(),
	}

	h.mux.HandleFunc("POST /api/auth/email/start", h.emailStart)
	h.mux.HandleFunc("GET /api/auth/email/callback", h.emailCallback)
	h.mux.HandleFunc("GET /api/auth/oidc/start", h.oidcStart)
	h.mux.HandleFunc("GET /api/auth/oidc/callback", h.oidcCallback)
	h.mux.HandleFunc("GET /api/auth/config", h.config)
	h.mux.HandleFunc("GET /api/auth/me", h.me)
	h.mux.HandleFunc("POST /api/auth/logout", h.logout)
	h.mux.HandleFunc("POST /api/auth/invites", h.createInvite)
	h.mux.HandleFunc("GET /api/auth/invites", h.listInvites)
	h.mux.HandleFunc("GET /api/auth/invites/accept", h.acceptInvite)
	h.mux.HandleFunc("GET /api/auth/users", h.listUsers)
	h.mux.HandleFunc("POST /api/auth/users/{id}/disable", h.disableUser)
	h.mux.HandleFunc("POST /api/auth/users/{id}/enable", h.enableUser)
	h.mux.HandleFunc("DELETE /api/auth/users/{id}", h.deleteUser)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

// --- magic link ---------------------------------------------------------

func (h *Handler) emailStart(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidEmail)
		return
	}

	// Always 200, regardless of whether a mail goes out — no account
	// enumeration. An internal failure is logged by the service's caller; the
	// response is still 200.
	if err := h.svc.StartEmailLogin(r.Context(), body.Email); err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) emailCallback(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		h.renderError(w, r, ErrTokenInvalid)
		return
	}
	jsonClient := wantsJSON(r)
	user, session, err := h.svc.CompleteEmailLogin(r.Context(), token, currentUserID(r), sessionContext(r, jsonClient))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	h.completeSignIn(w, r, user, session, jsonClient, "")
}

// --- OIDC -------------------------------------------------------------

func (h *Handler) oidcStart(w http.ResponseWriter, r *http.Request) {
	redirectURL, err := h.svc.StartOIDC(r.Context(), safeRelativePath(r.URL.Query().Get("redirect_to")))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) oidcCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		h.renderError(w, r, ErrTokenInvalid)
		return
	}
	state, code := q.Get("state"), q.Get("code")
	if state == "" || code == "" {
		h.renderError(w, r, ErrTokenInvalid)
		return
	}
	jsonClient := wantsJSON(r)
	user, session, returnTo, err := h.svc.CompleteOIDC(r.Context(), state, code, currentUserID(r), sessionContext(r, jsonClient))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	h.completeSignIn(w, r, user, session, jsonClient, returnTo)
}

// --- session endpoints ------------------------------------------------

// config reports which sign-in methods the client should offer. It needs no
// authentication — the login page is viewed by anonymous visitors. The only
// field today is oidc: an object with the button label and the start path when
// an OIDC provider is configured, or null. It exposes no provider secrets.
func (h *Handler) config(w http.ResponseWriter, r *http.Request) {
	type oidcLogin struct {
		Label     string `json:"label"`
		StartPath string `json:"start_path"`
	}
	body := struct {
		OIDC *oidcLogin `json:"oidc"`
	}{}
	if label, ok := h.svc.OIDCLogin(); ok {
		body.OIDC = &oidcLogin{Label: label, StartPath: "/api/auth/oidc/start"}
	}
	writeJSON(w, http.StatusOK, body)
}

// meResponse embeds User and adds the raw (unresolved) language preference —
// see internal/settings' design note on why /api/auth/me carries the raw
// value while GET /api/settings carries the resolved one.
type meResponse struct {
	User
	Language *string `json:"language"`
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	writeJSON(w, http.StatusOK, meResponse{
		User:     user,
		Language: h.svc.UserLanguage(r.Context(), user.ID),
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if _, ok := UserFromContext(r.Context()); !ok {
		writeUnauthorized(w)
		return
	}
	if err := h.svc.Logout(r.Context(), tokenFromRequest(r)); err != nil {
		h.renderError(w, r, err)
		return
	}
	if _, err := r.Cookie(CookieName); err == nil {
		h.clearCookie(w)
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- invites --------------------------------------------------------

func (h *Handler) createInvite(w http.ResponseWriter, r *http.Request) {
	inviter, ok := UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.renderError(w, r, ErrInvalidEmail)
		return
	}
	inv, err := h.svc.CreateInvite(r.Context(), inviter.ID, body.Email)
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, InviteInfo{
		ID:    inv.ID,
		Email: inv.Email,
		InvitedBy: InviteInviter{
			ID:          inviter.ID,
			Email:       inviter.Email,
			DisplayName: inviter.DisplayName,
		},
		CreatedAt:  inv.CreatedAt,
		ExpiresAt:  inv.ExpiresAt,
		AcceptedAt: inv.AcceptedAt,
	})
}

func (h *Handler) listInvites(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	invites, err := h.svc.ListInvites(r.Context())
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, invites)
}

// --- admin: users ----------------------------------------------------

// requireAdmin returns the authenticated admin user, or writes 401/403 and
// returns ok=false. The first real use of is_admin as an authorization
// boundary in the backend.
func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request) (User, bool) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return User{}, false
	}
	if !user.IsAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return User{}, false
	}
	return user, true
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	users, err := h.svc.ListUsers(r.Context())
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	out := make([]AdminUser, len(users))
	for i, u := range users {
		out[i] = toAdminUser(u)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) disableUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	u, err := h.svc.DisableUser(r.Context(), r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAdminUser(u))
}

func (h *Handler) enableUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	u, err := h.svc.EnableUser(r.Context(), r.PathValue("id"))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAdminUser(u))
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	if err := h.svc.SoftDeleteUser(r.Context(), r.PathValue("id")); err != nil {
		h.renderError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		h.renderError(w, r, ErrInviteInvalid)
		return
	}
	jsonClient := wantsJSON(r)
	user, session, err := h.svc.AcceptInvite(r.Context(), token, currentUserID(r), sessionContext(r, jsonClient))
	if err != nil {
		h.renderError(w, r, err)
		return
	}
	h.completeSignIn(w, r, user, session, jsonClient, "")
}

// --- shared helpers -------------------------------------------------

// completeSignIn finishes a sign-in flow: a JSON client gets the token and the
// user in the body; a browser gets the ff_session cookie and a redirect to a
// validated in-app path.
func (h *Handler) completeSignIn(w http.ResponseWriter, r *http.Request, user User, session string, jsonClient bool, returnTo string) {
	if jsonClient {
		writeJSON(w, http.StatusOK, map[string]any{
			"session_token": session,
			"user":          user,
		})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    session,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, safeRelativePathOrRoot(returnTo), http.StatusFound)
}

func (h *Handler) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// wantsJSON is the browser-vs-API client discriminator: an explicit client=api
// query marker, or an Accept header asking for JSON.
func wantsJSON(r *http.Request) bool {
	if r.URL.Query().Get("client") == "api" {
		return true
	}
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func sessionContext(r *http.Request, jsonClient bool) SessionContext {
	client := ClientWeb
	if jsonClient {
		client = ClientAPI
	}
	return SessionContext{Client: client, UserAgent: r.UserAgent(), IP: clientIP(r)}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// currentUserID is the id of the already-authenticated user on the request, or
// "" — the seam that turns any sign-in flow into an explicit account link.
func currentUserID(r *http.Request) string {
	if u, ok := UserFromContext(r.Context()); ok {
		return u.ID
	}
	return ""
}

func tokenFromRequest(r *http.Request) string {
	const prefix = "Bearer "
	if h := r.Header.Get("Authorization"); len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	if c, err := r.Cookie(CookieName); err == nil {
		return c.Value
	}
	return ""
}

// safeRelativePath returns target only if it is a same-origin relative path
// ("/x", no scheme, no host, no "//" prefix); otherwise "".
func safeRelativePath(target string) string {
	if target == "" || !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") {
		return ""
	}
	u, err := url.Parse(target)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return ""
	}
	return target
}

func safeRelativePathOrRoot(target string) string {
	if p := safeRelativePath(target); p != "" {
		return p
	}
	return "/"
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
