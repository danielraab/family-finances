# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.4.1] - 2026-09-19

### Added

- Convert an existing transaction into a self-transfer, from the entry
  edit page: you pick the counterparty account and which side the entry's
  own account is on (sender or receiver), rather than it being inferred
  from the amount's sign. Backed by a new
  `POST /api/entries/{id}/self-transfer` that soft-deletes the original
  entry and creates its replacement in one database transaction,
  recomputing the balance-adjustment chain on both accounts. Title,
  description, date, tags and category carry over; counterparty and
  location are dropped, as on any self-transfer.
- Read-only summary modals for entries and recurring transactions.
  Reading an entry no longer means opening its edit form: an entry's
  title opens a summary on the `/entries` ledger, an account's detail
  page, `/reports` and the dashboard's entry-list card, and a recurring
  transaction's title does the same on `/recurring` and from the Upcoming
  block. Edit is one click away inside, under the same permission rule
  the edit page applies, and the recurring summary also offers "Create
  transaction" — carrying the occurrence's own date along when it was
  opened from an Upcoming row.
- A per-month amount on `/recurring`, beside the per-year one on every
  row and in the per-currency footer totals — the figure a household
  actually budgets against.

### Changed

- The `/entries` ledger's controls now work on a phone: "New entry" is a
  plus glyph below `sm`, the search field has a clear button, and the
  seven filters moved into a panel whose header carries "Clear all
  filters" and, on narrow viewports, a collapse toggle with a count of
  the active filters, so a collapsed panel never hides that the ledger is
  filtered. The ledger also remembers its filter, search and sort state,
  which is what a save or a conversion returns you to.
- `/recurring` got the same treatment: a plus-glyph create action, and a
  row's "Create transaction" is now an inline-flex control that no longer
  renders broken when it wraps.
- The Upcoming block's rows are now two lines — title against amount,
  then date and account against the create action — so nothing overflows
  or truncates to a few characters in a narrow dashboard card, where the
  block also no longer draws its own box inside the card's.
- All 23 dialogs now share one `Modal` shell, which owns the backdrop,
  panel and title and bounds the panel's height — previously nothing did,
  and the new summaries are the tallest content in the app.

### Fixed

- The i18n-coverage CI job now actually posts its pull request comment.
  Both the comment and stale-comment-removal steps were implicitly gated
  on the coverage step succeeding, so they were skipped in exactly the
  case they exist for: a locale short of 100%.
- The entry edit page's action buttons wrap onto their own lines on a
  narrow viewport instead of being squeezed into one row with their text
  broken mid-word.
- A self-transfer's badge opens the entry's summary, like the row's title
  next to it, instead of jumping to the edit page.
- Negative amounts on `/recurring` no longer wrap after their minus sign.
- German translations for the self-transfer conversion, which had left
  German at 99.1% i18n coverage.

## [0.4.0] - 2026-09-18

### Added

- New `self_transfer` entry kind for moving money between two of your own
  (or shared) accounts as a single entry instead of two unrelated ones: the
  entry form gains a "Self-transfer" option alongside Transaction and
  Balance adjustment, revealing a "To account" picker restricted to
  append+, same-currency, non-disabled accounts. A self-transfer's amount
  is included in summary/flow-summary totals (unlike a balance adjustment)
  and renders once per side of the transfer that's within the caller's
  account scope, with a badge on each leg linking to the other.

### Changed

- **BREAKING**: an account's `currency` can no longer be changed once the
  account has any entry — `PATCH /api/accounts/{id}` with a changed
  `currency` on such an account is rejected, closing the gap that could let
  a same-currency transfer pair drift into a mismatch after the fact. The
  account edit form disables the currency field accordingly.
- The i18n-coverage CI job now only comments on a pull request when some
  locale is short of 100% coverage, and removes a stale comment once a
  previously-short PR reaches full coverage.

### Fixed

- The entry-kind radio buttons on `/entries/new` no longer stretch apart
  vertically when they wrap onto multiple lines on narrow/mobile
  viewports.
- German translations for the self-transfer feature, which had left German
  at 99.1% i18n coverage.

## [0.3.2] - 2026-09-17

### Added

- Shortcut to create a recurring transaction directly from an entry: an
  icon-only action on transaction entries opens the recurring transaction
  form prefilled from the entry, and the new recurring transaction is
  auto-linked back to the entry it was created from.
- `/recurring/{id}/edit` gains a "Linked transactions" section listing the
  entries linked to that template, newest-first and cursor-paginated via
  "Load more", each linking to that entry's own edit page.
- `recurring_transaction_id` filter on `GET /api/entries`, scoped to the
  caller's visible accounts, backing the linked-transactions list above.

## [0.3.1] - 2026-09-16

### Added

- Bulk actions on the `/entries` ledger: select entries via row/header
  checkboxes and act on all of them at once — set category, add/remove/set
  tags, link to a recurring transaction (disabled across a multi-account
  selection), or delete. Each action confirms in a dialog and reports
  progress and any failures in a shared result modal.

## [0.3.0] - 2026-09-16

### Added

