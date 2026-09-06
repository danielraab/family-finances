## 1. Frontend: `AccountForm.tsx`

- [x] 1.1 Add a `GET /api/accounts` fetch (`Promise.all` alongside the
  existing `GET /api/account-types` fetch in the mount `useEffect`);
  derive `institutes`: distinct, non-empty `financial_institute` values
  across the fetched accounts, deduplicated by exact string match, sorted
  alphabetically (`localeCompare`).
- [x] 1.2 Track whether the financial institute field is focused (state or
  CSS `:focus-within`, whichever reads simplest); render a chip row below
  the field, visible only while focused, listing `institutes` filtered by
  a case-insensitive substring match against the field's current value
  (all of them when the value is empty).
- [x] 1.3 Wire chip clicks to `set("financial_institute", value)`,
  replacing the field's current value — no array/append logic, no
  self-exclusion for the account currently being edited.
- [x] 1.4 Visual style matches `TagInput.tsx`'s existing suggestion-chip
  pattern (rounded-full outline buttons) for consistency, without
  extracting a shared component.

## 2. i18n

- [x] 2.1 Add new keys to `frontend/src/i18n/locales/en.json` first, then
  `de.json`, only if the implementation ends up needing user-facing copy
  beyond the institute names themselves (e.g. none expected — confirm
  during implementation). Confirmed: no new copy needed, the chips render
  only the institute names themselves.

## 3. Verify

- [x] 3.1 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build`.
- [ ] 3.2 Manual pass: as a visitor with two or more accounts sharing a
  `financial_institute` value, open `/accounts/new`, focus the field, and
  confirm the shared value (and every other distinct value) appears as a
  chip; type part of it and confirm the chip list narrows
  (case-insensitively); click a chip and confirm the field fills with its
  exact value; blur the field and confirm the chips hide. Repeat on
  `/accounts/{id}/edit` for an existing account, including clicking the
  chip matching that account's own current value. Confirm a visitor with
  no accounts (or none with `financial_institute` set) sees no chip row.
  Confirm typing an institute name that matches nothing still submits
  successfully.
  **Not run**: this sandbox has no Docker daemon, so the Postgres-backed
  backend from `compose.yaml` can't be started to log in and exercise the
  real form. Needs a manual pass in an environment that can run the app.

## 4. Spec sync

- [x] 4.1 Apply this change's `specs/web-client-accounts` delta onto
  `openspec/specs/web-client-accounts/spec.md` by hand (the `openspec` CLI
  is unavailable in this environment, as for prior changes).
