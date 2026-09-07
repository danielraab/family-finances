## Context

See proposal.md - Why. Relevant existing constraints:

- Account balance is always computed live from an account's entries
  (`account-entries` spec, "Account balance is always computed live") —
  never stored or cached. Reassigning an entry's `account_id` therefore
  requires no balance migration on either side.
- `entry.Update` (`backend/internal/entry/entry.go`) deliberately has no
  `AccountID` field today; `entry.Service.Update` never re-checks account
  ownership/disabled state because it never receives a new account.
  `entry.Service.Create` already has this exact check (`s.accounts.Owner`
  + `disabled`) — the move path reuses it rather than inventing a second
  version.
- The frontend's category picker (`categories.tsx`) already has a
  disabled-current-value rendering pattern (render it, but don't offer it
  as a choice) that the account `<select>` reuses for consistency.

## Goals / Non-Goals

**Goals:**
- Let a user reassign an existing entry's `account_id` to another account
  they own, through both the API and the edit form.
- Reuse the existing ownership/disabled-account checks and existing UI
  patterns (disabled-current-value rendering, `Dialog` confirmation)
  rather than introducing new mechanisms.

**Non-Goals:**
- Currency conversion. This change never touches an entry's `amount`; a
  cross-currency move is a client-side warning only, not a computation.
- Bulk move (moving multiple entries at once) — out of scope, one entry at
  a time via the existing edit form.
- An audit trail of past account moves — not part of this change.

## Decisions

**`entry.Update` gains an `AccountID *string` field, validated in
`Service.Update` the same way `Service.Create` validates it.** Both call
`s.accounts.Owner(ctx, id)` and reject on ownership mismatch
(`ErrInvalidValue`) or `disabled` (`ErrAccountDisabled`). Extracting a
shared helper (e.g. `s.checkAccount(ctx, ownerID, accountID)`) avoids
duplicating that pair of checks between `Create` and `Update`.

**No new sentinel errors.** Moving into another user's account and moving
into a disabled account reuse `ErrInvalidValue` and `ErrAccountDisabled`
respectively — the same errors `Create` already returns for the same
conditions, so `httpapi/respond.go`'s status mapping needs no change.

**Currency handling is entirely client-side.** The backend does not read
or compare `currency` when validating a move — it already has no concept
of currency conversion, so introducing a check that blocks or flags a
currency mismatch would be new backend behavior for a purely
informational, UI-level concern. The frontend already fetches full
`Account` objects (including `currency`) for the account picker, so
comparing the selected account's currency to the entry's original account
is a local computation with no extra request.

**The edit form's account field is a single component with three visual
states (locked / unlocked / unlocked-with-selection), not a separate
"Move to…" dialog like `categories.tsx`'s reparent flow.** Categories
reparent from a tree-shaped picker reached from a list row, with its own
subtree-exclusion logic; entries are edited one at a time already on a
dedicated page, and the account field is a flat list with no comparable
structural exclusion problem, so an inline unlock is simpler and avoids a
second modal on top of the existing delete-confirmation one.

**The account-change confirmation dialog reuses the existing
`@headlessui/react` `Dialog`/`DialogPanel`/`DialogTitle` pattern already
used for delete confirmation** (`entries.$entryId.edit.tsx`), as a second,
independently-triggered dialog instance — triggered by Save when the
pending account selection differs from `entry.account_id`, rather than by
a dedicated button.

## Risks / Trade-offs

- **A user moves an entry cross-currency by mistake.** Mitigated by the
  inline warning (while selecting) and the confirmation dialog (before
  submitting) — two chances to notice, no hard block, consistent with the
  rest of this app trusting the user once warned.
- **Extracting a shared ownership/disabled-account helper touches
  `Service.Create` as well as `Service.Update`.** Low risk — it is a pure
  refactor of existing, already-tested logic into a shared method; no
  behavior change for `Create`.
