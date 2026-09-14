## Context

`/home` (`web-client-home`) is currently a fixed view: every account the
visitor owns or has permission on rendered as an `AccountCard`, followed by
one all-accounts `FlowChart` (income/outcome bars, year view, prev/next
pager). Nothing about it is stored — it's derived fresh from
`GET /api/accounts` + `GET /api/entries/flow-summary` on every load.

`/reports` is a live filter form (category/tag/account/date-range →
"Generate report" → `GET /api/entries` + `GET /api/entries/summary`), not a
saved concept. There is no existing "query" entity anywhere in the backend.
This change does not introduce one — each dashboard card carries its own
filter inline, and `/reports` is untouched.

Every filterable entity a card might reference (account, category, tag) is
independently shareable at `view`/`append`(+) tiers, per
`account-sharing`/`category-sharing`/`tag-sharing`. A card's filter must be
validated against the caller's actual, current access, and must keep
working sensibly if that access is later revoked.

## Goals / Non-Goals

**Goals:**
- Let each user compose their own `/home` from an ordered set of cards,
  four types: account stat, query stat, entry list, bar chart.
- Persist the layout per user, edited through explicit add/remove/reorder
  actions (no drag-and-drop), consistent with `/categories`'s established
  interaction convention.
- Validate every card's account/category/tag reference against the
  caller's real access at write time, and degrade gracefully — not
  error — when that access is later revoked.
- Reuse existing building blocks wherever they fit: `AccountCard`,
  `BarChart`/`FlowChart`, and `/reports`' account/category/tag/date-range
  filter controls.

**Non-Goals:**
- No saved/named/shareable "query" entity. A card's filter is a plain
  value on the card; it cannot be reused by another card or by `/reports`
  without re-entering it.
- No free-form drag-and-drop or per-card resizing. Card size is fixed by
  type; position is controlled only by move-up/move-down.
- No auto-migration of a user's pre-existing fixed view into cards. Every
  dashboard starts empty; see "Migration Plan".
- No configurable entry-list page size — fixed at 10, matching the
  original ask; a config knob can be added later if wanted.
- No cross-user dashboard sharing. A card can point at a shared account/
  category/tag, but the card itself, and the layout it belongs to, is
  always private to its owning user.

## Decisions

### One row per card, not one JSON blob per user

`dashboard_cards` (new table): `id`, `user_id`, `sort_order`, `type`,
`config JSONB`, timestamps. Mirrors the `tags`/`categories` shape (a list
of independently addressable entities) rather than `user_settings`'s
single-row-per-user shape. This matches the chosen edit model
(move-up/move-down, add one, delete one) exactly the way
`POST /api/categories/{id}/move-up`/`/move-down` already works, and avoids
a full-layout read-modify-write race between, say, two open tabs.

`internal/dashboard` is a new domain package in the standard four-file
shape (`dashboard.go`, `service.go`, `store.go`, `handler.go`), store
implementations in both `storage/memory` and `storage/postgres`, migration
`0027_dashboard_cards.sql`.

Endpoints:
```
GET    /api/dashboard/cards                 list caller's cards, sort_order asc
POST   /api/dashboard/cards                 create { type, config }
PATCH  /api/dashboard/cards/{id}             update { config } (type is immutable)
DELETE /api/dashboard/cards/{id}
POST   /api/dashboard/cards/{id}/move-up
POST   /api/dashboard/cards/{id}/move-down
```
`move-up`/`move-down` are a no-op (`200`, unchanged) at either end of the
list, exactly like the category sibling-reorder endpoints.

**Alternative considered**: one `dashboard_layout` row per user (JSON
array of cards), matching `internal/settings`. Rejected — it turns every
single-card add/remove/reorder into a full-array read-modify-write, and
this domain (a growable list of independent, orderable entities) already
has a house pattern that doesn't have that problem.

### `type` is immutable; `config` shape is a plain object, not a discriminated union in the API

