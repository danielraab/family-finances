## Context

`frontend/src/lib/amount.ts` already centralizes amount formatting
(`formatAmount`, `amountToInput`/`inputToAmount`, both working in the
stored integer's 4-decimal fixed point). No amount anywhere in the app
carries color today, and the entry form's amount `<input>` is a single
free-typed string parsed by `inputToAmount` (`Number(trimmed)` — accepts a
leading `-` like any other character). `entries.new.tsx` and
`entries.$entryId.edit.tsx` each define their own copy of the field; there
is no shared entry-form component (unlike accounts, which share
`AccountForm.tsx`).

Both entry kinds store `amount` as the same signed integer
(`STORED_DECIMAL_PLACES`, 4dp). A `transaction`'s amount is a signed delta
(positive credit, negative debit); a `balance_adjustment`'s amount is an
absolute balance snapshot at that point in time (per `account-entries`'s
balance formula) — it can still be negative (an overdrawn account) but sign
doesn't carry the same "which direction did money move" meaning, which is
why the ask keeps its form field exactly as it is today.

## Goals / Non-Goals

**Goals:**

- One shared way to derive a sign→color class, used identically by every
  read-only render site and the form's field/button styling, so the three
  colors (and the zero exception) can't drift between places.
- A transaction amount field where the sign is a first-class piece of
  state (a button), not something buried in a typed string — typing `-`
  is *interpreted*, never inserted.
- A transaction can't be saved with amount `0`.

**Non-Goals:**

- Any change to `balance_adjustment`'s amount field — it keeps its current
  plain, unstyled, sign-typed input and still accepts `0`.
- Any backend or API contract change — `amount`'s wire representation and
  validation are unaffected; this is a client-side presentational and
  form-validation change only.
- Coloring or underlining anything other than the four read-only sites and
  the two form routes already named — no dashboard/report surfaces exist
  yet to extend this to.
- A generic reusable `EntryForm` component covering the whole form — only
  the amount field's sign-toggle logic is factored out, to avoid a larger
  refactor than this change calls for.

## Decisions

### Decision: a single `amountColorClass` helper in `lib/amount.ts`

```ts
export function amountColorClass(amount: number): string {
  if (amount < 0) return "text-red-600 dark:text-red-400";
  if (amount > 0) return "text-emerald-600 dark:text-emerald-400";
  return "";
}
```

Red/emerald match colors already established in this codebase
(`entries.form.saveError`'s `text-red-600 dark:text-red-400`, and
`AccountStatus`'s emerald "open" badge in `accounts.index.tsx`) rather than
introducing a new palette. Zero returns `""` — no color class — so it
inherits whatever neutral text color the surrounding element already has;
this is the "0 stays black" rule, and it's centralized here so every call
site gets it for free instead of special-casing zero separately four times.

Applied at each of the four read-only sites by appending the class to the
existing `font-mono tabular-nums` span/cell that wraps `formatAmount(...)`.
None of those sites need to change *what* they render, only add a
conditional class next to what's already there.

### Decision: underline is a second, independent class, applied only where `kind` is in scope

`entries.index.tsx`'s table row and `accounts.$accountId.index.tsx`'s
recent-entries row both already have the full `Entry` object in hand, so
they add `entry.kind === "balance_adjustment" ? "underline" : ""` alongside
`amountColorClass(entry.amount)`. The accounts-overview list and the
account-detail balance figure have no single entry backing them (a balance
is a computed sum across many entries), so they only ever get the color
class, never the underline — the proposal's "entry rows only" scope is
enforced simply by which components have an `Entry` to read `kind` from,
not by a special flag.

### Decision: the transaction amount field splits into magnitude state + sign state, combined only at submit

Today: `const [amount, setAmount] = useState("")`, parsed once via
`inputToAmount(amount)` at submit. New shape, for the `kind === "transaction"`
branch only:

```ts
const [amount, setAmount] = useState("");       // magnitude text only, never signed
const [negative, setNegative] = useState(true); // default minus, per the ask

function handleAmountChange(raw: string) {
  if (raw.includes("-")) {
    setNegative(true);
    raw = raw.replaceAll("-", "");
  }
  setAmount(raw);
}
```

Handling the sign inside `onChange` (rather than intercepting keydown)
covers typing *and* pasting a `-` uniformly, and satisfies the exact rule
given: a `-` anywhere in the new value always forces `negative` to `true`
and is always stripped, whether or not the field was already negative —
"if it is minus, the character is removed/not entered" falls out of the
same branch instead of needing a separate "already negative" case. There is
no equivalent interception for `+` — it's simply not a valid character for
a magnitude-only field and is filtered the same way any other non-numeric
input already would be (existing `inputMode="decimal"` input hygiene is
unchanged).

Submit combines the two: `const magnitude = inputToAmount(amount)`, `if
(magnitude === null || magnitude === 0) → invalidField("amount")` (new
zero-check, transaction-only), else `finalAmount = negative ? -magnitude :
magnitude`. Editing an existing transaction initializes both pieces of
state from the stored (signed) amount: `negative = entry.amount < 0`,
`amount = amountToInput(Math.abs(entry.amount))`.

### Decision: field/button styling reads sign *and* magnitude, not sign alone

The pastel field background and the button's accent color both key off a
three-way resolution — matching `amountColorClass`'s zero exception, so a
freshly-focused empty (or `0`) field never shows red just because the
toggle currently says minus:

```ts
const magnitudeValue = Number(amount) || 0;
const sign = magnitudeValue === 0 ? "zero" : negative ? "negative" : "positive";
```

- `zero`: today's plain `inputClass` (transparent bg, default border/text)
  — unchanged from what the field looks like now.
- `negative`: pastel field — `bg-red-100 text-red-800 border-red-200
  dark:bg-red-900/40 dark:text-red-300 dark:border-red-900/50` (same
  light-bg/dark-text formula `AccountStatus`'s badges already use).
- `positive`: the emerald equivalent — `bg-emerald-100 text-emerald-800
  border-emerald-200 dark:bg-emerald-900/40 dark:text-emerald-300
  dark:border-emerald-900/50`.

The toggle button is a small square/circle button rendered before the
input (one flex row, so the two form one visual control), always showing
`−` or `+` per `negative`, and always solid-colored regardless of
magnitude (`bg-red-600 text-white hover:bg-red-700` /
`bg-emerald-600 text-white hover:bg-emerald-700`) — it's the accent the
field's pastel is a tint of, present even while the field itself is
sitting neutral at `0`, since the button's job is "what will typing a
digit produce," not "what's currently entered." Clicking it flips
`negative`; it carries a translated `aria-label` (new i18n key) since its
content is a bare glyph.

### Decision: `balance_adjustment`'s branch is untouched, selected by `kind`

`entries.new.tsx` already has `kind` as form state (radio); the amount
field renders the new toggle+input for `kind === "transaction"` and falls
straight through to today's existing `<input>` JSX, unchanged, for `kind
=== "balance_adjustment"` — no shared component, no color, `0` still valid
(`inputToAmount` alone, no extra zero check). `entries.$entryId.edit.tsx`
branches the same way on the entry's immutable `entry.kind`. This is a
plain conditional render inside each route file, not a prop on a shared
component — keeps the "completely unaffected" requirement obviously true
by inspection rather than threading a `disabled`-style flag through shared
code.

### Decision: factor the transaction amount field into a small shared component

Both routes need identical magnitude/sign state, the same `onChange`
sign-interception, and the same three-way styling — worth pulling into one
`frontend/src/components/SignedAmountInput.tsx` (or similar) taking
`{ magnitude, onMagnitudeChange, negative, onToggleSign, currency, invalid
}` as props, rather than duplicating ~30 lines of logic across both route
files the way the rest of the form fields are already duplicated. This
mirrors how `TagInput.tsx` is already a small shared component used by
both entry-form routes for a similarly self-contained piece of the form.

## Migration Plan

Frontend-only, no data migration. Order of work:

1. `lib/amount.ts`: add `amountColorClass`.
2. Read-only sites: `entries.index.tsx`, `accounts.index.tsx`,
   `accounts.$accountId.index.tsx` — apply color (+ underline where `kind`
   is available).
3. New `SignedAmountInput.tsx` component.
4. Wire it into `entries.new.tsx` (transaction branch; default `negative =
   true`) and `entries.$entryId.edit.tsx` (transaction branch; initialize
   from `entry.amount`'s sign), including the zero-amount submit check.
5. i18n keys (`en.json` first, then `de.json`).

## Verification

- `cd frontend && pnpm lint && pnpm exec tsc && pnpm build` (no backend
  changes, no new test framework exists in this package per
  `frontend/AGENTS.md`).
- Manual pass: negative/zero/positive amounts render red/black/green in
  all four read-only spots; a `balance_adjustment` entry's amount is
  underlined in the entries list and in an account's recent entries, a
  `transaction`'s is not; creating a transaction starts sign at minus,
  typing or pasting `-` never inserts the character and always leaves the
  field on minus; toggling the button flips sign and both the field tint
  and the button's own accent update; a `0`-magnitude transaction is
  blocked on submit with a validation message; a `balance_adjustment`
  entry's amount field looks and behaves exactly as before (no button, no
  color, `0` accepted).
