## ADDED Requirements

### Requirement: A running-balance series can be sampled per day over a month

`GET /api/entries/balance-series` SHALL accept `account_id` (repeatable,
omitted meaning every non-deleted account the caller owns, same as
`GET /api/entries`), `unit` (only `day` is defined), `year`, and `month`
(1-12, required). It SHALL return the running balance of the matching
accounts sampled at each local midnight of `year`/`month` — one point at
`00:00` on each calendar day of that month, plus a closing point at the end
of the last day (equivalently, `00:00` on the first day of the following
month) — using the caller's resolved timezone setting (`user-settings`) to
determine those midnight boundaries.

Each point's value SHALL be the balance computed exactly as
`GET /api/accounts/{id}/balance` computes it as of that instant — the
running sum of `amount` over the account's non-deleted entries with
`booking_timestamp` strictly before the instant, with a
`balance_adjustment` acting as an anchor (everything before the latest
adjustment at or before the instant is ignored, and the result lands
exactly on that adjustment's `balance` reading). When several accounts
match, each point SHALL carry one balance per currency (the currency of the
contributing accounts), the same per-currency grouping shape as
`GET /api/entries/flow-summary`; accounts sharing a currency are summed.

Unlike `GET /api/entries` and `GET /api/entries/summary`, this endpoint
SHALL NOT accept `category_id`, `category_mode`, `tag_id`, `from`, `to`, or
`q`. A balance is defined only by the account and the instant; a
`balance_adjustment` carries no category or tag, so honouring such a filter
would drop the anchors the running sum depends on and yield a number that
is not a balance.

#### Scenario: One point per day plus a closing point

- **WHEN** `GET /api/entries/balance-series?account_id={id}&unit=day&year=2026&month=3`
  is called
- **THEN** the response has 32 points — one at `00:00` on each of the 31
  days of March 2026 in the caller's timezone, and a final point at `00:00`
  on 1 April 2026

#### Scenario: The first point is the balance of all prior history

- **WHEN** an account has entries both before and during March 2026, and
  the balance series for March 2026 is requested
- **THEN** the first point's value equals the account's balance as of
  `00:00` on 1 March 2026 — reflecting every non-deleted entry booked
  before that instant and nothing booked on or after it

#### Scenario: A transaction moves the line the following day

- **WHEN** an account holds `100000` at `00:00` on 10 March 2026 and a
  single `transaction` of `-2500` is booked on 10 March 2026
- **THEN** the point for 10 March is `100000` and the point for 11 March is
  `97500`

#### Scenario: A mid-month balance adjustment re-anchors the line

- **WHEN** a `balance_adjustment` with `balance: 50000` is booked on
  15 March 2026 for an account whose computed balance just before it was
  `48000`
- **THEN** every point from 16 March onward reflects `50000` plus any later
  entries, regardless of the balance on 14 March

#### Scenario: Timezone determines the day boundaries

- **WHEN** the caller's resolved timezone is `Europe/Vienna` and an entry
  is booked at `2026-03-09T23:30:00Z` (00:30 on 10 March local)
- **THEN** the entry's local day is 10 March, so it is reflected from the
  11 March point onward — not from the 10 March point, where a UTC reading
  of the same timestamp would place it

#### Scenario: Multiple accounts of different currencies

- **WHEN** the caller owns a EUR account and a USD account and requests the
  balance series with no `account_id`
- **THEN** each point carries a separate EUR balance and USD balance, and
  never a single combined figure

#### Scenario: A month with no entries is a flat line

- **WHEN** an account has no entries booked during or after the requested
  month
- **THEN** every point has the same value — the account's balance as of the
  month's first midnight

#### Scenario: Filtering parameters are rejected

- **WHEN** `GET /api/entries/balance-series?unit=day&year=2026&month=3&tag_id={tid}`
  is called
- **THEN** the request is rejected (`400`)

#### Scenario: month is required

- **WHEN** `GET /api/entries/balance-series?unit=day&year=2026` is called
  with no `month`
- **THEN** the request is rejected (`400`)

#### Scenario: A soft-deleted entry does not affect the series

- **WHEN** an entry that would otherwise move the balance during the
  requested month has been soft-deleted
- **THEN** the series is identical to one computed with that entry absent
