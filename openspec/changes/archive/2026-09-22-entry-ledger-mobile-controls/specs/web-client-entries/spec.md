## ADDED Requirements

### Requirement: The ledger's create action is icon-only on a narrow viewport

On a viewport narrower than the `sm` breakpoint, `/entries`' "New entry"
action SHALL render as a plus glyph with no visible label. At `sm` and
above it SHALL render its translated label as before. At every width its
accessible name SHALL be that same translated label, and its destination
SHALL be unchanged — `/entries/new`, carrying the current `account_id`
filter when one is applied.

#### Scenario: A phone-width visitor sees a plus button

- **WHEN** an authenticated visitor opens `/entries` on a viewport
  narrower than `sm`
- **THEN** the create action shows a plus glyph and no visible label,
  while still exposing its translated label as its accessible name

#### Scenario: A wide viewport keeps the label

- **WHEN** an authenticated visitor opens `/entries` on a viewport at or
  above `sm`
- **THEN** the create action shows its translated label

#### Scenario: The icon-only action carries the account filter

- **WHEN** a visitor on a viewport narrower than `sm` is viewing
  `/entries?account_id={id}` and activates the create action
- **THEN** the client navigates to `/entries/new?account_id={id}`

### Requirement: The ledger's search input can be cleared in one action

The ledger's free-text search input SHALL offer a clear action inside the
field whenever the field holds text, and SHALL NOT render it when the
field is empty. Activating it SHALL empty the field, remove `q` from the
URL without waiting for the input's debounce interval, and leave keyboard
focus on the search input. The clear action SHALL be the only clear
affordance the field presents, including in browsers that render a native
search-cancel control.

#### Scenario: Clearing a search empties the field and the URL

- **WHEN** an authenticated visitor with search text applied activates the
  search field's clear action
- **THEN** the field is empty, `q` is absent from the URL, and the
  unfiltered-by-text results are shown

#### Scenario: An empty search field offers no clear action

- **WHEN** the ledger's search field holds no text
- **THEN** no clear action is rendered inside it

#### Scenario: Clearing does not get undone by a pending debounce

- **WHEN** a visitor types search text and activates the clear action
  before the input's debounce interval elapses
- **THEN** `q` stays absent from the URL and the typed text is not
  re-applied

### Requirement: The ledger offers a clear-all-filters action

`/entries` SHALL offer a single action clearing every filter at once,
rendered only while at least one filter is active. Activating it SHALL
remove the account, category, tag, kind, date-range, free-text search,
and upcoming-recurring parameters from the URL in one navigation, and
SHALL leave the sort field and direction untouched. With no
date-range parameter present the ledger's own default range applies
again, per "The entry ledger defaults to the last 2 weeks".

A filter counts as active when its parameter is present in the URL; the
date range counts as one active filter when any of `range`, `from`, or
`to` is present, and the implicitly applied default range does not count.

#### Scenario: Clearing every filter at once

- **WHEN** an authenticated visitor with an account filter, a tag filter,
  and search text applied activates the clear-all-filters action
- **THEN** all three parameters are removed from the URL and the ledger
  shows the default unfiltered view

#### Scenario: Clearing filters preserves the sort

- **WHEN** a visitor sorted by amount ascending activates the
  clear-all-filters action
- **THEN** the results remain sorted by amount ascending

#### Scenario: No action is offered when nothing is filtered

- **WHEN** a visitor opens `/entries` with no filter parameters in the URL
- **THEN** no clear-all-filters action is rendered

### Requirement: The ledger's filter controls collapse on a narrow viewport

The ledger's filter controls SHALL be presented in a panel with a header.
On a viewport narrower than the `sm` breakpoint the panel's controls
SHALL be hidden behind a toggle in that header, closed on load; from `sm`
up the controls SHALL always be shown and no toggle SHALL be rendered.
While at least one filter is active the header SHALL show the number of
active filters, counted as defined above, so a closed panel never hides
that the ledger is filtered.

#### Scenario: Filters start collapsed on a phone

- **WHEN** an authenticated visitor opens `/entries` on a viewport
  narrower than `sm`
- **THEN** the filter controls are not shown, and a toggle for them is

#### Scenario: Expanding the panel reveals every control

- **WHEN** that visitor activates the toggle
- **THEN** the account, category, tag, kind, date-range,
  upcoming-recurring, and search controls are all shown

#### Scenario: A collapsed panel still reports active filters

- **WHEN** a visitor on a viewport narrower than `sm` follows a link to
  `/entries?account_id={id}`
- **THEN** the filter panel is closed and its header shows that one
  filter is active

#### Scenario: A wide viewport shows the controls unconditionally

- **WHEN** an authenticated visitor opens `/entries` on a viewport at or
  above `sm`
- **THEN** the filter controls are shown and no toggle is rendered

## MODIFIED Requirements

### Requirement: Entry list filters, search, and sort controls

`/entries` SHALL offer controls for every backend filter (`account_id`,
`category_id`, `tag_id`, `kind`, a date range via the shared date-range
filter — see `web-client-date-range-filter`), a free-text search input, and
a way to sort by booking timestamp or amount in either direction. When no
entries match the current filters, the page SHALL show text distinguishing
"no entries match these filters" from "no entries exist yet."

Those controls SHALL be laid out as a responsive grid that fills the
available width — one column below the `sm` breakpoint, two from `sm`,
and three from `lg` — with each control sized to its cell rather than to
its content, and the free-text search input spanning the grid's full
width.

#### Scenario: Combining filters narrows the results

- **WHEN** an authenticated visitor sets an account filter, a category
  filter, and a date range together
- **THEN** only entries matching all three narrow the results

#### Scenario: No matches under the current filters

- **WHEN** the current filters match no entries but entries exist on the
  account
- **THEN** the page shows text indicating no entries match the current
  filters, not that none exist at all

#### Scenario: Controls stack in a single column on a phone

- **WHEN** an authenticated visitor views the ledger's filter controls on
  a viewport narrower than `sm`
- **THEN** each control occupies its own full-width row
