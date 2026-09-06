## Why

`AccountForm.tsx`'s `financial_institute` field is a plain free-text input.
A visitor with several accounts at the same bank (common in a family
finances app — checking, savings, a kid's account, all at one institute)
has to retype the exact name on every new account, with no help recalling
how they spelled it last time. The account's owner already has that
history sitting in their own account list; the form just doesn't use it.

## What Changes

- `AccountForm.tsx` (shared by `/accounts/new` and `/accounts/{id}/edit`)
  fetches the caller's own accounts (`GET /api/accounts`, already used
  elsewhere in the app) alongside its existing account-types fetch, and
  derives the distinct, non-empty `financial_institute` values already on
  those accounts — exact-string dedupe, sorted alphabetically.
- The financial institute field shows these as clickable suggestion chips
  (matching `TagInput`'s existing chip visual language) whenever it is
  focused, narrowing to substring matches as the visitor types. Clicking a
  chip sets the field to that value. The field stays plain free text —
  typing an institute not in the list is still accepted.
- No backend or API contract changes: everything needed is already
  returned by `GET /api/accounts`, and the field itself is unchanged
  (still an optional string). This is a client-only, additive UI change.

## Capabilities

### Modified Capabilities

- `web-client-accounts`: the "Creating and editing an account" requirement
  gains the financial-institute suggestion behavior described above.

## Impact

- **Code**: `frontend/src/components/AccountForm.tsx` only — a second
  parallel fetch in its existing `useEffect`, a derived suggestion list,
  and new markup for the chip row under the financial institute field.
  New i18n keys in `frontend/src/i18n/locales/{en,de}.json` if any new
  user-facing copy is needed (e.g. none is strictly required — the chips
  are just institute names — but double-check during implementation).
- **API contract**: none. No schema or path changes.
- **Spec**: delta on `web-client-accounts` only.
