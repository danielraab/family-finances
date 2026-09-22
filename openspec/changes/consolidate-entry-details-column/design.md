# Design

## Context

See `proposal.md` for the motivation. `entries.index.tsx` currently renders
Account, Category, and Tags as separate adjacent table columns. The route
already resolves the accounts, categories, and tags needed for each row and
uses `AccountLabel`, `CategoryLabel`, and `TagLabel` to preserve entity icon,
sharing, and unavailable-entity behavior.

## Goals / Non-Goals

**Goals:**

- Give entry titles more horizontal space by replacing three metadata columns
  with one Details column.
- Preserve all established metadata semantics while making the per-entry
  hierarchy easy to scan.
- Keep the translated header and table-column order explicit.

**Non-Goals:**

- Redesign the ledger into cards or change its existing horizontal-overflow
  behavior on narrow viewports.
- Change fetching, filters, sorting, entry data, permissions, or API contracts.
- Introduce a generic metadata component for other views.

## Decisions

### Compose the existing labels in the route

The Details cell will compose the existing `AccountLabel`, `CategoryLabel`,
and `TagLabel` render paths rather than introduce new labels or data shaping.
This preserves entity icons, shared-owner badges/tooltips, and the established
`notShared` fallbacks exactly where the user sees them.

The alternative, a new reusable entry-metadata component, would add an API and
abstraction for a layout that currently has one consumer. It can be extracted
later only if another entry listing requires the identical presentation.

### Use a vertical account → category → tags hierarchy

Details will be a vertical stack: account first, category second when set, and
a wrapping tag group last when tags exist. Account provides the primary ledger
context; category refines it; tags can be numerous and therefore form the
least rigid group. Missing category and tags will not render em-dash rows,
keeping entries without optional metadata compact.

The alternative, a single inline row, would make long labels and several tags
compete for width and undermine the title-space improvement.

### Keep Details directly after Date

The header order will be selection checkbox, Date, Details, Title, Amount.
The localized `entries.columns.details` key supplies the user-facing header in
English and German. Positioning the group before Title preserves a consistent
scan from when the entry happened, through its classification, to what it was.

### Reuse current table responsiveness

The existing table wrapper already handles narrow widths with horizontal
scrolling. The grouped cell reduces column count and does not require a mobile
card variant or breakpoint-specific structural changes.

## Risks / Trade-offs

- [Entries with many tags make rows taller] → Tag labels wrap only within the
  Details cell, which keeps columns stable and makes every tag visible.
- [Shared owner badges can consume horizontal room] → Existing label components
  already wrap their sharing treatment in narrow cells; reusing them avoids a
  new truncation rule.
- [Metadata becomes less individually scannable by column] → The Details
  header and fixed account/category/tags order make the grouping predictable,
  while freeing space for the primary title content.

## Migration Plan

1. Update the ledger table markup and translations in one frontend change.
2. Run the frontend lint, type-check, and production build.
3. Verify entries with complete, missing, shared, unavailable, and many-tag
   metadata at desktop and narrow widths.

No deployment migration, API rollout, or rollback data handling is needed. A
rollback restores the previous frontend table markup and translations.
