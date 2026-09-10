package account

import (
	"context"
	"log/slog"
	"strings"
)

// UserLookup is the narrow view of internal/auth that account needs for
// sharing: resolving a share-invite email to an existing user (without
// exposing a directory of registered users — see design.md), and reporting
// whether the instance currently allows sending a new application invite.
// *auth.Service satisfies this structurally (its existing InviteEnabled
// method matches this interface's method of the same name; ByEmail is
// added to auth.Service for this purpose).
type UserLookup interface {
	// ByEmail resolves email to an existing, non-disabled, non-soft-deleted
	// user's id and display name. ok is false when no such user exists —
	// never an error, so a normal "not registered" outcome doesn't need
	// sentinel-matching across the package boundary.
	ByEmail(ctx context.Context, email string) (userID, displayName string, ok bool, err error)
	// InviteEnabled reports whether an authenticated user may currently
	// send a new application invite (mirrors AUTH_SIGNUP_ENABLED /
	// AUTH_INVITE_ENABLED).
	InviteEnabled() bool
}

// Mailer sends the one transactional email account produces. internal/mailer
// implements it over SMTP, alongside auth.Mailer. permission is a plain
// string (not Permission) so a leaf adapter package like internal/mailer
// can satisfy this interface structurally without importing internal/
// account — the same shape auth.Mailer already uses for its own strings.
type Mailer interface {
	// SendAccountShare emails addr that accountTitle has been shared with
	// them by granterName at permission, with a link into the application.
	SendAccountShare(ctx context.Context, addr, accountTitle, granterName, permission, link string) error
}

// Service is the account use-case layer. It depends on the Store interface
// plus the optional UserLookup/Mailer side-effect interfaces sharing uses.
type Service struct {
	store   Store
	users   UserLookup // nil until SetUserLookup is called
	mailer  Mailer     // nil disables the share-notification email
	baseURL string
}

// Option customizes a Service at construction.
type Option func(*Service)

// WithMailer wires the share-notification email sender.
func WithMailer(m Mailer) Option { return func(s *Service) { s.mailer = m } }

// WithBaseURL sets the application base URL used to build a share
// notification's link (typically AUTH_BASE_URL — the same value auth's
// magic-link/invite emails already use).
func WithBaseURL(url string) Option { return func(s *Service) { s.baseURL = url } }

// NewService builds the account service.
func NewService(store Store, opts ...Option) *Service {
	s := &Service{store: store}
	for _, o := range opts {
		o(s)
	}
	return s
}

// SetUserLookup wires the email-lookup/invite-eligibility source after
// construction. account.Service and auth.Service have a wiring cycle
// (auth.NewService needs account.Service as a NewUserHook; account.Service
// needs auth.Service as its UserLookup) that a constructor option can't
// resolve — package main breaks it by building both, then calling this
// once auth.Service exists. A nil users (never called) makes every sharing
// email-invite fail closed (ErrInvalidValue), never panic.
func (s *Service) SetUserLookup(u UserLookup) { s.users = u }

// Create validates in and creates an account owned by ownerID — the real
// owner, always the caller.
func (s *Service) Create(ctx context.Context, ownerID string, in New) (Account, error) {
	if err := validateNew(in); err != nil {
		return Account{}, err
	}
	in.Type = strings.TrimSpace(in.Type)
	acc, err := s.store.Create(ctx, ownerID, in)
	if err != nil {
		return Account{}, err
	}
	acc.Permission = PermissionOwner
	return acc, nil
}

// Get returns id as seen by callerID: ErrNotFound when callerID has no
// permission on it at all (real ownership or any share) — the Store
// resolves and always returns Permission (possibly empty), and this is the
// one place an empty Permission is translated into "not found."
func (s *Service) Get(ctx context.Context, callerID, id string) (Account, error) {
	acc, err := s.store.Get(ctx, id, callerID)
	if err != nil {
		return Account{}, err
	}
	if acc.Permission == "" {
		return Account{}, ErrNotFound
	}
	return acc, nil
}

// List returns every non-deleted account callerID owns or has any
// permission on, each already carrying callerID's effective Permission.
func (s *Service) List(ctx context.Context, callerID string) ([]Account, error) {
	return s.store.List(ctx, callerID)
}

// VisibleIDs returns the id of every non-deleted account callerID owns or
// has any share on. Satisfies internal/entry's AccountLookup interface,
// used to scope an entry listing or balance query to accounts that still
// exist and are visible to the caller.
func (s *Service) VisibleIDs(ctx context.Context, callerID string) ([]string, error) {
	return s.store.VisibleIDs(ctx, callerID)
}

