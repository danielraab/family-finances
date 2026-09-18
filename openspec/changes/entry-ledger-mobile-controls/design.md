# Design

## Which parameters "clear all filters" clears

The ledger's URL carries eleven parameters. Three groups, handled
differently:

| Parameter(s) | Cleared? |
| --- | --- |
| `account_id`, `category_id`, `tag_id`, `kind`, `q` | Yes |
| `range`, `from`, `to` | Yes — back to the implicit "Last 2 weeks" default |
| `show_recurring` | Yes |
| `sort`, `dir` | **No** |
| `last` | Never present at this point (resolved on arrival) |

Sort is excluded because it is not a filter: it changes the order of the
same result set, not which entries are in it. A visitor who sorted by
amount and then cleared their filters almost certainly still wants the
amount sort.

`show_recurring` is included even though it adds rows rather than
removing them. It is a toggle in the same panel, it is counted as active
in the badge, and leaving it on after "Clear all filters" would
contradict the count dropping to zero.

Clearing is one `navigate({ search: … })` call that keeps only `sort` and
`dir`, so it is a single history entry and the existing persistence
effect stores the cleared state like any other change.

## What counts as an "active" filter

One per control, so the badge matches what the visitor sees when they
expand the panel:

- `account_id`, `category_id`, `tag_id`, `kind`, `q` — each counts when
  present.
- The date range counts **once** when any of `range`, `from`, or `to` is
  present. The implicit "Last 2 weeks" default is not in the URL and does
  not count — otherwise every visitor would arrive to a ledger that
  claims to be filtered.
- `show_recurring` counts when `true`.

Maximum is therefore 7, matching the seven controls in the panel.

## Why the panel collapses only below `sm`

The collapse exists because seven stacked controls fill a phone screen
before a single entry is visible. At `sm` and up the grid is two or three
columns and the whole panel is two or three rows, which does not crowd
the list — so the toggle is not rendered there at all rather than
remembering a separate desktop state.

The collapsed state starts closed on every load and is deliberately not
persisted. The active-filter count in the header is what tells a visitor
the ledger is filtered; persisting "open" would trade the screen space
back for information the badge already carries. A visitor arriving from
an account page (`?account_id=…`) sees the panel closed with a `1` badge,
which is the intended reading of that state.

## Why the search clear button is not the browser's

`input[type="search"]` renders a cancel glyph in Chrome and Safari on
desktop, but not on iOS Safari or on Firefox — the field is unclearable
on exactly the devices this change is about. The panel therefore renders
its own button, absolutely positioned inside the field's box, and
suppresses `::-webkit-search-cancel-button` so a desktop Chrome visitor
does not get two X's side by side.

The button clears the debounce timer before navigating. Without that, a
visitor who types and immediately clears would have the pending 300ms
timeout fire afterwards and re-apply the text they just removed.
