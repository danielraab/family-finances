## Context

`internal/recurringtransaction` was deliberately built without a `kind`:
its package doc and `recurring-transactions`' spec both state that a
template "always represents a `transaction`-kind entry when materialized;
there is no recurring equivalent of a balance adjustment." That reasoning
holds for `balance_adjustment` — an absolute reading is a correction, and
correcting a balance on a schedule is meaningless — but not for
`self_transfer`, which is an ordinary repeating movement of money that
happens to touch two accounts. This change reopens that one decision and
nothing else about it: `balance_adjustment` remains outside the recurring
model.

Everything a two-sided template needs already exists one package over.
`add-self-transfer` worked out the hard parts for entries: the
`entry_legs` view (a `UNION ALL` of the stored row and, for a
`self_transfer`, a second projection with `account_id`/`to_account_id`
swapped and `amount` negated), the once-or-twice listing rule that falls
out of filtering that view by `account_id = ANY(…)`, the both-accounts
`append`+ permission rule, and the same-currency requirement. This change
is mostly that same shape applied to a table whose queries are far
simpler: `recurring_transactions` has no keyset pagination, no cursor, and
one `List` query that `List`, `Summary`, and `Preview` all share.

The one genuinely new thing is a **visibility control**: unlike entries,
where both legs of a transfer always show, a `/recurring` reader is looking
at a list of standing commitments and mostly does not want a transfer
between their own accounts padding it — and when they do want it, they may
want one side or both. That is what the two flags express.

## Goals / Non-Goals

**Goals:**
- Let a standing transfer between two of the caller's own accounts be
  templated, with the same next-suggested-date, per-year amount, and
  linked-entry machinery every other template has.
- Make a `self_transfer` template unable to describe something that could
  not be booked — the same field, permission, and currency rules its
  materialized entry will face, checked at template time.
- Reuse `entry_legs`' worked-out projection rather than inventing a second
  way to be two-sided.
- Give the `/recurring` reader explicit control over whether transfers
  appear at all, and as one side or both.

**Non-Goals:**
- A recurring `balance_adjustment` — still meaningless, still absent.
- Cross-currency transfers — out of scope for entries, out of scope here.
- Changing `kind` on an existing template, in either direction. There is no
  recurring analogue of `POST /api/entries/{id}/self-transfer`; a template
  is cheap to delete and recreate, unlike an entry with a ledger position
  and a balance-adjustment chain behind it.
- Automatic creation of anything. The manual-only model is untouched: a
  self-transfer template suggests a date and prefills a form, exactly as a
  transaction template does.

## Decisions

**An explicit `kind` field, not an inferred one.** A nullable
`to_account_id` alone would be enough to tell the two shapes apart, and
would avoid two columns that must agree. Rejected: every surface that
consumes a template — the form, the row, the summary modal, the
materialization prefill — already switches on `entry.kind` for the entry
side of the same concept, and a template whose `kind` is the value its
entry will carry means the prefill copies a field rather than deriving one.
The redundancy is made safe the way `entries` already makes it safe, with a
`CHECK ((kind = 'self_transfer') = (to_account_id IS NOT NULL))`
constraint, so the two columns cannot disagree in storage. `kind` is
immutable after creation, like `entry.Kind` — no field for it on `Update`,
so `DisallowUnknownFields` rejects an attempt to set it — and so is
`to_account_id`, for the same reason its entry counterpart is: a
template's two accounts are fixed when it is written.

**Per-kind field rules copy the entry's exactly, because the entry is what
they produce.** `category_id` is required for a `transaction` template and
optional for a `self_transfer` one; `counterparty`/`location` are rejected
on a `self_transfer`. Any other choice would let a template be built that
`POST /api/entries` refuses, which the visitor would only discover at
booking time. In storage this means the current `category_id uuid NOT NULL`
becomes nullable with a conditional constraint, mirroring `entries_check`'s
own `(kind = 'transaction' AND category_id IS NOT NULL) OR (…)` shape.

**`recurring_transaction_legs`, a direct copy of `entry_legs`' idea.** The
view is the stored row with `true AS native`, `UNION ALL` the swapped,
negated projection for `self_transfer` rows with `false AS native`. Every
list mode is then one `WHERE` clause over that single view rather than
three different `FROM`s:

```
exclude (default)  →  WHERE native AND kind = 'transaction'
one leg            →  WHERE native
both legs          →  (no extra clause)
```

with `account_id = ANY(…)` — the caller's resolved visible accounts,
unchanged — applied in every mode. "Appears once per account of its two
that is in scope" falls out of that filter exactly as it does for entries,
with no mode-specific logic. Ordering gains `native DESC` as a tie-break
so a template's outgoing side sorts immediately before its incoming one
when both are listed.

