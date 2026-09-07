## ADDED Requirements

### Requirement: Entry amounts are colored by sign, with a distinct treatment for balance adjustments

The entries list SHALL render each entry's amount in a color reflecting its
sign: red when negative, the default/neutral text color when exactly zero,
and green when positive. An entry whose `kind` is `balance_adjustment`
SHALL additionally have its amount rendered underlined, distinguishing it
from a `transaction`'s amount at a glance.

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

### Requirement: A transaction's amount is entered via a sign toggle and cannot be zero

On `/entries/new` and `/entries/{id}/edit`, when the entry's `kind` is
`transaction`, the amount field SHALL present a sign toggle control
alongside a magnitude-only input, rather than a single free-typed signed
value. Typing or pasting a `-` character into the magnitude input SHALL
never insert that character into the field; it SHALL instead set the
toggle to minus, whether or not it was already set to minus. A newly
created transaction SHALL start with the toggle defaulted to minus.
Submitting the form with a magnitude that resolves to exactly zero SHALL
be rejected with a validation error and SHALL NOT submit the request — a
transaction's amount must be strictly positive or strictly negative.

The amount field's background and text color SHALL reflect the current
sign and magnitude the same way the read-only display does: the default
neutral styling while the magnitude is zero, a red-tinted styling while
the toggle is set to minus and the magnitude is non-zero, and a
green-tinted styling while the toggle is set to plus and the magnitude is
non-zero. The toggle control itself SHALL use a more saturated accent of
the same red/green colors, distinct from the field's lighter tint.

This requirement does not apply when the entry's `kind` is
`balance_adjustment` — see the following requirement.

#### Scenario: Typing a minus sets the toggle and is not inserted

- **WHEN** an authenticated visitor types `-` into the transaction amount
  field, regardless of the toggle's current state
- **THEN** the toggle is set to minus and the `-` character does not appear
  in the field's text

#### Scenario: A new transaction defaults to minus

- **WHEN** an authenticated visitor opens `/entries/new` and selects
  `kind: transaction`
- **THEN** the amount field's sign toggle starts set to minus

#### Scenario: Toggling the sign updates the field's styling

- **WHEN** an authenticated visitor has entered a non-zero magnitude and
  activates the sign toggle
- **THEN** the resulting signed amount, the field's background/text tint,
  and the toggle's own accent color all switch to match the new sign

#### Scenario: A zero-magnitude transaction cannot be submitted

- **WHEN** an authenticated visitor submits the transaction amount field
  with a magnitude of zero
- **THEN** the form shows a validation error and does not submit

### Requirement: A balance adjustment's amount field is unaffected by the sign toggle

On `/entries/new` and `/entries/{id}/edit`, when the entry's `kind` is
`balance_adjustment`, the amount field SHALL remain a single free-typed
input with no sign toggle control and no sign-based coloring, and SHALL
continue to accept a magnitude that resolves to exactly zero.

#### Scenario: Balance adjustment amount field has no toggle

- **WHEN** an authenticated visitor opens the amount field for an entry
  (new or existing) whose `kind` is `balance_adjustment`
- **THEN** no sign toggle button is shown and the field is not colored by
  sign

#### Scenario: A zero-amount balance adjustment can still be submitted

- **WHEN** an authenticated visitor submits a `balance_adjustment` entry
  with an amount of zero
- **THEN** the entry is saved successfully
