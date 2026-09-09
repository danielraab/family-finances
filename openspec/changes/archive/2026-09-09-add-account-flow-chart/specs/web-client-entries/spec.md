## MODIFIED Requirements

### Requirement: Entry amounts are colored by sign, with a distinct treatment for balance adjustments

The entries list SHALL render each entry's amount in a color reflecting its
sign: red when negative, the default/neutral text color when exactly zero,
and green when positive. An entry whose `kind` is `balance_adjustment`
SHALL additionally have its amount rendered underlined, distinguishing it
from a `transaction`'s amount at a glance, and SHALL additionally show its
computed delta (`amount`) as a small, gray annotation next to its reading,
rendered without sign-based coloring.

#### Scenario: Negative amount is red

- **WHEN** the entries list renders an entry with a negative amount
- **THEN** the amount is shown in red

#### Scenario: Zero amount stays neutral

- **WHEN** the entries list renders an entry with an amount of exactly zero
- **THEN** the amount is shown in the default text color, neither red nor
  green

#### Scenario: Positive amount is green

- **WHEN** the entries list renders an entry with a positive amount
- **THEN** the amount is shown in green

#### Scenario: A balance adjustment's amount is underlined

- **WHEN** the entries list renders an entry whose `kind` is
  `balance_adjustment`
- **THEN** its amount is rendered underlined, in addition to its sign color

#### Scenario: A transaction's amount is not underlined

- **WHEN** the entries list renders an entry whose `kind` is `transaction`
- **THEN** its amount is rendered without an underline

#### Scenario: A balance adjustment shows its delta alongside its reading

- **WHEN** the entries list renders an entry whose `kind` is
  `balance_adjustment`
- **THEN** its computed delta is shown next to its reading, in a smaller,
  gray typeface, not colored by sign

### Requirement: A balance adjustment's amount field is unaffected by the sign toggle

On `/entries/new` and `/entries/{id}/edit`, when the entry's `kind` is
`balance_adjustment`, the amount field SHALL remain a single free-typed
input with no sign toggle control and no sign-based coloring, and SHALL
continue to accept a magnitude that resolves to exactly zero. This field
edits the entry's `balance` reading, not its `amount` — submitting the form
for a `balance_adjustment` SHALL send the entered value as `balance`, and
the computed `amount` (delta) returned by the backend is never edited
directly.

#### Scenario: Balance adjustment amount field has no toggle

- **WHEN** an authenticated visitor opens the amount field for an entry
  (new or existing) whose `kind` is `balance_adjustment`
- **THEN** no sign toggle button is shown and the field is not colored by
  sign

#### Scenario: A zero-amount balance adjustment can still be submitted

- **WHEN** an authenticated visitor submits a `balance_adjustment` entry
  with an amount of zero
- **THEN** the entry is saved successfully

#### Scenario: Submitting a balance adjustment sends balance, not amount

- **WHEN** an authenticated visitor submits the create or edit form for a
  `kind: balance_adjustment` entry
- **THEN** the request body carries the entered value as `balance`, and the
  form does not submit an `amount` field for that entry
