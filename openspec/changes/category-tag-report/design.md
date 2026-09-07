## Context

See `proposal.md` - Why. Relevant existing shape:

- `internal/entry.Filter` already carries `CategoryID`/`CategoryIDs` and
  `TagID`; `Service.List` resolves `CategoryID` via
  `CategoryLookup.Subtree` (always — there is no exact-only mode) before
  calling `Store.List`. `Store.List` (memory and postgres) filters on the
  already-resolved `CategoryIDs`.
- No aggregate query exists anywhere. `account.Service.Balance` is the one
  precedent for a live-computed (never cached) numeric result, but it's a
  single account/single currency by construction; this is the first
  place amounts are summed across accounts of potentially different
  currencies.
- The frontend has no query library — `entries.index.tsx` fetches directly
  in `useEffect`s keyed off a `JSON.stringify(search)` dependency, firing
  on every filter change. `/reports` deliberately does not do this.

## Goals / Non-Goals

**Goals:**
- Reuse `internal/entry`'s existing filter-resolution logic (account
  visibility, category subtree/exact resolution, tag ownership scoping)
  identically between the list and the new sum, so they can never disagree
  about which entries match.
- Keep `GET /api/entries`'s current behavior byte-for-byte unchanged for
  every existing caller when `category_mode` is omitted.
- Make the sum correct without transferring every matching row to the
  client — compute it in SQL.

**Non-Goals:**
- No currency conversion. Multi-currency results are shown as separate
  per-currency sums, never combined.
- No new pagination model for the summary endpoint — it returns one small,
  aggregated payload, not a page of rows.
- Not touching `/entries`'s own UI or its `category_id` semantics; it
  continues to omit `category_mode` and gets today's subtree behavior.

## Decisions

**`category_mode` as a new, additive query param, not a new filter field
name.** Alternative considered: repurpose `category_id` to mean "exact"
and require a separate `category_subtree_id` for the old behavior. Rejected
because it would silently change `/entries`'s existing behavior (it never
sends anything beyond `category_id`) unless that page is also touched — out
of scope per the proposal. An additive, default-preserving param keeps the
blast radius to exactly the new code path.

**`Filter` gains a `CategoryMode` (or equivalent enum) alongside the
existing `CategoryID`.** `Service.List` (and the new `Service.Sum`) branch
on it: `exact` sets `CategoryIDs = [id]` directly; `subtree` (or unset)
keeps calling `categories.Subtree` as today. Both entry points share one
private resolution helper so the two never diverge.

**New `Store.Sum(ctx, ownerID, Filter) (map[string]int64, int, error)`
method, implemented in both `storage/memory` and `storage/postgres`.**
Alternative considered: compute the sum in Go by calling `Store.List`
without a limit and adding in the service layer. Rejected — that's exactly
the "page through everything" approach the proposal exists to avoid, and
duplicates the correctness (not just performance) concern for any owner
with more entries than fit in memory. The postgres implementation is
`SELECT accounts.currency, SUM(entries.amount), COUNT(*) ... GROUP BY
accounts.currency`, reusing the same joins/filters `List` already builds.
`kind = 'transaction'` is forced into the query itself, not left to the
caller, so the endpoint can never be called in a way that includes balance
adjustments.

**`GET /api/entries/summary` as a new endpoint on the existing `entry`
package/handler**, not a new domain package. It's the same filter
vocabulary and the same ownership/visibility rules as `GET /api/entries`;
splitting it into a separate package would duplicate the account-visibility
and category/tag resolution logic for no benefit.

**Frontend: `/reports` keeps filter state as local/URL "draft" state,
separate from a `generatedQuery` value that only updates when "Generate
report" is clicked.** The entries list and summary fetches key off
`generatedQuery`, not off the live filter controls — mirroring the
`searchKey` pattern `entries.index.tsx` already uses for `/entries`, but
decoupling it from the URL-backed draft state instead of using the URL
state directly as the fetch trigger. Arriving with filters already in the
URL populates the draft controls but leaves `generatedQuery` unset until
the button is clicked, satisfying "never auto-fetch."

**Reuse, not extract, `entries.index.tsx`'s table/infinite-scroll
markup.** Alternative considered: extract a shared `<EntryTable>`
component now. Deferred — `/reports`'s list is read-only (no create link,
no in-row sort-by-column affordance tied to a live-fetching page) and
`/entries` isn't being touched in this change; pulling out a shared
component is a reasonable follow-up once both pages exist side by side.

## Risks / Trade-offs

- [Two round trips per "Generate report" click (list + summary), so a
  transient failure of one but not the other is possible] → Treat them as
  independent requests with independent loading/error states; a failed
  summary fetch doesn't block showing the list, and vice versa.
- [`Store.Sum`'s postgres query duplicates `List`'s filter-building SQL
  fragments] → Factor the shared `WHERE`/join construction into a helper
  both `List` and `Sum` call, so a future filter addition can't update one
  and forget the other.
- [Per-currency sums could surprise a user with many small-balance foreign
  transactions into thinking totals are missing] → Each sum is labeled
  with its currency; no further mitigation needed given the explicit
  design decision against conversion.