// Access resolves what callerID may do on id — satisfies internal/entry's
// AccountLookup interface. Return values are plain (not the Access struct)
// so *Service can satisfy that interface structurally without internal/
// entry importing this package (mirroring the pre-sharing Owner method's
// same shape). permission is "" when callerID has no access at all;
// entry.Service treats that the same way this package treats it (as not
// found), without needing to know shares exist as a concept.
func (s *Service) Access(ctx context.Context, id, callerID string) (currency string, disabled bool, permission string, err error) {
	a, err := s.store.Access(ctx, id, callerID)
	if err != nil {
		return "", false, "", err
	}
	return a.Currency, a.Disabled, string(a.Permission), nil
}

// Update validates and applies a partial change to id, authorized for
// callerID. The caller must hold owner-tier permission (real ownership or
// a shared owner grant); every field, type included, is editable by any
// owner-tier caller. A provided type is trimmed of surrounding whitespace
// before it is stored.
func (s *Service) Update(ctx context.Context, callerID, id string, upd Update) (Account, error) {
	current, err := s.store.Get(ctx, id, callerID)
	if err != nil {
		return Account{}, err
	}
	if current.Permission == "" {
		return Account{}, ErrNotFound
	}
	if !current.Permission.AtLeast(PermissionOwner) {
		return Account{}, ErrForbidden
	}
	if err := validateUpdate(current, upd); err != nil {
		return Account{}, err
	}
	if upd.Type != nil {
		trimmed := strings.TrimSpace(*upd.Type)
		upd.Type = &trimmed
	}
	updated, err := s.store.Update(ctx, id, upd)
	if err != nil {
		return Account{}, err
	}
	updated.Permission, updated.Shared, updated.OwnerName = current.Permission, current.Shared, current.OwnerName
	return updated, nil
}

// setDisabled backs Disable/Enable: both require owner-tier permission.
func (s *Service) setDisabled(ctx context.Context, callerID, id string, disabled bool) (Account, error) {
	current, err := s.store.Get(ctx, id, callerID)
	if err != nil {
		return Account{}, err
	}
	if current.Permission == "" {
		return Account{}, ErrNotFound
	}
	if !current.Permission.AtLeast(PermissionOwner) {
		return Account{}, ErrForbidden
	}
	updated, err := s.store.SetDisabled(ctx, id, disabled)
	if err != nil {
		return Account{}, err
	}
	updated.Permission, updated.Shared, updated.OwnerName = current.Permission, current.Shared, current.OwnerName
	return updated, nil
}

// Disable blocks creating new entries against the account without hiding it
// or affecting its existing entries. Reversible via Enable. Requires
// owner-tier permission (real or shared).
func (s *Service) Disable(ctx context.Context, callerID, id string) (Account, error) {
	return s.setDisabled(ctx, callerID, id, true)
}

// Enable reverses Disable. Requires owner-tier permission (real or shared).
func (s *Service) Enable(ctx context.Context, callerID, id string) (Account, error) {
	return s.setDisabled(ctx, callerID, id, false)
}

// Delete soft-deletes id. Requires owner-tier permission (real or shared).
func (s *Service) Delete(ctx context.Context, callerID, id string) error {
	current, err := s.store.Get(ctx, id, callerID)
	if err != nil {
		return err
	}
	if current.Permission == "" {
		return ErrNotFound
	}
	if !current.Permission.AtLeast(PermissionOwner) {
		return ErrForbidden
	}
	return s.store.SoftDelete(ctx, id)
}

// --- sharing ---------------------------------------------------------

// ListShares returns every current share on accountID, visible to any
// caller holding at least PermissionView on it (the real owner is not
// included — a client already has their identity via GET
// /api/accounts/{id}'s owner_name and renders them as a fixed first row).
func (s *Service) ListShares(ctx context.Context, callerID, accountID string) ([]AccountShare, error) {
	access, err := s.store.Access(ctx, accountID, callerID)
	if err != nil {
		return nil, err
	}
	if access.Permission == "" {
		return nil, ErrNotFound
	}
	return s.store.ListShares(ctx, accountID)
}

