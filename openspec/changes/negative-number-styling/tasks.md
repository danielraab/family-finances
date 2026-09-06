## 1. Shared color helper

- [x] 1.1 `frontend/src/lib/amount.ts`: add `amountColorClass(amount:
  number): string` — negative → `"text-red-600 dark:text-red-400"`,
  positive → `"text-emerald-600 dark:text-emerald-400"`, zero → `""`.

## 2. Read-only amount display

- [x] 2.1 `frontend/src/routes/entries.index.tsx`: apply
  `amountColorClass(entry.amount)` to the amount cell; add `underline`
  when `entry.kind === "balance_adjustment"`.
- [x] 2.2 `frontend/src/routes/accounts.index.tsx`: apply
  `amountColorClass(balance)` to each account's balance figure (no
  underline — not tied to a single entry's kind).
- [x] 2.3 `frontend/src/routes/accounts.$accountId.index.tsx`: apply
  `amountColorClass(balance)` to the account's headline balance (no
  underline); apply `amountColorClass(entry.amount)` plus the
  `balance_adjustment` underline to each row of the recent-entries list.

## 3. Shared sign-toggle amount input

- [x] 3.1 New `frontend/src/components/SignedAmountInput.tsx`: props for
  magnitude text + its `onChange`, current sign (`negative: boolean`) +
  its toggle callback, currency (for the label), and `invalid`. Owns the
  three-way (`zero` / `negative` / `positive`) styling on the input
  (pastel bg/text/border per the design) and renders the accent-colored
  `−`/`+` toggle button before it, joined into one control. The input's
  `onChange` strips any `-` from the incoming value and forces `negative`
  to `true` whenever one was present, regardless of the prior sign state,
  before storing the cleaned magnitude text.
- [x] 3.2 Toggle button has a translated `aria-label` reflecting its
  action (new i18n key).

## 4. Wire into the entry form

- [x] 4.1 `frontend/src/routes/entries.new.tsx`: for `kind === "transaction"`,
  render `SignedAmountInput` in place of the plain input, with `negative`
  defaulting to `true`; for `kind === "balance_adjustment"`, keep today's
  plain input exactly as-is. Submit validation: for a transaction, reject
  (via the existing `invalidField` mechanism) when the parsed magnitude is
  `null` or `0`; combine `negative` + magnitude into the signed integer
  sent to `POST /api/entries`.
- [x] 4.2 `frontend/src/routes/entries.$entryId.edit.tsx`: same branching
  on the entry's (immutable) `kind`; initialize `SignedAmountInput`'s state
  from the existing entry (`negative = entry.amount < 0`, magnitude =
  `amountToInput(Math.abs(entry.amount))`) when `kind === "transaction"`;
  same submit-time zero rejection and sign recombination for
  `PATCH /api/entries/{id}`.

## 5. i18n

- [x] 5.1 Add new keys to `frontend/src/i18n/locales/en.json` first, then
  `de.json`: the sign-toggle button's `aria-label` (e.g. "Switch to
  positive"/"Switch to negative", or a single toggle label — whichever the
  implementation needs), and a validation message for a zero-amount
  transaction (e.g. `entries.form.amountZero`).

## 6. Verify

- [x] 6.1 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [x] 6.2 Manual pass in the browser: confirm red/black/green on negative/
  zero/positive amounts in all four read-only spots; confirm
  `balance_adjustment` amounts are underlined in the entries list and in
  an account's recent entries, and `transaction` amounts are not;
  create a new transaction and confirm the sign starts at minus, that
  typing or pasting `-` (at any point in the string) never inserts the
  character and always results in minus, that clicking the toggle flips
  sign and updates both the field's pastel tint and the button's own solid
  accent; confirm submitting a transaction with magnitude `0` is blocked
  with a validation message; confirm a `balance_adjustment` entry's amount
  field is visually and behaviorally unchanged (no button, no color, `0`
  still accepted) in both the new-entry and edit forms.
