package tag

import (
	"context"
	"errors"
	"log/slog"
	"strings"
)

// UserLookup is the narrow view of internal/auth that tag needs for
// sharing: resolving a share-invite email to an existing user, and
// reporting whether the instance currently allows sending a new
// application invite. *auth.Service satisfies this structurally — it
// already implements the identical interface for category.UserLookup.
type UserLookup interface {
	// ByEmail resolves email to an existing, non-disabled, non-soft-deleted
	// user's id and display name. ok is false when no such user exists —
	// never an error.
	ByEmail(ctx context.Context, email string) (userID, displayName string, ok bool, err error)
	// InviteEnabled reports whether an authenticated user may currently
	// send a new application invite.
	InviteEnabled() bool
}

// Mailer sends the one transactional email tag produces. internal/mailer
// implements it alongside category.Mailer.
type Mailer interface {
	// SendTagShare emails addr that tagName has been shared with them by
	// granterName at permission, with a link into the application.
	SendTagShare(ctx context.Context, addr, tagName, granterName, permission, link string) error
}

// Service is the tag use-case layer. It depends on the Store interface
// plus the optional UserLookup/Mailer side-effect interfaces sharing uses.
// Every method is scoped to callerID — the authenticated caller — with no
// admin override; a tag the caller has no permission on (a different
// owner's, unshared) reads as ErrNotFound.
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

// NewService builds the tag service.
func NewService(store Store, opts ...Option) *Service {
	s := &Service{store: store}
	for _, o := range opts {
		o(s)
	}
	return s
}

// SetUserLookup wires the email-lookup/invite-eligibility source after
// construction — package main builds tag.Service before auth.Service
// exists (mirrors category.Service.SetUserLookup's same wiring-cycle
// need). A nil users (never called) makes every sharing email-invite fail
// closed (ErrInvalidValue), never panic.
func (s *Service) SetUserLookup(u UserLookup) { s.users = u }

// List returns every tag callerID owns or has any share on.
func (s *Service) List(ctx context.Context, callerID string) ([]Tag, error) {
	return s.store.List(ctx, callerID)
}

// Get returns id as seen by callerID: ErrNotFound when callerID has no
// permission on it at all (real ownership or any share) — the Store
// resolves and always returns Permission (possibly empty), and this is the
// one place an empty Permission is translated into "not found."
func (s *Service) Get(ctx context.Context, callerID, id string) (Tag, error) {
	t, err := s.store.GetForCaller(ctx, callerID, id)
	if err != nil {
		return Tag{}, err
	}
	if t.Permission == "" {
		return Tag{}, ErrNotFound
	}
	return t, nil
}

// Create validates name and creates a tag owned by ownerID, or
// ErrDuplicateName if they already have one with this name.
func (s *Service) Create(ctx context.Context, ownerID, name string) (Tag, error) {
	if err := validateName(name); err != nil {
		return Tag{}, err
	}
	t, err := s.store.Create(ctx, ownerID, name)
	if err != nil {
		return Tag{}, err
	}
	t.Permission = PermissionOwner
	return t, nil
}

// GetOrCreate returns ownerID's existing tag named name, creating it first
// if none exists yet — the inline "create a tag on the fly from the entry
// form" path, so the caller never has to handle ErrDuplicateName itself.
func (s *Service) GetOrCreate(ctx context.Context, ownerID, name string) (Tag, error) {
	if err := validateName(name); err != nil {
		return Tag{}, err
	}
	existing, err := s.store.ByName(ctx, ownerID, name)
	if err == nil {
		existing.Permission = PermissionOwner
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Tag{}, err
	}
	created, err := s.store.Create(ctx, ownerID, name)
	if errors.Is(err, ErrDuplicateName) {
		// Lost a race with a concurrent create of the same name.
		existing, err := s.store.ByName(ctx, ownerID, name)
		if err != nil {
			return Tag{}, err
		}
		existing.Permission = PermissionOwner
		return existing, nil
	}
	if err != nil {
		return Tag{}, err
	}
	created.Permission = PermissionOwner
	return created, nil
}

// Update renames ownerID's tag.
func (s *Service) Update(ctx context.Context, ownerID, id, name string) (Tag, error) {
	if err := validateName(name); err != nil {
		return Tag{}, err
	}
	t, err := s.store.Update(ctx, ownerID, id, name)
	if err != nil {
		return Tag{}, err
	}
	t.Permission = PermissionOwner
	return t, nil
}

// Delete removes ownerID's tag, detaching it from every entry it was
// attached to (enforced by the real backend's ON DELETE CASCADE on
// entry_tags.tag_id) — allowed regardless of how many of the owner's own
// entries carry it, unless the tag currently has at least one active
// share, in which case it returns ErrInUse.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	return s.store.Delete(ctx, ownerID, id)
}

