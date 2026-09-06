## Why

Every rendered amount in the frontend — the entries ledger, the accounts
overview, an account's balance, its recent-entries list — is plain,
unstyled text today, whatever its sign. A negative balance or a refund
reads identically to a deposit; nothing distinguishes a `balance_adjustment`
snapshot from a `transaction` delta at a glance. The create/edit entry form
is worse for data entry: sign lives inside a free-typed string
(`-12.50`), there's no visual cue for which way an amount currently points,
and a transaction can be saved with an amount of exactly `0`, which isn't a
meaningful transaction.

## What Changes

- **Read-only amount display** (entries list, accounts overview balances,
  account detail balance, account detail recent entries): every amount is
  colored by sign — negative red, zero left at the default/neutral color,
  positive green. Wherever a single entry's `kind` is known (the entries
  list and the recent-entries list — not the computed account-balance
  figures), a `balance_adjustment` entry's amount is additionally
  underlined, so it reads as visually distinct from a `transaction` delta.
- **Transaction entry form** (`/entries/new` and `/entries/{id}/edit`, only
  when `kind` is `transaction`): the amount input gains a sign toggle
  button in front of it. The input itself only ever holds the magnitude —
  typing (or pasting) a `-` flips the sign to minus and the character is
  never inserted into the field, whether the sign was already minus or not.
  The field's background and text follow the same red/black/green rule as
  the read-only case, but as a filled pastel (badge-style) treatment; the
  toggle button itself uses a bolder, more saturated accent of the same two
  colors so it reads as the primary cue. A new transaction entry starts
  with the sign defaulted to minus. Submitting a transaction with a
  magnitude of `0` is rejected client-side with a validation message — a
  transaction must be strictly positive or negative.
- **`balance_adjustment` entries are explicitly out of scope for the form
  change**: the amount field for a balance adjustment keeps today's plain
  input exactly as-is — no toggle button, no coloring, `0` still accepted,
  sign still typed directly. Only its read-only rendering (color +
  underline) changes, per the bullet above.

## Capabilities

### Modified Capabilities

- `web-client-entries`: entries list amount coloring/underline; the entry
  form's amount field gains the sign-toggle control and the
  zero-amount-rejected-for-transactions rule (`balance_adjustment` amount
  entry is unaffected).
- `web-client-accounts`: accounts overview and account detail balance
  coloring; account detail recent-entries amount coloring/underline.

## Impact

- **Code**:
  - `frontend/src/lib/amount.ts` — a small helper resolving an amount's
    sign to a color class (shared by every read-only render site and the
    form's field/button styling).
  - `frontend/src/routes/entries.index.tsx`,
    `frontend/src/routes/accounts.index.tsx`,
    `frontend/src/routes/accounts.$accountId.index.tsx` — apply the color
    (+ underline where `kind` is available) to each rendered amount.
  - `frontend/src/routes/entries.new.tsx`,
    `frontend/src/routes/entries.$entryId.edit.tsx` — the transaction
    branch of the amount field gets the new sign-toggle control (likely
    factored into a small shared component given the two routes already
    duplicate this field); the `balance_adjustment` branch is untouched.
    Zero-amount validation for transactions.
  - New i18n keys in `frontend/src/i18n/locales/{en,de}.json` (toggle
    button `aria-label`, the zero-amount validation message).
- **API contract**: none — `amount` is already a plain signed integer on
  the wire; this is purely presentational and client-side validation.
- **Spec**: deltas on `web-client-entries` and `web-client-accounts`.
