## Context

`internal/entry` (`backend/internal/entry/entry.go`) models an entry as
belonging to exactly one account: `Kind` is a closed two-value type
(`transaction`, `balance_adjustment`), `Balance(asOf)` is a single
`SUM(amount) WHERE account_id = X AND booking_timestamp <= asOf`, the
balance-adjustment recompute algorithm walks one account's adjustment chain
after a mutation, and `GET /api/entries`'s cursor (`BookingTimestamp,
Amount, ID`) assumes one stored row produces exactly one output row. A
self-transfer entry — one row, two accounts — breaks all four assumptions on
purpose, in exchange for the two properties a two-entries-linked-by-id
design (considered and rejected during exploration — see the change's
conversation history) can't offer for free: atomicity (one insert, no
partial-failure window) and consistency (one amount/date/category, so the
two "sides" can never disagree; deleting the row removes both legs).

`internal/account` (`backend/internal/account/account.go`) has no notion of
`internal/entry` today — `Update.Currency` is validated only for ISO-4217
shape (`settings.ValidateCurrency`), with no check against existing entries.
Dependencies in this codebase flow one way (domain packages import none of
each other; a reverse need is expressed as a narrow interface the dependent
package declares and the other package satisfies structurally — see
`entry.AccountLookup`/`entry.CategoryLookup`/`entry.TagLookup`, each
satisfied by `*account.Service`/`*category.Service`/`*tag.Service` without
an import cycle, wired via a `WithXxx` setter in `main.go` when the
dependency can only be resolved after both services exist). Locking an
account's currency once it has entries needs the same trick in the other
direction: `internal/account` declaring the interface this time, satisfied
by `*entry.Service`.

## Goals / Non-Goals

**Goals:**
- Represent a transfer between two accounts as one entry row, consistent by
  construction (one amount, one category, one booking timestamp).
- Reuse every existing account-permission, category/tag, and listing
  mechanism `internal/entry` already has rather than inventing parallel
  ones — a self-transfer is a `Kind`, not a new domain package.
- Make the two-sided nature of a self-transfer's permission and rendering
  rules explicit and testable, since they're the one genuinely new shape
  here (everything else is variations on what `transaction` already does).

**Non-Goals:**
- Cross-currency transfers (no FX rate, no second amount) — out of scope
  entirely, not merely unvalidated.
- Transfers to an account the caller doesn't have at least `append` on —
  this is not a payment/P2P feature between different users' private
  money; both sides must be within the caller's own writable reach.
- Retroactively fixing currency mismatches on accounts that already exist
  with entries — the immutability rule only prevents new mismatches from
  this point forward.

## Decisions

**`Kind` gains a third value, `self_transfer`, with a new `ToAccountID
*string` field on `Entry`/`New`/`Update`.** `ToAccountID` is required
exactly when `Kind == KindSelfTransfer` (mirrors the existing
`Balance`-required-exactly-for-`KindBalanceAdjustment` shape in
`validateNew`) and rejected otherwise. `AccountID` remains the sending side,
`Amount` remains the signed delta already defined for `transaction` (applied
to `AccountID`'s balance; the receiving side sees `-Amount`).
`ToAccountCurrency`/`ToAccountName` are resolved server-side unconditionally
— the same reasoning `Entry.AccountCurrency`/`CreatedByName`'s doc comment
already gives for why those are resolved regardless of the viewing caller's
access: a viewer with only `append` on the sending account still needs to
see which account the money went to and in what currency, even without any
permission on that account at all.

**Category is optional for `self_transfer`; a self-transfer's amount is
included in `Sum`/`FlowSummary`.** `validateNew`'s "`Kind == KindTransaction
&& CategoryID == nil` is invalid" check narrows to `KindTransaction`
specifically, not "not a balance adjustment." `Sum`/`FlowSummary`'s existing
`kind != 'balance_adjustment'` exclusion stays exactly that — a
`self_transfer` is not additionally excluded, since (unlike an absolute
balance reading) it's a real movement of money into or out of an account and
belongs in an income/outcome picture the same way a `transaction` does; a
transfer viewed across both its accounts together nets to zero by
construction, which is the honest answer for "does this net worth", while a
single-account view honestly shows the outflow/inflow.

**Create requires `append`+ on both `AccountID` and `ToAccountID`, plus
matching `Currency`.** `Service.Create`'s existing account/disabled check
(the one `move-entry-between-accounts`'s design.md already extracted into a
shared helper) runs once per account; a currency mismatch is a new
`ErrInvalidValue` (`400`) case, checked only for `self_transfer`.

**Edit requires `append`+ still held on both accounts at edit time; a
self-transfer with insufficient current access on either side is
read-only.** This is stricter than `transaction`'s edit rule (`append`
edits only what they created; `entry_admin`/`owner` edits anything on their
account) precisely because a self-transfer edit necessarily touches balance
on both accounts at once — an editor who has lost access to one side can no
longer be trusted to move money in or out of it, even indirectly by editing
the shared amount. **Delete keeps the plain per-account rule** (checked
against whichever side the caller still holds `entry_admin`/`owner` on) —
deleting doesn't require *both* sides' permission because it doesn't
require agreeing on a shared value going forward; it only removes a row the
caller already has sufficient standing to remove on their own side. This
asymmetry (edit needs both, delete needs one) is the one place this
capability's authorization rule doesn't just mirror `transaction`'s, so it
gets its own explicit scenarios in the spec.

**`GET /api/entries` becomes, functionally, a `UNION ALL` of two
projections of the same table** — the existing single projection
(`account_id` as stored, `amount` as stored) plus a second one, active only
for `self_transfer` rows, keyed by `to_account_id` with `amount` negated.
Both projections go through the same `WHERE`/filter/sort/cursor pipeline as
one combined result set. This is what makes "renders twice when both
accounts are in scope, once when only one is" fall out naturally: a
`self_transfer` row satisfies the first projection's `account_id = ANY(…)`
when its sender is in scope and the second projection's `to_account_id =
ANY(…)` when its receiver is in scope, independently — both can be true at
once. The per-row identity used for the keyset cursor and the JSON `id`
gains a second, implicit dimension (which projection produced it); the
existing `(BookingTimestamp, Amount, ID)` tie-break already disambiguates
the two projections of one row without changes, since they carry
opposite-signed `Amount`. The listing response marks a row produced by the
second (receiver) projection with a `direction`/leg indicator so the client
can render the flipped sign and the "other side" link correctly — exact
field name and shape is an implementation detail for `tasks.md`, not a
product decision.

**Recompute-on-mutation runs once per side.** Creating, editing, or deleting
a `self_transfer` triggers the existing balance-adjustment recompute
(`recomputeFrom`/`findAdjustment`/`setAmount`) independently for
`AccountID` and for `ToAccountID`, using the same "only the earliest
adjustment at/after the mutated position, plus the one after it, can ever
need a new amount" argument already proven for the single-account case —
nothing about that argument depends on there being only one account
involved, since each account's own adjustment chain is unaffected by what
happens on the other account.

**`internal/account` gains a narrow `EntryLookup` interface** —
`HasEntries(ctx, accountID) (bool, error)` — satisfied structurally by
`*entry.Service`, wired via `account.WithEntryLookup(entrySvc)` in
`main.go` after both services are constructed (mirroring
`account.WithUserLookup`/`category.WithMailer`'s existing post-construction
wiring for the same reason: `entry` already imports `account`, so the
dependency can only run this direction as an interface, never a direct
import). `Service.Update` rejects a `Currency` change when `HasEntries`
reports `true`, regardless of the new value's shape or whether it matches
the old one byte-for-byte — the rule is "has any entry," not "would this
particular change conflict with anything," which keeps it a single cheap
existence check rather than a per-entry currency-consistency scan.

## Risks / Trade-offs

- **[Risk]** The listing rewrite (`UNION ALL` of two projections) is the
  largest single piece of implementation risk in this change — it touches
  `buildWhere`, cursor encoding, and the in-memory store's mirror
  implementation, none of which have ever had to produce more result rows
  than underlying storage rows. → **Mitigation**: keep the two projections
  structurally identical (same columns, same filters applied to each side
  independently) so `buildWhere`'s existing per-filter logic is reused
  verbatim on both halves of the union rather than duplicated with
  variations.
- **[Risk]** Locking `currency` once an account has entries is a behavior
  change for existing accounts, not just new self-transfer users — anyone
  who was relying on being able to fix a typo'd currency after logging a
  few entries loses that. → **Mitigation**: called out as **BREAKING** in
  the proposal; no migration path is offered because there's no reasonable
  automatic fix (changing a currency after the fact silently reinterprets
  every existing entry's amount in a new denomination, which was already
  wrong before this change — this closes that gap rather than opening a
  new one).
- **[Trade-off]** Delete needing only one side's permission, while edit
  needs both, is an asymmetry a caller could find surprising ("I can
  delete this but not fix a typo in its title"). Accepted deliberately —
  see the Decisions section — but worth calling out prominently in the
  edit form's read-only state so it doesn't read as a bug.