// InviteShare shares accountID with the user identified by email, at
// permission, authorized for callerID (owner-tier required). callerName is
// the caller's own display name (or email), supplied by the handler from
// the authenticated request — used only for the notification email's "X
// shared this with you" line, so account.Service never needs a second
// lookup for its own caller's identity.
//
// Per design.md's "sharing by email is a synchronous, revealing lookup"
// decision, this deliberately does not hide whether email matched a user —
// unlike authentication's anti-enumeration endpoints, the caller here is
// already authenticated and already holds an owner-tier grant on a real
// account. An unmatched email returns ShareResult{Matched: false,
// InviteAllowed: ...} rather than an error, so the caller can offer to
// send an application invite instead.
func (s *Service) InviteShare(ctx context.Context, callerID, callerName, accountID, email string, permission Permission) (ShareResult, error) {
	if !permission.valid() {
		return ShareResult{}, ErrInvalidValue
	}
	current, err := s.store.Get(ctx, accountID, callerID)
	if err != nil {
		return ShareResult{}, err
	}
	if current.Permission == "" {
		return ShareResult{}, ErrNotFound
	}
	if !current.Permission.AtLeast(PermissionOwner) {
		return ShareResult{}, ErrForbidden
	}
	if s.users == nil {
		return ShareResult{}, ErrInvalidValue
	}

	targetID, _, ok, err := s.users.ByEmail(ctx, email)
	if err != nil {
		return ShareResult{}, err
	}
	if !ok {
		return ShareResult{Matched: false, InviteAllowed: s.users.InviteEnabled()}, nil
	}
	if targetID == callerID || targetID == current.OwnerID {
		return ShareResult{}, ErrInvalidValue
	}

	share, err := s.store.CreateOrUpdateShare(ctx, accountID, targetID, permission, callerID)
	if err != nil {
		return ShareResult{}, err
	}

	if s.mailer != nil {
		link := strings.TrimRight(s.baseURL, "/") + "/accounts/" + accountID
		if sendErr := s.mailer.SendAccountShare(ctx, email, current.Title, callerName, string(permission), link); sendErr != nil {
			// Best-effort — the share is created either way; a failed
			// notification email shouldn't undo it. Mirrors
			// notifyUserCreated's "log, don't surface" pattern.
			slog.Error("account: share notification email failed", "account_id", accountID, "error", sendErr)
		}
	}
	return ShareResult{Matched: true, Share: &share}, nil
}

// UpdateSharePermission changes an existing share's permission, authorized
// for callerID (owner-tier required). The real owner can never be a target
// (ErrInvalidValue) — they carry no share row to change.
func (s *Service) UpdateSharePermission(ctx context.Context, callerID, accountID, targetUserID string, permission Permission) (AccountShare, error) {
	if !permission.valid() {
		return AccountShare{}, ErrInvalidValue
	}
	current, err := s.store.Get(ctx, accountID, callerID)
	if err != nil {
		return AccountShare{}, err
	}
	if current.Permission == "" {
		return AccountShare{}, ErrNotFound
	}
	if !current.Permission.AtLeast(PermissionOwner) {
		return AccountShare{}, ErrForbidden
	}
	if targetUserID == current.OwnerID {
		return AccountShare{}, ErrInvalidValue
	}
	return s.store.UpdateSharePermission(ctx, accountID, targetUserID, permission)
}

// RevokeShare removes targetUserID's share on accountID, unconditionally
// (no soft delete — see design.md). It authorizes two distinct cases the
// same way: an owner-tier callerID revoking someone else's share, or any
// user revoking their own (self-leave, any tier) — so this single method
// backs both "Revoke" and "Leave" in the client, and a single
// DELETE /api/accounts/{id}/shares/{userId} route serves both. The real
// owner can never be a target (ErrInvalidValue) — they carry no share row,
// so there is nothing to revoke and no self-leave for them either.
func (s *Service) RevokeShare(ctx context.Context, callerID, accountID, targetUserID string) error {
	current, err := s.store.Get(ctx, accountID, callerID)
	if err != nil {
		return err
	}
	if current.Permission == "" {
		return ErrNotFound
	}
	if targetUserID == current.OwnerID {
		return ErrInvalidValue
	}
	if targetUserID != callerID && !current.Permission.AtLeast(PermissionOwner) {
		return ErrForbidden
	}
	return s.store.DeleteShare(ctx, accountID, targetUserID)
}

// --- account types -----------------------------------------------------

// ListInUseTypes returns the distinct, non-empty type labels present on
// ownerID's own non-deleted accounts, sorted case-insensitively — for the
// account form's autocomplete. A type has no existence apart from being
// written on an account, so there is nothing to create, rename, or delete.
func (s *Service) ListInUseTypes(ctx context.Context, ownerID string) ([]string, error) {
	return s.store.ListInUseTypes(ctx, ownerID)
}
