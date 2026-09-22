## 1. Shared icon

- [x] 1.1 Extract the repeat-arrows SVG currently inlined in
      `src/components/RecurringTransactionBadge.tsx` into a small
      `RecurringIcon` component (props: `width`/`height`, default matching
      current usage) in the same file or a new
      `src/components/RecurringIcon.tsx`, and use it from
      `RecurringTransactionBadge`. No visual change to the existing badge.

## 2. `entries.$entryId.edit.tsx`: dirty tracking

- [x] 2.1 Add a `dirty` boolean state, initialized `false`. Set it `true`
      from the existing field setters (`setAmount`, `setTransactionAmount`,
      `setTransactionNegative`, `setBookingTimestamp`, `setTitle`,
      `setDescription`, `setCategoryId`, `setCounterparty`, `setLocation`,
      `setTagNames`, `setSelectedAccountId`, `setRecurringTransactionId`) —
      e.g. via a small wrapping helper used in each `onChange`, not a
      value-diff against the loaded entry.
- [x] 2.2 Reset `dirty` to `false` right after the entry finishes loading
      (end of the existing load `useEffect`) and right after a successful
      save in `performSubmit`.

## 3. `entries.$entryId.edit.tsx`: the action itself

- [x] 3.1 Parameterize `performSubmit`'s post-save navigation (currently
      hardcoded `navigate({ to: "/entries", search: { account_id:
      selectedAccountId } })`) so a caller can request landing on
      `/recurring/new?from_entry_id=<id>` instead, without duplicating the
      validation/PATCH/account-change-confirmation logic.
- [x] 3.2 Add the icon-only action beside the "Linked recurring
      transaction" `<select>` (wrapped in a `flex items-center gap-2` row
      together, mirroring the account field's unlock-button layout),
      using a local `PlusGlyph` ("+" icon, matching the shape already
      duplicated locally in `accounts.index.tsx`/`AccountCard.tsx`) rather
      than `RecurringIcon` — the field's own label already supplies the
      "recurring" context. Shown only when `canEdit && entry.kind ===
      "transaction"`; since this sits inside the form's `<fieldset
      disabled={!canEdit}>`, the `canEdit` check must still be explicit on
      the `Link` variant (an `<a>` isn't reached by a disabled fieldset,
      unlike the `<button>` variant).
      - Clean (`!dirty`): renders as a `Link`/navigate to
        `/recurring/new?from_entry_id={entryId}`, `aria-label`/`title` from
        `entries.edit.createRecurring`.
      - Dirty: renders as a `type="button"` that runs `handleSubmit`'s
        validation and, on the account-changed path, opens the existing
        `confirmingAccountChange` dialog exactly as Save does; on a direct
        (no account change) or confirmed save, calls `performSubmit` with
        the task-3.1 destination override. `aria-label`/`title` from
        `entries.edit.saveAndCreateRecurring`. Disabled while `submitting`,
        matching Save's existing disabled state.
- [x] 3.3 Confirm the confirmation-dialog "confirm" button
      (`confirmingAccountChange`'s `onClick`) still knows which destination
      to use when it was opened via this action vs. the normal Save button
      — thread the pending destination through the same state that already
      holds `pendingAmount`.

## 4. `recurring.new.tsx`: `from_entry_id` prefill

- [x] 4.1 Add `validateSearch` for an optional `from_entry_id` string
      param, mirroring `entries.new.tsx`'s `NewEntrySearch` pattern.
- [x] 4.2 When present, fetch `GET /api/entries/{id}` on mount and prefill
      `RecurringTransactionForm`'s `initial`: `account_id` (also passed as
      `accountLocked`), `title`, `description`, `category_id`,
      `counterparty`, `location`, tag names (resolved against the already
      fetched `GET /api/tags` the way `entries.$entryId.edit.tsx` does),
      `amountMagnitude`/`negative` from the entry's signed `amount`, and
      `starts_on` set to the entry's `booking_timestamp` date portion
      (reuse or mirror `entries.$entryId.edit.tsx`'s local-date handling,
      date-only, no time-of-day).
- [x] 4.3 Only fetch/prefill when `entry.kind === "transaction"` data is
      expected — if the fetched entry is a `balance_adjustment` (shouldn't
      normally happen since task 3.2 only ever links here from a
      transaction entry, but guard against a hand-edited URL), fall back to
      the empty form rather than prefilling nonsensical fields.

## 5. `recurring.new.tsx`: auto-link on success

- [x] 5.1 After a successful `POST /api/recurring-transactions` reached via
      `from_entry_id`, call `PATCH /api/entries/{id}` with
      `recurring_transaction_id` set to the new template's id.
- [x] 5.2 Navigate to `/recurring` once the link `PATCH` settles
      (success or failure — per design.md's risk mitigation, a failed link
      doesn't block navigation since the template itself was already
      created); when not reached via `from_entry_id`, keep the existing
      unconditional navigate-to-`/recurring` behavior unchanged.

## 6. i18n

- [x] 6.1 Add `entries.edit.createRecurring` and
      `entries.edit.saveAndCreateRecurring` to `en.json` first, then
      `de.json`.

## 7. Verification

- [x] 7.1 Frontend: `pnpm lint`, `pnpm exec tsc`, `pnpm build` (from
      `frontend/`).
- [x] 7.2 Manually exercise, in a real browser against the dev server: the
      action is hidden on a balance-adjustment entry and for a read-only
      visitor; a clean transaction entry navigates straight to
      `/recurring/new?from_entry_id=...` with fields prefilled and the
      account locked; a dirty entry shows the "save and create" label,
      saves, and only then navigates; an account change from this action
      still shows the confirmation dialog; submitting the prefilled
      recurring form creates the template, links back to the origin entry
      (verify via the entry's "Linked recurring transaction" field or its
      ledger badge), and lands on `/recurring`.
