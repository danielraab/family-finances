## Context

`AccountForm.tsx` is the shared create/edit form for `/accounts/new` and
`/accounts/{id}/edit`. Today it fetches only `GET /api/account-types` on
mount; `financial_institute` is a plain, optional, free-text `<input>` with
no history or suggestion behavior. `GET /api/accounts` (already called by
`accounts.index.tsx` and `entries.new.tsx`) is not paginated and is scoped
server-side to the caller's own, non-deleted accounts (disabled accounts
included, soft-deleted ones excluded — see `accounts`'s soft-delete
requirement) — exactly the population a "have I used this institute
before" suggestion needs, with no new endpoint.

The codebase already has a precedent for a free-text field with
type-to-match suggestion chips: `TagInput.tsx`, used by the entry form's
tag input. It filters the visitor's existing tags by substring match
against the current draft and renders up to 6 as clickable pills below the
input, adding the clicked tag's name to a multi-value array.

## Goals / Non-Goals

**Goals:**

- Surface the visitor's own previously-used `financial_institute` values as
  clickable suggestions on the account create/edit form, so adding another
  account at a familiar institute doesn't require retyping (or
  misremembering) its exact spelling.
- Keep the field free text — a new institute name is always accepted, the
  suggestions are a convenience, not a constraint.
- No backend change: derive everything client-side from data the frontend
  already has access to via an existing endpoint.

**Non-Goals:**

- No new backend endpoint, no `financial_institute` normalization/dedupe on
  the server, no change to the field's validation (still optional,
  unconstrained text).
- No cross-user suggestions — accounts are strictly single-owner today (per
  `accounts`), so only the caller's own accounts are ever a source.
- No fuzzy/case-insensitive merge of near-duplicate spellings (e.g. "N26"
  vs. "n26" both existing) — dedupe is exact string. If a visitor has been
  inconsistent, both variants show as separate suggestions; cleaning that up
  is a data-entry concern, not this change's.
- No most-recently-used or most-frequently-used ordering — alphabetical
  only.
- No reusable/extracted "suggestion chips" component. `TagInput` stays as
  is; this is a second, independent, single-value implementation inline in
  `AccountForm.tsx`, not a shared primitive — the two differ enough
  (single-value replace vs. multi-value append, always-visible-on-focus
  vs. type-to-reveal) that forcing a shared abstraction over two call sites
  isn't worth it.

## Decisions

### Decision: derive suggestions from `GET /api/accounts`, fetched by `AccountForm.tsx` itself

`AccountForm.tsx` adds a second parallel fetch (`Promise.all` alongside the
existing `GET /api/account-types` call) rather than expecting a caller to
pass the account list in as a prop. This matches how the form already
self-fetches account types, and keeps `accounts.new.tsx` /
`accounts.$accountId.edit.tsx` unchanged. Account counts here are small
(no pagination on this endpoint at all), so an extra fetch on every form
mount is cheap and not worth caching or lifting.

The suggestion list is derived with a plain `Array.from(new Set(...))`
over the fetched accounts' `financial_institute` values, dropping empty
strings, sorted alphabetically (locale-aware `localeCompare`). No
memoization beyond React's normal render — the list is tiny and recomputed
only when the fetched accounts change (once, on mount).

### Decision: suggestions show on focus, not only once typing starts

`TagInput` hides its suggestions until `draft.trim()` is non-empty, sensible
when a visitor's tag catalog can grow large. A visitor's distinct financial
institutes are typically a handful at most, and the goal here is recall
("what did I call this bank last time?"), not confirmation of a name
already half-typed — so the chip row is shown whenever the field has focus,
listing every suggestion when the field is empty and narrowing by
case-insensitive substring match as the visitor types. It hides on blur,
same as any other transient helper UI in the form.

### Decision: a chip click replaces the field's value, not appends

Unlike `TagInput`'s array-valued `onChange`, `financial_institute` is a
single string. Clicking a suggestion chip calls the form's existing
`set("financial_institute", value)` directly — no array bookkeeping, no
"already selected" filtering (a single field can't have more than one
value selected already). The account currently being edited already has
its own value in the fetched list; clicking it is a harmless no-op, so no
self-exclusion is added.

### Decision: no dedicated component — inline state in `AccountForm.tsx`

The chip row is a handful of lines: derived `suggestions` array, a
`focused` boolean (or CSS `:focus-within`-driven visibility — implementation
detail, whichever reads simpler in the final diff), and the chip buttons
themselves. Given there would be exactly one consumer and the interaction
differs from `TagInput` in the ways above, this stays inline rather than
becoming a new shared component — consistent with the repo's stated
preference against premature abstraction for a single call site.

## Migration Plan

No data migration — purely additive client-side behavior.

1. `AccountForm.tsx`: add the `GET /api/accounts` fetch, derive
   `institutes` (sorted, deduped, non-empty), track focus state for the
   financial institute field, render the chip row filtered by the current
   draft value, wire chip clicks to `set("financial_institute", ...)`.
2. i18n: add any new copy only if the implementation ends up needing
   visible text beyond the institute names themselves (e.g. none is
   expected, but confirm during implementation).

## Verification

- `pnpm lint && pnpm exec tsc && pnpm build` from `frontend/`.
- Manual pass: open `/accounts/new` for a visitor with two or more
  existing accounts sharing a `financial_institute` value — confirm
  focusing the field shows that value as a chip, clicking it fills the
  field, and typing narrows the chip list by substring (case-insensitive).
  Confirm a visitor with no accounts (or none with a `financial_institute`
  set) sees no chip row. Confirm `/accounts/{id}/edit` behaves the same,
  including for the account being edited itself. Confirm typing a brand
  new institute name not in the list is still accepted on submit.