**The summary always follows the list's flags.** `GET
/api/recurring-transactions/summary` takes the same two parameters and
applies the same three modes, so the per-year total under the list is
always the total of the rows in it. With both legs in scope the two
`per_year_amount`s are equal and opposite — the negation is already in the
view's `amount`, so `PerYearAmount` needs no knowledge of any of this — and
the currency nets to zero. That is the same honest answer
`GET /api/entries/summary` already gives for a real self-transfer seen from
both sides, and it is why the flags default to off: a reader asking "what
do my recurring commitments cost me a year" should not have to reason about
a zero that is really two numbers.

**Preview ignores the flags and always emits both legs.**
`GET /api/recurring-transactions/preview` is not the `/recurring` list; it
feeds the Upcoming block, the projected bar segments, and the projected
balance line on surfaces where the *real* self-transfer entries beside it
are already two-sided and unconditional. A flag there would make a
projected balance on the receiving account silently omit money the visitor
can watch leaving the sending one, which is the specific thing a projected
balance exists to show. So `Preview` queries the view with no extra clause,
and `PreviewItem` gains `kind`/`to_account_id`/`to_account_name` so a
projected row can render like the entry it would become. The asymmetry
with the list is deliberate and worth stating plainly: the flags are a
property of the `/recurring` page's reading experience, not of the domain.

**One-leg mode shows the native leg, and nothing else.** With
`include_self_transfer` on and `self_transfer_both_legs` off, only `native`
rows are listed, and the account-scope filter still applies to that row's
own `account_id`. The consequence, stated so it reads as a decision rather
than a bug: a caller who can see only the *receiving* account of a
transfer sees that template in neither mode except "both legs". The
alternative — falling back to the receiver leg when the sender is out of
scope, so a template never disappears — was rejected because it makes a
row's sign depend on which shares the reader happens to hold, so two
readers of the same list would see the same commitment as opposite
numbers. Losing a row is easier to understand than a sign that changes per
viewer, and "both legs" always recovers it.

**Creating or editing a `self_transfer` template requires `append`+ on both
accounts.** Today a template is gated only by the caller's tier on its one
parent account. For a `self_transfer` that is not enough: the template
names an account on the other side that the caller may have no standing to
put money into, and `Service.checkAccount` already exists to check exactly
this — it runs once per account instead of once. Edit requires it too, and
at edit time, mirroring `account-entries`' "editing a self-transfer
requires current permission on both accounts" rule and for the same reason:
the template's amount governs a movement into an account the editor must
still be trusted with. Delete is left alone — it keeps the existing
per-template rule against the parent account, matching the same asymmetry
`add-self-transfer` settled on for entries (edit needs both, delete needs
one), since deleting a template removes a suggestion and moves no money.

**Materialization needs no new mechanism, and the receiving leg is already
safe.** `entries.new.tsx` prefills from
`GET /api/recurring-transactions/{id}` — a single fetch by id, which always
returns the stored row in its native orientation, never a leg — so a
"Create transaction" action on a receiver-side row carries only the id and
the booking date, and the form still resolves `account_id` to the template's
sending account. That matters because `RecurringTransactionLookup.SameAccount`
validates an entry's `recurring_transaction_id` against its `account_id`
alone; booking the receiver leg with the accounts swapped would be rejected.
The prefill gains `kind` and `to_account_id` and otherwise works as it does
today. `GET /api/recurring-transactions/{id}` is deliberately left
leg-free for this reason.

**The currency lock extends to templates, via the same trick it already
uses.** `internal/account`'s `EntryLookup.HasEntries` locks an account's
`currency` once it has any entry. A `self_transfer` template pairs two
accounts and creates no entry until someone books one, so an account named
only by a template can still change currency today and break every future
booking — a hole this change would open. `internal/account` gains a second
narrow lookup, satisfied structurally by `*recurringtransaction.Service`
and wired post-construction in `main.go`, exactly as `WithEntryLookup` and
`WithUserLookup` already are. Like `HasEntries`, it is a cheap existence
check — "is this account named by any non-deleted self-transfer template,
on either side" — not a per-template currency comparison.

## Risks / Trade-offs

- **[Trade-off]** Two flags rather than one tri-state parameter. The pair
  can express `self_transfer_both_legs=true` with
  `include_self_transfer=false`, which means nothing; it is defined as
  ignored, the way `add-self-transfer-conversion` already defines `last`
  alongside an explicit filter as ignored. Accepted because the pair maps
  one-to-one onto the two checkboxes and onto two URL parameters that can
  be set independently, which a tri-state would fold into one control whose
  third value is only reachable through the second.
- **[Risk]** The list and the preview disagree about self-transfers by
  design — hidden by default in one, always shown in the other. A reader
  who turns the list flag off and then sees a transfer in the Upcoming
  block could read it as a bug. → **Mitigation**: the Upcoming block sits
  beside real entries that behave the same way, and the row is visibly a
  transfer (the other account is named on it); the alternative, threading
  the flags through every preview consumer, would make a projected balance
  wrong rather than merely surprising.
- **[Risk]** `category_id` going nullable is a storage-level loosening for
  every template, not just self-transfer ones — a bug in the conditional
  constraint would let a `transaction` template through with no category,
  which the API contract still requires. → **Mitigation**: the constraint
  is copied from `entries_check`, which has carried exactly this shape
  since migration 0030, and the domain's `validateNew`/`validateUpdate`
  keep their own check independently of it.
- **[Trade-off]** Requiring `append`+ on both accounts to *edit* means a
  template can become read-only to someone who could edit it yesterday, if
  a share on the far account is revoked. Accepted for the same reason
  `add-self-transfer` accepted it for entries, and it is worth surfacing in
  the edit form rather than leaving as a failed save.