`type ∈ {account_stat, query_stat, entry_list, bar_chart}`. Changing a
card's purpose is delete-and-recreate, not an edit — matching how nothing
else in this codebase lets an entity's fundamental kind change in place
(an entry's `kind`, similarly, is fixed at creation).

`config` is one flexible JSON object; which fields are meaningful depends
on `type`, unenforced by the OpenAPI schema itself (kept as a single
object type, not a `oneOf`, matching this API's general preference for
simplicity — see `icon`/`color` on accounts/categories being similarly
untyped at the schema level). Server-side validation (not just schema
validation) still enforces the required/allowed fields per type:

| type | config fields |
|---|---|
| `account_stat` | `account_id` (required) — no `title` |
| `query_stat` | `account_id?`, `category_id?`, `include_subcategories?` (default `true`), `tag_id?`, `range?`, `title?` |
| `entry_list` | same optional filter fields as `query_stat` (no `account_id`-only special case — omitted filters mean "every account/category/tag the caller can see", exactly like `/reports` with nothing selected), plus `columns?` (`2`-`4`, default `2`) — see "Layout" below |
| `bar_chart` | `account_id?`, `category_id?`, `include_subcategories?`, `tag_id?`, `title?`, `unit` (required, `month` \| `day`) |

`title` (see "Card heading links to /reports" below) is the one config
field shared by exactly the three filter-bearing types and no others —
`account_stat`'s heading is always its account's own name.

`range` (on `query_stat`/`entry_list`) reuses `/reports`' own shape
verbatim: `{ preset?: PresetKey, from?: date, to?: date }`
(`frontend/src/lib/dateRangePresets.ts`). The backend never interprets
`preset` — it's resolved to concrete `from`/`to` timestamps client-side,
at render time, via the same `resolveEffectiveRange` `/reports` already
uses, immediately before calling `/api/entries/summary` or
`/api/entries`. This is what makes a "this month" card stay live across
days/months rather than freezing the range it was created with, at zero
backend cost.

`bar_chart` intentionally has no `range` — it follows `FlowChart`'s
existing model instead (`unit=month` → 12-bucket year view with a
prev/next **year** pager; `unit=day` → a month's daily buckets with a
prev/next **month** pager). Which year/month is currently displayed is
ephemeral UI state, not persisted on the card, exactly like today's
`FlowChart` always opens on the current year.

### Server-side validation of card config

At create and update, `internal/dashboard` validates every id in `config`
against the caller's actual access, structurally mirroring
`internal/entry`'s `AccountLookup`/`CategoryLookup`/`TagLookup` pattern
(flat-value interfaces to avoid import cycles):

- `account_id` (any type): the caller must have **any** permission tier on
  that account (`Access(...) != ""`) — filtering only needs visibility,
  not write access, same bar `/reports`' account filter clears today.
- `category_id`: the caller must have `view`+ visibility (owned, or
  shared at `view` or `append`) — matches what `/reports`' category
  filter already allows picking, wider than `entry.Usable`'s `append`+
  bar (which governs *selecting* a category on an entry, not filtering by
  one).
- `tag_id`: same `view`+ bar, mirroring `tag-sharing`.
- A missing/invalid id, or a field the given `type` doesn't use, is
  `ErrInvalidValue` (`400`) — the same sentinel-error-to-status mapping
  every other domain package uses.
- `unit` on `bar_chart`: must be `month` or `day`, same validation
  `flow-summary` itself already does.

This is a one-time check at write time, not re-verified on every render —
identical in spirit to how a category's `Usable` check only fires when
`category_id` is newly set on an entry, never on an unrelated update.

### Stale references degrade to a placeholder, never an error

Access to an account/category/tag can be revoked after a card is created
(the referenced entity's owner un-shares it, or it's soft-deleted). The
dashboard-cards endpoints don't re-validate on every `GET` — a listed
card's `config` ids are returned as-is. The **frontend** resolves them
against `GET /api/accounts` / `GET /api/categories` / `GET /api/tags`
exactly as `/reports` already does today for an entry's account
(`entries.notShared`), and when an id doesn't resolve, the card renders a
"no longer accessible" placeholder (translated, card-shaped, so the grid
doesn't jump) instead of erroring or silently vanishing. The card stays in
the layout — the visitor removes it explicitly via the same remove action
as any other card, in edit mode.

**Alternative considered**: have the backend null out or auto-delete a
card when its reference goes stale (e.g., a hook on share revocation).
Rejected — no other domain package in this codebase reaches into another
one's data on a lifecycle event (categories/tags already accept the
weaker "stale reference stays until something touches it again" model for
disabled categories/tags on existing entries), and it would require
`internal/account`/`internal/category`/`internal/tag` to know about
`internal/dashboard`, violating the one-way dependency rule.

### Layout: full-width chart cards, 3-4 per row otherwise, entry_list width configurable

`bar_chart` cards always render one per row, full width (a 12-bar chart
is illegible squeezed into a third of the row). Every other type flows in
a responsive grid extending the same breakpoint pattern the current
`/home` already uses for `AccountCard`
(`grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4`). Card
position among same-row types is controlled only by sort order; there is
no independent column/row placement.

`entry_list` is the one type whose width is itself configurable —
`config.columns` (`2`-`4`, default `2`) sets how many of the grid's
columns it spans, via a literal, statically-written Tailwind class per
value (`sm:col-span-2 lg:col-span-2 xl:col-span-2` for `2`, and so on) —
never a runtime-constructed class string, since Tailwind's scanner only
emits classes it can see literally in source. `account_stat`/`query_stat`
stay fixed at one column (small stat tiles gain nothing from more room);
`bar_chart` stays always-full-width regardless of any span, since its own
"one per row" rule already subsumes it. The picker on the add-card form
offers three named widths (Narrow/Wide/Full width, i.e. 2/3/4 columns)
rather than a raw number input, matching this app's general preference
for named options over free-form numeric fields in a form.

### Edit mode: button-driven, no drag-and-drop

An "Edit" toggle on `/home` reveals, per card, ▲/▼ move buttons (disabled
at either end of the whole list, the same accessibility affordance
`/categories`'s reorder buttons already use), an edit action, and a
remove action, plus an "Add card" control that opens a type picker
followed by that type's config form. The config forms reuse `/reports`'
existing account/category/tag `<select>`s and `DateRangeFilter` component
directly rather than rebuilding equivalent controls. The per-card control
toolbar shares the card's own border/rounding with zero gap
(`overflow-hidden` wrapper), rather than floating as a visually separate
box above it, so which card a control acts on is unambiguous.

**Editing reuses the add-card form** (`CardFormDialog`, generalized from
what was originally `AddCardDialog`) rather than a second dialog: passing
it the card being edited locks the (immutable) type selector and seeds
every other field from the card's current `config`; submitting calls
`PATCH` instead of `POST` and replaces the card in place rather than
appending a new one. This avoids maintaining two parallel form
implementations for what is otherwise identical field-by-field UI.

### Card heading links to /reports

`query_stat`/`entry_list`/`bar_chart` cards accept an optional `title` in
`config` (never `account_stat`, whose heading is always its account's own
name and already links to that account's details page). The card's
heading — the custom `title` when set, else the same generated filter
summary shown before this existed — is always a link to `/reports`,
prefilling its account/category/include-subcategories/tag/date-range
controls from the card's own filter (mapped in one place,
`cardReportsSearch` in `lib/dashboardFilter.ts`; a `bar_chart` card's
`unit`/`columns` have no `/reports` equivalent and are dropped).
Activating it never auto-runs the report — `/reports` still requires its
own "Generate report" click, exactly as it does for a visitor who
navigates there directly, so this is strictly a convenience for jumping
into `/reports` with less re-entry, not a shortcut around it.

**Alternative considered**: auto-generate the report on arrival (skip
the "Generate report" click) when arriving from a card's heading link.
Rejected — it would make `/reports`' behavior context-dependent on how
the visitor arrived, and the one extra click costs little given the
filter is already prefilled.

### Two empty states

- Zero accounts at all → today's existing "create your first account"
  empty state, unchanged (`web-client-home`'s current empty-state
  requirement).
- One or more accounts, zero cards → a new "add your first card" prompt,
  shown only once at least one account exists (so a brand-new user isn't
  told to build a dashboard before they have anything to point it at).

## Risks / Trade-offs

- **[Risk]** Card config drifts out of sync with entity state (renamed
  account, category moved/reparented) between loads → **Mitigation**:
  the frontend always resolves ids against the live `GET /api/accounts`/
  `/categories`/`/tags` response on each render, the same way every other
  page in this app shows current entity state rather than a cached
  snapshot; nothing about a card is denormalized/cached.
- **[Risk]** `config`'s per-type validation logic (a plain JSON object,
  not a typed union) could accept a config that's valid JSON but
  meaningless for its `type` (e.g., a `bar_chart` missing `unit`) →
  **Mitigation**: `internal/dashboard`'s service layer validates the
  exact required/allowed field set per `type`, unit-tested per type, the
  same way `internal/entry` validates `kind`-dependent required fields
  (`balance` XOR `amount`) today.
- **[Trade-off]** No saved/shared query entity means the same filter
  can't be defined once and reused across cards or with `/reports` — a
  visitor who wants "this month, groceries category" on three different
  card types re-enters that filter three times. Accepted for this change;
  a future saved-query capability could let a card reference one by id
  without changing `dashboard_cards`'s shape (`config` would just gain a
  `query_id` alternative to its inline fields).

## Migration Plan

- Migration `0027_dashboard_cards.sql` only adds a new table — no
  destructive change to existing data.
- **No backfill.** Every existing user's `dashboard_cards` starts empty on
  deploy, per the proposal's explicit choice to ship empty rather than
  auto-seed today's fixed view. `/home` immediately shows the new
  "add your first card" empty state for every current user until they
  configure it themselves.
- Rollout is a single deploy; no feature flag — this fully replaces the
  old `/home` rendering path in the same change (old fixed-view code is
  deleted, not dual-run).
- Rollback: revert the deploy. The new table is additive, so rolling back
  the binary to a pre-change version leaves `dashboard_cards` unused but
  harmless; no data migration needs reversing.

## Open Questions

- None outstanding — layout, persistence shape, edit interaction, empty
  states, and validation strategy were all resolved during exploration
  before this proposal was written.
