package category

import (
	"context"
	"errors"
	"log/slog"
	"strings"
)

// UserLookup is the narrow view of internal/auth that category needs for
// sharing: resolving a share-invite email to an existing user, and
// reporting whether the instance currently allows sending a new
// application invite. *auth.Service satisfies this structurally — it
// already implements the identical interface for account.UserLookup.
type UserLookup interface {
	// ByEmail resolves email to an existing, non-disabled, non-soft-deleted
	// user's id and display name. ok is false when no such user exists —
	// never an error.
	ByEmail(ctx context.Context, email string) (userID, displayName string, ok bool, err error)
	// InviteEnabled reports whether an authenticated user may currently
	// send a new application invite.
	InviteEnabled() bool
}

// Mailer sends the one transactional email category produces. internal/
// mailer implements it alongside account.Mailer.
type Mailer interface {
	// SendCategoryShare emails addr that categoryName has been shared with
	// them by granterName at permission, with a link into the application.
	SendCategoryShare(ctx context.Context, addr, categoryName, granterName, permission, link string) error
}

// Service is the category use-case layer. It depends on the Store
// interface plus the optional UserLookup/Mailer side-effect interfaces
// sharing uses. Every method is scoped to callerID — the authenticated
// caller — with no admin override; a category the caller has no
// permission on (a different owner's, unshared) reads as ErrNotFound.
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
// notification's link.
func WithBaseURL(url string) Option { return func(s *Service) { s.baseURL = url } }

// NewService builds the category service.
func NewService(store Store, opts ...Option) *Service {
	s := &Service{store: store}
	for _, o := range opts {
		o(s)
	}
	return s
}

// SetUserLookup wires the email-lookup/invite-eligibility source after
// construction — package main builds category.Service before auth.Service
// exists (mirrors account.Service.SetUserLookup's same wiring-cycle need).
// A nil users (never called) makes every sharing email-invite fail closed
// (ErrInvalidValue), never panic.
func (s *Service) SetUserLookup(u UserLookup) { s.users = u }

// List returns every non-deleted category callerID owns or has any share
// on.
func (s *Service) List(ctx context.Context, callerID string) ([]Category, error) {
	return s.store.List(ctx, callerID)
}

// Get returns id as seen by callerID: ErrNotFound when callerID has no
// permission on it at all (real ownership or any share) — the Store
// resolves and always returns Permission (possibly empty), and this is the
// one place an empty Permission is translated into "not found."
func (s *Service) Get(ctx context.Context, callerID, id string) (Category, error) {
	c, err := s.store.GetForCaller(ctx, callerID, id)
	if err != nil {
		return Category{}, err
	}
	if c.Permission == "" {
		return Category{}, ErrNotFound
	}
	return c, nil
}

// Create validates in and, when it sets a ParentID, checks it exists among
// ownerID's own categories.
func (s *Service) Create(ctx context.Context, ownerID string, in New) (Category, error) {
	if err := validateName(in.Name); err != nil {
		return Category{}, err
	}
	if !validPresentationToken(in.Icon) || !validPresentationToken(in.Color) {
		return Category{}, ErrInvalidValue
	}
	if in.ParentID != nil {
		ok, err := s.store.Exists(ctx, ownerID, *in.ParentID)
		if err != nil {
			return Category{}, err
		}
		if !ok {
			return Category{}, ErrInvalidValue
		}
	}
	c, err := s.store.Create(ctx, ownerID, in)
	if err != nil {
		return Category{}, err
	}
	c.Permission = PermissionOwner
	return c, nil
}

// Update validates upd, rejecting a reparent onto the category itself or
// one of its own descendants (ErrCycle), scoped to ownerID's own tree.
func (s *Service) Update(ctx context.Context, ownerID, id string, upd Update) (Category, error) {
	if upd.Name != nil {
		if err := validateName(*upd.Name); err != nil {
			return Category{}, err
		}
	}
	if upd.Icon != nil && !validPresentationToken(*upd.Icon) {
		return Category{}, ErrInvalidValue
	}
	if upd.Color != nil && !validPresentationToken(*upd.Color) {
		return Category{}, ErrInvalidValue
	}
	if upd.ParentID.Set && upd.ParentID.Value != nil {
		newParent := *upd.ParentID.Value
		if newParent == id {
			return Category{}, ErrCycle
		}
		ok, err := s.store.Exists(ctx, ownerID, newParent)
		if err != nil {
			return Category{}, err
		}
		if !ok {
			return Category{}, ErrInvalidValue
		}
		descendants, err := s.store.Subtree(ctx, ownerID, id)
		if err != nil {
			return Category{}, err
		}
		for _, d := range descendants {
			if d == newParent {
				return Category{}, ErrCycle
			}
		}
	}
	c, err := s.store.Update(ctx, ownerID, id, upd)
	if err != nil {
		return Category{}, err
	}
	c.Permission = PermissionOwner
	return c, nil
}