// Disable blocks ownerID's tag from being newly attached to an entry,
// without affecting any entry that already carries it.
func (s *Service) Disable(ctx context.Context, ownerID, id string) (Tag, error) {
	t, err := s.store.SetDisabled(ctx, ownerID, id, true)
	if err != nil {
		return Tag{}, err
	}
	t.Permission = PermissionOwner
	return t, nil
}

// Enable reverses Disable.
func (s *Service) Enable(ctx context.Context, ownerID, id string) (Tag, error) {
	t, err := s.store.SetDisabled(ctx, ownerID, id, false)
	if err != nil {
		return Tag{}, err
	}
	t.Permission = PermissionOwner
	return t, nil
}

// OwnedBy satisfies internal/entry's TagLookup interface: reports whether
// every id in tagIDs is visible to callerID — owned by them, or shared
// with them at any tier.
func (s *Service) OwnedBy(ctx context.Context, callerID string, tagIDs []string) (bool, error) {
	return s.store.OwnedBy(ctx, callerID, tagIDs)
}

// Usable satisfies internal/entry's TagLookup interface: reports whether
// every id in tagIDs is usable by callerID for tagging an entry — owned by
// them, or shared with them at append tier — and not disabled.
func (s *Service) Usable(ctx context.Context, callerID string, tagIDs []string) (bool, error) {
	return s.store.Usable(ctx, callerID, tagIDs)
}

// --- sharing ---------------------------------------------------------

// ListShares returns every current share on tagID, visible to any caller
// holding at least PermissionView on it (the real owner is not included —
// a client already has their identity via GET /api/tags/{id}'s owner_name
// and renders them as a fixed first row).
func (s *Service) ListShares(ctx context.Context, callerID, tagID string) ([]TagShare, error) {
	t, err := s.store.GetForCaller(ctx, callerID, tagID)
	if err != nil {
		return nil, err
	}
	if t.Permission == "" {
		return nil, ErrNotFound
	}
	return s.store.ListShares(ctx, tagID)
}

// InviteShare shares tagID with the user identified by email, at
// permission, authorized for callerID. There is no shareable "owner" tier
// that could also admit this — only the tag's real owner may ever invite,
// change, or revoke a share. callerName is the caller's own display name
// (or email), supplied by the handler from the authenticated request —
// used only for the notification email's "X shared this with you" line.
//
// Mirrors category.Service.InviteShare's deliberately non-anti-enumeration
// decision: the caller here is already authenticated and already holds a
// real-owner grant on a real tag, so an unmatched email is reported back
// synchronously (Matched: false) rather than hidden.
func (s *Service) InviteShare(ctx context.Context, callerID, callerName, tagID, email string, permission Permission) (ShareResult, error) {
	if !permission.valid() {
		return ShareResult{}, ErrInvalidValue
	}
	current, err := s.store.GetForCaller(ctx, callerID, tagID)
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

	share, err := s.store.CreateOrUpdateShare(ctx, tagID, targetID, permission, callerID)
	if err != nil {
		return ShareResult{}, err
	}

	if s.mailer != nil {
		link := strings.TrimRight(s.baseURL, "/") + "/tags/" + tagID + "/sharing"
		if sendErr := s.mailer.SendTagShare(ctx, email, current.Name, callerName, string(permission), link); sendErr != nil {
			// Best-effort — the share is created either way; a failed
			// notification email shouldn't undo it.
			slog.Error("tag: share notification email failed", "tag_id", tagID, "error", sendErr)
		}
	}
	return ShareResult{Matched: true, Share: &share}, nil
}

// UpdateSharePermission changes an existing share's permission, authorized
// for callerID — the tag's real owner only. The real owner can never be a
// target (ErrInvalidValue) — they carry no share row to change.
func (s *Service) UpdateSharePermission(ctx context.Context, callerID, tagID, targetUserID string, permission Permission) (TagShare, error) {
	if !permission.valid() {
		return TagShare{}, ErrInvalidValue
	}
	current, err := s.store.GetForCaller(ctx, callerID, tagID)
	if err != nil {
		return TagShare{}, err
	}
	if current.Permission == "" {
		return TagShare{}, ErrNotFound
	}
	if current.Permission != PermissionOwner {
		return TagShare{}, ErrForbidden
	}
	if targetUserID == current.OwnerID {
		return TagShare{}, ErrInvalidValue
	}
	return s.store.UpdateSharePermission(ctx, tagID, targetUserID, permission)
}

// RevokeShare removes targetUserID's share on tagID, unconditionally (no
// soft delete). It authorizes two distinct cases the same way: the real
// owner revoking someone else's share, or any user revoking their own
// (self-leave, any tier) — so this single method backs both "Revoke" and
// "Leave" in the client. The real owner can never be a target
// (ErrInvalidValue) — they carry no share row, so there is nothing to
// revoke and no self-leave for them either.
func (s *Service) RevokeShare(ctx context.Context, callerID, tagID, targetUserID string) error {
	current, err := s.store.GetForCaller(ctx, callerID, tagID)
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
	return s.store.DeleteShare(ctx, tagID, targetUserID)
}
