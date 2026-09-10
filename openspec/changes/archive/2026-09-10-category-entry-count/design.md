## Context

`internal/storage/postgres/tag.go` already exposes a per-tag `entry_count`
by folding a correlated subquery into `tagCols`, the shared column list
every tag query selects. `scanTag` reads it into `tag.Tag.EntryCount`
(`json:"entry_count"`), and `internal/storage/memory`'s tag store leaves it
`0` — an accepted gap, since `storage/memory` is test/local-dev only. The
web client renders it on the Tags settings tab.

Categories have all the same moving parts (`categoryCols` + `scanCategory`
in `internal/storage/postgres/category.go`, a `category.Category` domain
type, a `/categories` tree view) but no equivalent count. This change adds
one along exactly the tag precedent.

## Goals / Non-Goals

**Goals:**

- `Category` API responses carry `entry_count` — direct `entries.category_id`
  references, the caller's own, non-deleted.
- `/categories` shows each node's count.
- Reuse the tag pattern verbatim: one correlated subquery in the shared
  column list, no new endpoint, no `JOIN … GROUP BY`.

**Non-Goals:**

- Rolled-up / subtree counts. The tree already shows nesting; a per-node
  direct count is what mirrors tags and is unambiguous. A subtree total can
  be a later change if wanted.
- Counting on the in-memory store. It has no entries table visibility; it
  reports `0`, same as its tag store.
- Any change to entry, account, or reports behavior.
- A database migration — nothing is stored; `entry_count` is derived per
  request.

## Decisions

### Correlated subquery in `categoryCols`, not a JOIN

Append to `categoryCols`:

```sql
(SELECT count(*) FROM entries e
 WHERE e.category_id = categories.id AND e.deleted_at IS NULL)
```

`scanCategory` gains a trailing `&c.EntryCount`. Every existing category
query (`List`, `Get`, `Create`, `Update`, `SetDisabled`, `move`) picks it
up for free because they all select `categoryCols` and route through
`scanCategory`. Rationale: identical to `tagCols`; a `LEFT JOIN … GROUP BY`
would risk row duplication across the other subquery-free columns and force
every call site to change.

The subquery does not re-scope by `owner_id`: `entries.category_id` can only
point at a category with the same `owner_id` (an entry's category must be
owned by the entry's account owner, enforced at write time), and every
category query already filters `categories.owner_id = $1`. This matches
`tag.go`, which likewise scopes only through the tag row, not the entry.

### Domain type

`category.Category` gains `EntryCount int` with `json:"entry_count"`,
placed last, mirroring `tag.Tag.EntryCount`. No service-layer logic — it is
pure passthrough from the store.

### Memory store reports 0

`internal/storage/memory`'s category store constructs `category.Category`
values without entry visibility; `EntryCount` stays its zero value. A brief
doc comment notes the parity with the memory tag store. Handler tests that
run on the memory store therefore assert `entry_count` is present and `0`,
not a specific positive number; the real count is covered by a
`storage/postgres` integration test.

### OpenAPI

Add to the `Category` schema in `openapi/openapi.yaml`:

```yaml
entry_count:
  type: integer
  description: >-
    The number of the caller's own non-deleted entries directly
    categorized under this category. Direct references only — entries
    under a descendant category are not counted.
```

Add `entry_count` to the schema's `required` list (mirroring `Tag`, whose
`entry_count` is required). Regenerate both committed artifacts in the same
commit: `cd backend && go generate ./...` and
`cd frontend && pnpm generate:api`. The CI `contract` job fails on drift.

### Frontend rendering

In `categories.tsx` `renderNode`, add a small count indicator in the
existing left-hand `flex items-center gap-2` cluster, after the
active/disabled status pill — a non-interactive `<span>` styled like a
neutral chip (reuse the disabled-pill's neutral zinc classes, no emerald).
Wrap the number with an i18n key so it can carry a label/`title` for
screen readers, e.g. `categories.entryCount` → `"{{count}} entries"` with
i18next plural forms (`categories.entryCount_one` / `_other`). Add the keys
to `en.json` first, then `de.json`.

## Risks / Trade-offs

- **Per-row subquery cost on `GET /api/categories`** → Same shape and scale
  as the existing tag subquery; a category tree is small (tens of rows) and
  `entries(category_id)` is the FK column. If it ever matters, an index on
  `entries(category_id) WHERE deleted_at IS NULL` is the fix — out of scope
  here, and not done for tags either.
- **Count excludes subtree, which some users may expect** → Documented in
  the OpenAPI description and the spec; matches the tag mental model. Can be
  revisited without a breaking change (the field's meaning would widen, not
  its type).
- **Memory-store `0` could mask a real regression in handler tests** →
  Accepted, identical to the tag precedent; the postgres integration test
  is the real coverage for the count value.

## Migration Plan

No migration. Ship backend + regenerated contract + frontend together. The
new field is additive; older frontends ignore it. Rollback is a plain
revert — nothing persisted.