// Delete soft-deletes ownerID's category, or ErrInUse if it has a
// non-deleted child or is referenced elsewhere.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	return s.store.Delete(ctx, ownerID, id)
}

// setDisabled backs Disable/Enable, both real-owner-only operations.
func (s *Service) setDisabled(ctx context.Context, ownerID, id string, disabled bool) (Category, error) {
	c, err := s.store.SetDisabled(ctx, ownerID, id, disabled)
	if err != nil {
		return Category{}, err
	}
	c.Permission = PermissionOwner
	return c, nil
}

// Disable blocks the category from being newly selected on an entry,
// without affecting any entry or child category already referencing it.
// Reversible via Enable.
func (s *Service) Disable(ctx context.Context, ownerID, id string) (Category, error) {
	return s.setDisabled(ctx, ownerID, id, true)
}

// Enable reverses Disable.
func (s *Service) Enable(ctx context.Context, ownerID, id string) (Category, error) {
	return s.setDisabled(ctx, ownerID, id, false)
}

// MoveUp moves ownerID's category up one position among its current
// siblings. A no-op if it is already first.
func (s *Service) MoveUp(ctx context.Context, ownerID, id string) (Category, error) {
	c, err := s.store.MoveUp(ctx, ownerID, id)
	if err != nil {
		return Category{}, err
	}
	c.Permission = PermissionOwner
	return c, nil
}

// MoveDown moves ownerID's category down one position among its current
// siblings. A no-op if it is already last.
func (s *Service) MoveDown(ctx context.Context, ownerID, id string) (Category, error) {
	c, err := s.store.MoveDown(ctx, ownerID, id)
	if err != nil {
		return Category{}, err
	}
	c.Permission = PermissionOwner
	return c, nil
}

// Usable satisfies internal/entry's CategoryLookup interface: it reports
// whether id is usable by callerID for categorizing an entry — owned by
// them, or shared with them at append tier — and not disabled. Consulted
// only when a category is being newly set (created, or explicitly
// changed) on an entry; a disabled or since-unshared category already
// referenced by an entry stays valid there, since this method is never
// consulted for a value merely carried over unchanged.
func (s *Service) Usable(ctx context.Context, callerID, id string) (bool, error) {
	c, err := s.store.GetForCaller(ctx, callerID, id)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if c.Disabled {
		return false, nil
	}
	return c.Permission.AtLeast(PermissionAppend), nil
}

// Subtree satisfies internal/entry's CategoryLookup interface: it resolves
// id to the categories a caller-supplied filter should match. For a
// category callerID owns, that's id and every descendant within their own
// tree (delegating to the owner-scoped recursion below). For one visible
// to callerID only via a share (view or append — filtering only ever
// requires view), it's id alone, since a share never cascades to
// descendants. For one callerID has no access to at all, it's empty.
func (s *Service) Subtree(ctx context.Context, callerID, id string) ([]string, error) {
	c, err := s.store.GetForCaller(ctx, callerID, id)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if c.Permission == "" {
		return nil, nil
	}
	if c.Permission == PermissionOwner {
		return s.store.Subtree(ctx, callerID, id)
	}
	return []string{id}, nil
}

// SeedDefaults seeds ownerID — a brand-new user — with DefaultNames as root
// categories. It satisfies internal/auth's NewUserHook interface
// structurally, wired in by package main via auth.WithNewUserHooks.
func (s *Service) SeedDefaults(ctx context.Context, ownerID string) error {
	return s.store.SeedDefaults(ctx, ownerID)
}

// --- sharing ---------------------------------------------------------