- Recurring transactions: a new `/recurring` page for defining recurring
  costs and income (rent, subscriptions, salary, insurance) as templates
  with an interval, start date, and optional end date. Each shows its
  per-year amount and a total grouped by currency. "Create transaction"
  always requires a manual click — nothing is created automatically — and
  opens the entry form prefilled from the template. Entries linked to a
  recurring transaction show a badge on the `/entries` ledger and `/reports`
  results.
- A bounded, opt-in preview of upcoming recurring occurrences on
  `/entries`, `/reports`, and the dashboard's entry-list and bar-chart
  cards, with a new "recurring preview horizon" profile setting (1/2/3
  months from now, end of this/next month, or end of this year).
- A "Show recurring assumptions" toggle on the account balance chart and
  on the dashboard's new `line_chart` card, overlaying a projected balance
  line built from upcoming recurring occurrences.
- New `line_chart` dashboard card type: a running-balance chart, scoped to
  one account or summed across every account, with a month pager.
- Shared date-range presets (Today, Last 7/14/30 days, This week, Last
  week, Last 2 weeks, This month, Last month, This year) on `/entries` and
  `/reports`, plus a per-user week-start setting (Monday or Sunday).
- Category and tag filtering on the flow-summary data behind dashboard bar
  charts.

### Changed

- The landing page now redirects an already-authenticated visitor
  straight to `/home` instead of showing the marketing page.
- `LineChart` now uses the same floating, click-to-pin tooltip `BarChart`
  already had, in place of its fixed value box below the chart.
- Improved `BarChart` tooltip positioning so it stays anchored correctly
  on scroll and resize.

## [0.2.0] - 2026-09-13

### Added

- CSV/JSON transaction import wizard at `/entries/import`: pick an
  account, select a file, map its columns/fields to entry fields
  (title, amount, booking date, description, counterparty, location),
  with support for split debit/credit amount columns and configurable
  decimal/thousands separators. A fully offline dry run lists every
  row with its status (ready, suspicious, failed) and lets a failed
  row be remapped individually before importing, then creates entries
  with a live progress indicator and a final results summary. Reachable
  from the entries ledger and account detail page via a new "Import"
  action next to "New Entry".

### Fixed

- CSV file selection on Android, and structured JSON amount values
  that previously failed to parse.
- The import wizard now respects the browser's back button and shows
  row-specific examples when remapping a failed row's fields.
- A row rejected during import now surfaces the backend's own error
  message instead of a generic one.

## [0.1.2] - 2026-09-12

### Added

- Counterparty and location fields on transaction entries: a
  counterparty text input with autocomplete suggestions (backed by a
  new `GET /api/entries/counterparties` endpoint), and a location
  input with "Use GPS" and "Pick on map" pickers backed by
  Leaflet/OpenStreetMap. The ledger shows a globe icon and
  counterparty line on entries that carry them, with a read-only map
  preview on click.

## [0.1.1] - 2026-09-11

### Added

- More account-type suggestions (prepaid card, mortgage, brokerage,
  retirement, business, insurance) and a misc/"Other" catch-all.

### Changed

- Account creation and profile settings now share the same currency
  select component; the default-currency field in profile settings no
  longer takes free-text input, and new accounts preselect the
  visitor's saved default currency.
- Account-type suggestions are now translated and match the visitor's
  language instead of being hardcoded in English.

## [0.1.0] - 2026-09-11

### Added

- Account sharing with permission tiers, including the frontend UI for
  managing shared accounts.
- Category sharing with view/append permissions, with shared status and
  owner name surfaced in category dropdowns and category-aware entry
  visibility.
- Tag sharing functionality.
- `entry_count` on category responses, shown in the frontend.
- Category and tags columns in the entries list.
- Independent category/tag selection when generating reports.
- User display names.
- Dependabot configuration for automated dependency updates.

### Changed

- Account types are now free-text input; the separate account types
  management UI was removed.
- Category actions consolidated into a single edit dialog.

### Fixed

- CI: the `publish` job being skipped on every tag push, so tagged
  releases actually build and publish an image.
- Sidebar mobile responsiveness.
- `AccountLabel` layout and responsiveness for shared accounts.
- Invitation-row behavior and the revoke button's enabled condition.
- Magic link and invite acceptance now always resolve to the correct
  account.
- The category sharing page now opens correctly from the Share button.
- Sharing/invite form layout and accessibility.
- Frontend build and runtime compatibility with Node 26.

### Build

- Bumped Go, Node, and various GitHub Actions and frontend/backend
  dependencies.

## [0.0.0] - 2026-09-09

Initial tagged snapshot of the project.

[Unreleased]: https://github.com/danielraab/family-finances/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/danielraab/family-finances/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/danielraab/family-finances/compare/v0.3.2...v0.4.0
[0.3.2]: https://github.com/danielraab/family-finances/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/danielraab/family-finances/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/danielraab/family-finances/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/danielraab/family-finances/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/danielraab/family-finances/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/danielraab/family-finances/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/danielraab/family-finances/compare/v0.0.0...v0.1.0
[0.0.0]: https://github.com/danielraab/family-finances/releases/tag/v0.0.0
