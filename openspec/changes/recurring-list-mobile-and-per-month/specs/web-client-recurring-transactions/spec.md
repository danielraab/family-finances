## MODIFIED Requirements

### Requirement: The recurring transaction list shows entered, per-year and per-month amounts, and a per-currency total

`/recurring` SHALL list the caller's visible recurring transactions
(`GET /api/recurring-transactions`), each row showing at least its title,
category, entered `amount`, computed `per_year_amount`, and a per-month
amount derived from `per_year_amount` by dividing it by twelve. A
recurring transaction whose `ended` is `true` SHALL be visually
distinguished (e.g. muted styling or a badge) from an active one. Below
the list, the page SHALL show both the total per-year amount per currency
(`GET /api/recurring-transactions/summary`) and a per-month total derived
from each of those per-currency totals by dividing it by twelve, matching
the same per-currency grouping the API returns — never a single combined
figure across currencies.

Every per-month figure SHALL be rendered in the same currency, at the
same display precision, and with the same sign colouring as the per-year
figure it is derived from.

Every amount the list renders — entered, per year, per month, and each
per-currency total — SHALL be rendered on a single line at every viewport
width, with its sign attached to its digits.

#### Scenario: Row shows entered, per-year and per-month amounts

- **WHEN** a recurring transaction with `amount: -80000` and
  `per_year_amount: -960000` is listed
- **THEN** its row shows the entered amount, the per-year amount, and a
  per-month amount of `-80000`

#### Scenario: A non-monthly row's per-month amount is the annual figure divided by twelve

- **WHEN** a yearly recurring transaction with `per_year_amount: -6000000`
  is listed
- **THEN** its row shows a per-month amount of `-500000`

#### Scenario: A negative amount keeps its sign on one line

- **WHEN** a recurring transaction with a negative amount is listed
- **THEN** each of its amount cells renders the sign and the digits on
  the same line, at every viewport width

#### Scenario: Ended template is visually distinguished

- **WHEN** a listed recurring transaction's `ended` is `true`
- **THEN** its row is rendered with a distinct, muted treatment from active
  rows

#### Scenario: Total is per currency

- **WHEN** the caller's recurring transactions span two account
  currencies
- **THEN** the page shows two separate per-year totals and two separate
  per-month totals, one of each per currency

#### Scenario: The per-month total is the per-year total divided by twelve

- **WHEN** the summary endpoint returns a per-year total of `-15000000`
  for a currency
- **THEN** the page shows a per-month total of `-1250000` for that
  currency

### Requirement: Creating a transaction from a recurring transaction is always a manual action

Each recurring transaction (on the list and on its edit page) SHALL offer a
"Create transaction" action. Activating it SHALL navigate to
`/entries/new` prefilled from the template — account, title, description,
category, counterparty, location, tags, and signed amount — with
`booking_timestamp` prefilled to the recurring transaction's
`next_suggested_date`, and with `recurring_transaction_id` carried through
so the entry is linked once submitted. Every prefilled field SHALL remain
editable before submission, and no entry SHALL be created without the
visitor explicitly submitting that form — there is no automatic or
scheduled creation.

On the list, this action SHALL render as a single unbroken control whose
label never wraps across lines at any viewport width. Below the `sm`
breakpoint it SHALL render as a plus glyph with no visible label; at `sm`
and above it SHALL render its translated label. At every width its
accessible name SHALL be that same translated label and its destination
SHALL be unchanged.

#### Scenario: Create transaction prefills and links

- **WHEN** the visitor activates "Create transaction" on a recurring
  transaction and submits the prefilled form unchanged
- **THEN** a new entry is created matching the template's fields, booked on
  the template's `next_suggested_date`, with `recurring_transaction_id` set
  to that recurring transaction

#### Scenario: Prefilled fields remain editable

- **WHEN** the visitor activates "Create transaction" and changes the
  amount or booking date before submitting
- **THEN** the created entry reflects the edited values, still linked to
  the recurring transaction

#### Scenario: No background or scheduled creation

- **WHEN** a recurring transaction's `next_suggested_date` has passed with
  no visitor action taken
- **THEN** no entry is created automatically — the list still shows the
  template with its (past) `next_suggested_date`, unchanged until the
  visitor manually acts

#### Scenario: The row action is a plus glyph on a phone

- **WHEN** an authenticated visitor opens `/recurring` on a viewport
  narrower than `sm`
- **THEN** each row's create-transaction action shows a plus glyph and no
  visible label, occupies a single unbroken box, and still exposes its
  translated label as its accessible name

#### Scenario: The row action keeps its label on a wide viewport

- **WHEN** an authenticated visitor opens `/recurring` on a viewport at or
  above `sm`
- **THEN** each row's create-transaction action shows its translated
  label on one line

## ADDED Requirements

### Requirement: The recurring list's create action is icon-only on a narrow viewport

On a viewport narrower than the `sm` breakpoint, `/recurring`'s "New
recurring transaction" action SHALL render as a plus glyph with no visible
label. At `sm` and above it SHALL render its translated label as before.
At every width its accessible name SHALL be that same translated label,
and its destination SHALL be unchanged — `/recurring/new`.

#### Scenario: A phone-width visitor sees a plus button

- **WHEN** an authenticated visitor opens `/recurring` on a viewport
  narrower than `sm`
- **THEN** the create action shows a plus glyph and no visible label,
  while still exposing its translated label as its accessible name

#### Scenario: A wide viewport keeps the label

- **WHEN** an authenticated visitor opens `/recurring` on a viewport at or
  above `sm`
- **THEN** the create action shows its translated label