// ListShares returns every current share on categoryID, visible to any
// caller holding at least PermissionView on it (the real owner is not
// included — a client already has their identity via GET
// /api/categories/{id}'s owner_name and renders them as a fixed first row).
func (s *Service) ListShares(ctx context.Context, callerID, categoryID string) ([]CategoryShare, error) {
	c, err := s.store.GetForCaller(ctx, callerID, categoryID)
	if err != nil {
		return nil, err
	}
	if c.Permission == "" {
		return nil, ErrNotFound
	}
	return s.store.ListShares(ctx, categoryID)
}

// InviteShare shares categoryID with the user identified by email, at
// permission, authorized for callerID. Unlike account-sharing there is no
// shareable "owner" tier that could also admit this — only the category's
// real owner may ever invite, change, or revoke a share. callerName is the
// caller's own display name (or email), supplied by the handler from the
// authenticated request — used only for the notification email's "X
// shared this with you" line.
//
// Mirrors account.Service.InviteShare's deliberately non-anti-enumeration
// decision: the caller here is already authenticated and already holds a
// real-owner grant on a real category, so an unmatched email is reported
// back synchronously (Matched: false) rather than hidden.
func (s *Service) InviteShare(ctx context.Context, callerID, callerName, categoryID, email string, permission Permission) (ShareResult, error) {
	if !permission.valid() {
		return ShareResult{}, ErrInvalidValue
	}
	current, err := s.store.GetForCaller(ctx, callerID, categoryID)
	if err != nil {
		return ShareResult{}, err
	}
	if current.Permission == "" {
		return ShareResult{}, ErrNotFound
	}
	if current.Permission != PermissionOwner {
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
	if targetID == callerID {
		return ShareResult{}, ErrInvalidValue
	}

	share, err := s.store.CreateOrUpdateShare(ctx, categoryID, targetID, permission, callerID)
	if err != nil {
		return ShareResult{}, err
	}

	if s.mailer != nil {
		link := strings.TrimRight(s.baseURL, "/") + "/categories/" + categoryID + "/sharing"
		if sendErr := s.mailer.SendCategoryShare(ctx, email, current.Name, callerName, string(permission), link); sendErr != nil {
			// Best-effort — the share is created either way; a failed
			// notification email shouldn't undo it.
			slog.Error("category: share notification email failed", "category_id", categoryID, "error", sendErr)
		}
	}
	return ShareResult{Matched: true, Share: &share}, nil
}

// UpdateSharePermission changes an existing share's permission, authorized
// for callerID — the category's real owner only. The real owner can never
// be a target (ErrInvalidValue) — they carry no share row to change.
func (s *Service) UpdateSharePermission(ctx context.Context, callerID, categoryID, targetUserID string, permission Permission) (CategoryShare, error) {
	if !permission.valid() {
		return CategoryShare{}, ErrInvalidValue
	}
	current, err := s.store.GetForCaller(ctx, callerID, categoryID)
	if err != nil {
		return CategoryShare{}, err
	}
	if current.Permission == "" {
		return CategoryShare{}, ErrNotFound
	}
	if current.Permission != PermissionOwner {
		return CategoryShare{}, ErrForbidden
	}
	if targetUserID == current.OwnerID {
		return CategoryShare{}, ErrInvalidValue
	}
	return s.store.UpdateSharePermission(ctx, categoryID, targetUserID, permission)
}

// RevokeShare removes targetUserID's share on categoryID, unconditionally
// (no soft delete). It authorizes two distinct cases the same way: the
// real owner revoking someone else's share, or any user revoking their own
// (self-leave, any tier) — so this single method backs both "Revoke" and
// "Leave" in the client. The real owner can never be a target
// (ErrInvalidValue) — they carry no share row, so there is nothing to
// revoke and no self-leave for them either.
func (s *Service) RevokeShare(ctx context.Context, callerID, categoryID, targetUserID string) error {
	current, err := s.store.GetForCaller(ctx, callerID, categoryID)
	if err != nil {
		return err
	}
	if current.Permission == "" {
		return ErrNotFound
	}
	if targetUserID == current.OwnerID {
		return ErrInvalidValue
	}
	if targetUserID != callerID && current.Permission != PermissionOwner {
		return ErrForbidden
	}
	return s.store.DeleteShare(ctx, categoryID, targetUserID)
}
