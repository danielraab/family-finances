## Why

The entry ledger's toolbar and filter row were laid out for a desktop
window and never adapted to a phone. Three concrete problems:

- **The toolbar crowds the title.** `/entries`' header is the page title
  beside two text buttons, "Import" and "New entry". At 375px the three
  compete for the same line; "New entry" is the primary action on the
  page and the one hurt most by the squeeze.
- **The search field cannot be emptied without selecting its text.**
  `q` is a plain `type="search"` input. Chrome's native cancel glyph is
  the only clear affordance, and it is absent on iOS Safari and Firefox
  — so on the phones this app is actually read on, clearing a search
  means selecting the text and deleting it.
- **The filter row is six controls with no way out.** Account, category,
  tag, kind, date range, the upcoming-recurring toggle, and search are a
  `flex-wrap` row of auto-width controls. On a phone they stack into
  roughly a full screen of chrome before the first entry is visible, and
  once several are set there is no single action that returns the ledger
  to an unfiltered view — each one has to be walked back to "All …" by
  hand.

## What Changes

- The "New entry" action renders as a plus glyph alone below the `sm`
  breakpoint, and as the existing labelled button from `sm` up. Its
  accessible name is the same translated label at both sizes, so nothing
  is lost to a screen reader.
- The free-text search input gains a clear button, shown only while the
  field has text. Activating it empties the field, drops `q` from the
  URL immediately (without waiting out the 300ms debounce), and returns
  focus to the input. The browser's own search-cancel glyph is
  suppressed so there is exactly one clear affordance.
- The filter controls move into a bordered panel with a header row. The
  header carries a "Clear all filters" action, shown only while at least
  one filter is active, which returns every filter to its default in one
  navigation while leaving the sort field and direction alone.
- Below `sm` the panel's controls are collapsed behind a "Filters"
  toggle in that header, which carries a count of the currently active
  filters so a collapsed panel never hides that the ledger is filtered.
  From `sm` up the controls are always shown and the toggle is not
  rendered.
- Inside the panel the controls become a responsive grid — one column on
  a phone, two from `sm`, three from `lg` — with every control filling
  its cell instead of sizing to its content, and the search field
  spanning the full width of the grid.

## Non-goals

- No change to which filters exist, to their URL parameters, or to how
  they are persisted and restored via `?last=true`. This is layout and
  affordances only.
- **No change to `/reports`**, which has a filter row of the same shape.
  Sharing one filter-panel component between the two pages is a
  worthwhile follow-up, but the two sets of controls are not identical
  and unifying them is a larger change than this one.
- Sort state is not part of "clear all filters". Sorting is a view
  preference, not a filter, and clearing it would be a surprise.
- No new backend endpoint and no API contract change.

## Capabilities

### Modified Capabilities

- `web-client-entries`: the ledger's create action is icon-only on a
  phone; the search input can be cleared in one action; the filter
  controls live in a panel that collapses on a phone and offers a
  clear-all action.

## Impact

- `frontend/src/routes/entries.index.tsx` — the toolbar, the filter
  panel, and the clear-all/collapse state.
- `frontend/src/i18n/locales/{en,de}.json` — keys for the panel heading,
  the collapse toggle, the active-filter count, the clear-all action,
  and the search clear button.
