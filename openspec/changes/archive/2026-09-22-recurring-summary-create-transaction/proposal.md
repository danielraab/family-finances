## Why

The recurring summary modal is read-only and offers exactly two actions:
Close and Edit. But the reason a reader opens it is usually not to change
the template — it is to decide whether to book this occurrence. Every
surface that lists a recurring transaction already offers "Create
transaction" right there in the row (`/recurring`'s list, the Upcoming
block), so the summary is the one place where the reader can see the full
template and *cannot* act on it: they have to close the modal and find the
row's own button again.

That gap just got wider. An Upcoming row's title now opens this summary,
so the most natural path to a projected occurrence — see it listed, open
it to check the amount and account, book it — dead-ends at a modal with
nothing but Edit.

## What Changes

- **The recurring summary modal gains a "Create transaction" action**
  beside Edit, navigating to
  `/entries/new?recurring_transaction_id={id}` exactly as `/recurring`'s
  own row action does. It is bordered rather than filled, so it reads as
  an action without competing with Edit for primacy, and is distinct from
  Close, which is a dismissal.
- **When the summary was opened from a projected occurrence, that
  occurrence's date is carried into the action** as
  `booking_timestamp`, so booking from the third Upcoming row books that
  row's date and not the template's next suggested one. Opened from
  anywhere else (a `/recurring` row, a linked entry's badge) no date is
  sent and the entry form's own default applies, unchanged.

## Non-goals

- **No API contract change.** The action reuses the existing
  `/entries/new` search parameters.
- **No new translations.** The action reuses
  `recurring.createTransaction`, which both locales already carry.
- **No change to what the summary shows**, to Edit, to the entry summary,
  or to the cross-link between the two.
- **The action is not hidden on an ended template.** `/recurring` offers
  it on an ended row too; making the summary stricter than the list it
  mirrors would be a new rule, not a fix.

## Capabilities

### Modified Capabilities

- `web-client-recurring-transactions`: the read-only recurring summary
  offers a Create transaction action, carrying the projected date when it
  was opened from one.

## Impact

- `frontend/src/components/summary/RecurringSummaryModal.tsx` — the action.
- `frontend/src/components/summary/SummaryField.tsx` — a bordered
  secondary-action class beside the existing filled and quiet ones.
- `frontend/src/components/summary/useSummaryModals.tsx` — `openRecurring`
  takes an optional occurrence date and carries it to the modal.
- `frontend/src/components/UpcomingBlock.tsx` — a row's title passes its
  own `booking_timestamp`.
