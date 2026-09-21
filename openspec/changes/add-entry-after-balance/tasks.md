## 1. Contract and backend

- [x] 1.1 Add required `after_balance` to the Entry OpenAPI schema.
- [x] 1.2 Populate it for canonical and per-account-leg PostgreSQL reads.
- [x] 1.3 Mirror the ledger-order calculation in the in-memory store.
- [x] 1.4 Regenerate committed API artifacts and run backend tests.

## 2. Read-only entry view

- [x] 2.1 Render the after-entry balance for every entry summary.
- [x] 2.2 Add English and German labels and clarify the adjustment reading.
- [x] 2.3 Run frontend lint, type-check, and build.

## 3. Verification

- [x] 3.1 Run contract drift checks.
- [ ] 3.2 Capture the updated entry summary in a runnable browser environment.
