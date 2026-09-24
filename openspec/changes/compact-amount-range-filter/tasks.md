# Tasks

## 1. Compact amount-range control

- [x] 1.1 Replace the persistent amount-bound grid wrapper with a compact popover trigger and small overlay containing both controls; verify the pair opens and closes together.
- [x] 1.2 Add a precision-trimmed stored-amount display helper within the component and verify whole and fractional values preserve meaningful digits without trailing zeroes.
- [x] 1.3 Add translated trigger summary and placeholder text in English and German, and verify no new hardcoded user-facing text is introduced.

## 2. Verification

- [x] 2.1 From `frontend/`, run `pnpm lint`, `pnpm exec tsc --noEmit`, and `pnpm build`; verify all checks pass.
- [x] 2.2 Run `openspec validate compact-amount-range-filter --strict` from the repository root and verify the change passes validation.
