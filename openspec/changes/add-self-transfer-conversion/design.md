## Context

`add-self-transfer` deliberately made three things immutable after
creation: `Entry.Kind` (no field on `Update` at all), `Entry.ToAccountID`
("a self_transfer's two accounts are fixed at creation, never individually
reassigned"), and `Entry.AccountID` for a `self_transfer` specifically
(rejected outright, even though it's otherwise movable for a `transaction`/
`balance_adjustment`). Those decisions still hold and are not reopened
here — several other invariants lean on them: the `entry_legs` Postgres
view that gives a `self_transfer` its two-sided listing/summing behavior,
the stricter "`append`+ on both accounts" edit-permission rule (see
`account-entries`'s "Editing a self-transfer requires current permission on
both accounts" requirement), the recompute-on-mutation machinery running
once per account, and `internal/account`'s currency-immutability-once-
has-entries rule.

What this change adds instead is a **new, bounded escape hatch**: a
dedicated endpoint that performs the one specific transition users actually
asked for (`transaction` → `self_transfer`) by soft-deleting the original
entry and creating a genuinely new one, going through the exact same
Create-time checks a hand-built `POST /api/entries` call would. `Update`'s
contract, and everything built on the immutability of `kind`/
`to_account_id`/`account_id`, is untouched.

## Goals / Non-Goals

**Goals:**
- Let a user fix "this should have been a self-transfer" in one action,
  without hand-deleting and rebuilding the entry (losing its category and
  recurring link along the way).
- Reuse `self_transfer`'s existing Create-time authorization/validation
  rule verbatim, rather than inventing a new one — this operation *is*, at
  its core, "create a self-transfer," with an extra step of retiring the
  entry it's replacing.
- Make the transition atomic: the caller never observes a state with
  neither the original transaction nor its replacement self-transfer, and
  never a state with both live at once.
- Fix a real, independently-noticed gap in the entries list's filter
  redirect at the same time, since both the conversion's own success
  redirect and the ordinary edit-save redirect need the same fix.

**Non-Goals:**
- The reverse conversion (`self_transfer` → `transaction`) — not
  requested, not built. Nothing here precludes adding it later as its own,
  symmetric endpoint.
- Preserving the original entry's `id` — the soft-delete-and-recreate
  design deliberately produces a new one (see Decisions).
- A general saved-views or multi-slot filter-history feature — `?last=true`
  remembers exactly one thing, "the entries list's most recent filter
  state," not a list of named views.
- Any change to how `PATCH /api/entries/{id}` behaves for any kind.

## Decisions

**Soft-delete the original entry and create a new one, in one DB
transaction; the new entry gets a new `id`.** Considered: transforming the
existing row in place (same `id`, updating `kind`/`to_account_id`/
`amount`/`account_id` directly in storage, bypassing `Update`'s contract
entirely). Rejected in favor of soft-delete-and-recreate because it's a
much smaller, more auditable change — "create a self-transfer" and "delete
a transaction" are both operations this codebase already has fully
worked-out, tested authorization and recompute logic for; composing them
atomically is far less risk than teaching the storage layer a third way to
mutate an entry row that bypasses every existing kind/account invariant
check. The accepted cost: anything that referenced the original entry's
`id` (a bookmarked `/entries/{id}/edit` URL) now resolves to a
soft-deleted, dead entry — the same experience an ordinary `DELETE
/api/entries/{id}` already produces today, not a new kind of gap.

**Which account plays sender vs. receiver is caller-chosen, not inferred
from the amount's sign.** A `self_transfer`'s `amount` is valid with either
sign on `account_id` — nothing enforces that the sender's side must be
negative (see `account-entries`'s "A self-transfer's amount is signed from
the sending account's perspective" requirement, which states the
convention but doesn't constrain the sign). So a transaction's existing
amount sign is not a reliable signal for which role the account should
play after conversion; asking the caller directly is unambiguous where
inference would just be a guess dressed up as automation. The new UI
presents both options explicitly rather than defaulting one way silently.

**`recurring_transaction_id` survives the conversion only when the
original account keeps the sender (`account_id`) role.**
`RecurringTransactionLookup.SameAccount` — the check a link is validated
against, both at creation and whenever `Update` newly sets it — is always
checked against `account_id`, never `to_account_id`. If the original
account becomes the receiver, its recurring-transaction link (if any) is
simply dropped rather than either (a) silently re-pointed at the new
`account_id` it was never actually about, or (b) re-validated against an
account the recurring transaction was never defined on. Dropping is the
only option that doesn't invent a relationship that never existed.
`category_id` has no such asymmetry — it's valid on both `transaction` and
`self_transfer` regardless of which account holds which role — so it
always carries over unchanged. `counterparty`/`location` always carry over
as *absent*: `self_transfer` rejects both outright (existing `validateNew`
rule), so there's nothing to preserve either way.

**Authorization mirrors `self_transfer`'s Create rule applied to (the
original entry's account, the newly chosen account), plus the ordinary
entry-visibility gate.** Concretely: the entry is looked up the same way
`GET`/`PATCH /api/entries/{id}` already do — a caller with no permission at
all on the entry's own account gets `404`, indistinguishable from the
entry not existing (the standing "behaves as not found" convention every
other no-access case in this codebase follows). Past that gate, both
accounts — the original one and the newly chosen counterparty — go through
exactly the same check `Service.Create` already runs for a `self_transfer`
(`checkAccountCurrency`: `append`+ required on each, `400` otherwise;
non-disabled required, `422` otherwise; same `currency` required across
both, `400` otherwise). This is deliberately *not* the stricter "`append`+
on both, already-established, per the entry's own edit rule" check
`Update` applies to an existing `self_transfer` — there is no existing
`self_transfer` yet at the moment this runs, so Create's rule, not Update's,
is the one that actually applies.

**The conversion is a new, distinct route — not a mode-switch on the
existing edit form.** The edit form's fields (kind, account, to-account)
are immutable displays for a reason; folding "now pick a counterparty and
a role" into that same form would blur the line between "editing this
entry" and "replacing this entry with a different one." A separate route,
reached via an explicit "Transfer to self-transfer" action, keeps the
distinction visible to the user at the moment it matters.

**`?last=true` is a generic redirect-restore mechanism, used by both
redirects, not something self-transfer-specific.** `entries.index.tsx`
persists its current filter/search/sort state (the same fields already
carried in the URL — see `web-client-entries`'s "The entry ledger's filter,
search, and sort state lives in the URL" requirement) to `localStorage` on
every change. Any navigation back to the list that wants to restore "wherever
the visitor was" — the conversion's success redirect, and the ordinary
edit-save redirect, which today only reconstructs `account_id` and silently
drops every other filter — uses `/entries?last=true` instead of hand-
reconstructing search params. Resolution rule: `last` present *and no other
filter parameter present* resolves to the remembered state (or a bare,
unfiltered `/entries` if nothing has been stored yet, replacing the URL so
the `?last=true` hop doesn't linger in browser history); `last` present
*alongside* any other filter parameter is ignored outright — that
combination is never expected to be produced by anything in this app, but
if it occurs, the explicit filter wins rather than `last` silently
overriding it.

## Risks / Trade-offs

- **[Risk]** A new `id` on conversion means anything that held the
  original entry's id stops resolving to anything live — accepted, since
  it's the same experience an ordinary delete already produces, not a new
  failure mode this change introduces.
- **[Trade-off]** Asking the caller to explicitly choose sender/receiver,
  rather than inferring it from the amount's sign, adds one more decision
  to the conversion flow. Accepted because the alternative is a guess that
  can be silently wrong (a self-transfer's amount sign on `account_id`
  isn't actually constrained), which would be worse than one extra
  explicit choice.
- **[Risk]** `localStorage`-persisted filters are per-browser, not tied to
  the authenticated visitor — on a shared/public machine, a later,
  different visitor could see the previous visitor's last-applied filter
  selection (never any entry *data* itself, which is always fetched fresh
  and permission-checked server-side; only filter *choices* like which
  account or category was selected persist). Accepted as a low-severity,
  purely cosmetic leak, consistent with this app's existing `localStorage`
  usage for other per-browser conveniences (e.g. the theme preference).
