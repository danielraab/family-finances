## Context

`entries.$entryId.edit.tsx` already has a "Linked recurring transaction"
`<select>` that can attach an entry to an *existing* recurring transaction
(fetched scoped to the entry's account), and `entries.new.tsx` already
supports the reverse prefill direction: `?recurring_transaction_id=` +
optional `?booking_timestamp=` fetches a template and prefills a new entry
from it, locking the account (see `UpcomingBlock`'s "Create transaction"
link). There is no path today from an *entry* to a *new* template — the
motivating gap this change closes.

Two backend facts drive the design, both confirmed by reading
`backend/internal/recurringtransaction/service.go` and
`backend/internal/entry/service.go`:

- `NextSuggestedDate` is `starts_on` when a template has no linked entries,
  and otherwise `<latest linked entry's date> + 1 interval`
  (`service.go`'s `nextSuggestedDate`). If a template is created from an
  entry but never linked back to it, `NextSuggestedDate` stays pinned at
  `starts_on` (the entry's own date) — `UpcomingBlock` would then show a
  phantom "upcoming" occurrence for a date that's already booked.
- `checkRecurringTransaction` (`entry/service.go`) only enforces that a
  linked recurring transaction is on the same account as the entry — no
  date or amount coupling. Auto-linking the origin entry after template
  creation is therefore always safe to attempt once the template exists.

## Goals / Non-Goals

**Goals:**
- One click (two, if the entry has unsaved edits) from an existing
  transaction entry to a prefilled, auto-linked recurring transaction
  template.
- Stay inside the existing prefill idiom this codebase already uses
  (`entries.new.tsx`'s `?recurring_transaction_id=`): pass an id, fetch
  fresh on the destination route, never serialize a whole entity into the
  URL or router state.
- No backend changes — compose entirely from `GET /api/entries/{id}`,
  `POST /api/recurring-transactions`, and `PATCH /api/entries/{id}`.

**Non-Goals:**
- No opt-out of auto-linking (no checkbox) — see the phantom-suggestion
  reasoning above.
- No deep dirty-diffing against the loaded entry (e.g. detecting that a
  field was edited and then edited back to its original value) — a single
  touched/untouched flag is enough.
- No change to `entries.new.tsx`'s existing `recurring_transaction_id` /
  `booking_timestamp` flow, or to the "Linked recurring transaction"
  dropdown's own behavior (only its layout changes, to make room for the
  new button beside it).
- No balance-adjustment support — recurring transactions require
  `category_id` and have no balance concept, so the action is scoped to
  `entry.kind === "transaction"` only.

## Decisions

**Icon-only action, placed beside the "Linked recurring transaction"
select rather than in the Save/Delete button row.** That field is where a
visitor already thinks about this entry's relationship to a recurring
template — its `<select>` picks an *existing* one to link, so a "+"
button immediately to its right (same `flex items-center gap-2` wrapper
this file's account-field unlock control already uses) reads as "or
create a new one," without needing the icon itself to convey "recurring"
the way it would have to standing alone in the button row. The glyph is a
plain plus (`PlusGlyph`, the same hand-rolled path-data shape already
duplicated locally in `accounts.index.tsx`/`AccountCard.tsx` for "add new"
actions — not a shared component, matching how this codebase treats that
particular glyph) rather than `RecurringTransactionBadge.tsx`'s
repeat-arrows SVG; the "linked recurring transaction" context already
supplies the "recurring" meaning, so the icon only needs to say "add/create."
It's still icon-only (accessible name via `aria-label`/`title`, mirroring
this same file's existing ✎/✕ account-unlock buttons) to stay narrow on
mobile regardless of how long the two label strings are. Because this
field sits inside the form's `<fieldset disabled={!canEdit}>`, the
`<button>` variant (dirty state) is naturally disabled by the fieldset
along with the rest of the form, but the `<Link>` variant (clean state)
renders as a plain `<a>`, which a `disabled` fieldset does **not** reach —
so its render condition explicitly checks `canEdit` too, rather than
relying on the fieldset alone.

**Single touched/untouched boolean for dirty state, not a value diff.**
The form has no notion of "dirty" today. Reusing the app's cheapest
possible mechanism — a `dirty` flag flipped `true` by every field's
existing `onChange`/`set` path (via one added dependency-array effect or a
shared setter wrapper) and reset `false` right after the initial load and
right after a successful save — avoids introducing a baseline-comparison
system this codebase doesn't have anywhere else. The cost: toggling a
field back to its original value still counts as dirty. Accepted, per the
proposal's scope decision.

**`performSubmit`'s post-save destination becomes a parameter.**
Today it hardcodes `navigate({ to: "/entries", search: { account_id }
})`. The "Save and create recurring transaction" click needs to run the
exact same validation, PATCH call, and account-change confirmation dialog,
but redirect to `/recurring/new?from_entry_id=<id>` instead. Threading the
destination through as a parameter (rather than duplicating
`performSubmit`) keeps the two Save paths from drifting apart.

**`from_entry_id` fetch-and-prefill on `/recurring/new`, mirroring
`entries.new.tsx`'s existing pattern exactly.** A new optional search
param, validated the same way `recurring_transaction_id`/`account_id`
already are on that route. On mount, if present: fetch the entry, prefill
`RecurringTransactionForm`'s `initial` (title, description, category_id,
counterparty, location, tag names resolved via the already-fetched tag
list, amount magnitude/sign), pass `accountLocked` (the prop already
exists, currently used for the single-account case), and default
`starts_on` to the entry's `booking_timestamp` date portion (no
time-of-day on a recurring template). Considered instead: passing every
field as its own query parameter to avoid the extra fetch. Rejected —
every existing cross-page prefill in this app fetches by id, and a
several-field query string has no precedent here and would need its own
encoding/parsing rules for free text (description, location, tags).

**Auto-link happens as a second request after successful creation, on the
`/recurring/new` page, not as part of the entry-edit page's flow.** Once
`POST /api/recurring-transactions` returns the new template's id,
immediately `PATCH /api/entries/{id}` (the original `from_entry_id`) with
`recurring_transaction_id` set to it, then navigate to `/recurring`. Doing
the link on the destination page (rather than asking the edit page to poll
or redirect back) keeps the whole "create from entry" flow inside one
route/component and matches where the template id first becomes known.

**Lands on `/recurring`, not `/recurring/$id/edit` or back on the
originating entry.** `/recurring` is where the new template's existence is
most visible (the full list, ordered like every other post-create flow in
this app — see `recurring.new.tsx`'s current unconditional `navigate({ to:
"/recurring" })` on success, which this change keeps for the normal
no-`from_entry_id` path and extends to the `from_entry_id` path once the
link `PATCH` settles).

## Risks / Trade-offs

- **[Risk]** The auto-link `PATCH` could fail even though the template
  itself was created successfully (network blip, entry deleted
  concurrently), leaving a template with no linked entry and a
  `starts_on`-pinned `NextSuggestedDate` — the exact phantom-suggestion
  case this design tries to avoid. → **Mitigation**: still navigate to
  `/recurring` (the template exists and is usable either way); surface the
  link failure as a non-blocking toast/inline note rather than blocking
  navigation on it, since the template creation itself already succeeded
  and shouldn't be rolled back or re-attempted automatically.
- **[Risk]** A single dirty flag means a visitor who opens the account
  picker, changes nothing, and closes it still sees "Save and create
  recurring transaction" instead of the direct action. → **Mitigation**:
  accepted per Non-Goals; false positives here just cost one extra
  (harmless, idempotent) save, not incorrect data.
- **[Trade-off]** Icon-only with no visible label relies on `title`/hover
  or long-press for sighted mobile users to discover what the button does
  before tapping it. → Accepted per the earlier design discussion: it
  mirrors this same page's existing icon-only ✎/✕ controls, and the icon
  is already a recognized "recurring" affordance elsewhere in the app.

## Open Questions

None outstanding — the flow, dirty-state handling, icon choice, and
landing route were all settled during exploration before this proposal was
written.
