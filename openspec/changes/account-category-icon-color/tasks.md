## 1. API contract

- [x] 1.1 In `openapi/openapi.yaml`, add optional `icon` and `color` string
  properties (with the `^[a-z0-9-]{1,40}$` pattern and a short description
  noting they are opaque presentation tokens) to the `Account`,
  `AccountCreate`, and `AccountUpdate` schemas.
- [x] 1.2 In `openapi/openapi.yaml`, add the same `icon` and `color`
  properties to the `Category` and `CategoryWrite` schemas.
- [x] 1.3 Run `cd backend && go generate ./...` to sync
  `backend/openapi.yaml`, and `cd frontend && pnpm generate:api` to
  regenerate `frontend/src/api/schema.d.ts`; commit all three files
  together.
- [x] 1.4 Confirm the spec still lints (`spectral`) and the `contract` job
  would pass (no drift between the source and the two generated copies).

## 2. Database

- [x] 2.1 Add migration
  `backend/internal/storage/postgres/migrations/00NN_account_category_icon_color.sql`
  adding nullable `icon text` and `color text` columns to `accounts` and to
  `categories` (no default; existing rows become `NULL`).

## 3. Backend — account domain

- [x] 3.1 Add `Icon` and `Color` fields to `account.Account` (JSON
  `icon,omitempty` / `color,omitempty`), to `account.New`, and to
  `account.Update` (as `*string` so "omitted" and "set to empty" differ,
  matching how other clearable fields are modelled).
- [x] 3.2 Add shape validation (`^[a-z0-9-]{1,40}$` or empty) for `icon` and
  `color` in `validateNew` and `validateUpdate`, returning `ErrInvalidValue`
  (→ `422`) on failure.
- [x] 3.3 Apply the update semantics in `account.Service`: a nil pointer
  leaves the field untouched; a non-nil empty string clears it; a non-nil
  non-empty string sets it.
- [x] 3.4 Thread `icon`/`color` through the Postgres store
  (`storage/postgres/account.go`) INSERT / UPDATE / SELECT column lists and
  row scans.
- [x] 3.5 Thread `icon`/`color` through the in-memory store
  (`storage/memory/account.go`).
- [x] 3.6 Ensure `httpapi/account.go` request decoding and response
  encoding carry the two fields.
- [x] 3.7 Extend `account` service/handler/store tests: create with
  icon+colour, create with neither, malformed value rejected (`422`),
  clear-on-update via `""`, unrelated update preserves both.

## 4. Backend — category domain

- [x] 4.1 Add `Icon` and `Color` fields to `category.Category`,
  `category.New`, and `category.Update` (`*string` for the update), mirroring
  the account domain.
- [x] 4.2 Add the same shape validation in the category validators.
- [x] 4.3 Apply the omit / clear / set update semantics in
  `category.Service`.
- [x] 4.4 Thread `icon`/`color` through `storage/postgres/category.go` and
  `storage/memory/category.go`.
- [x] 4.5 Ensure `httpapi/category.go` decoding/encoding carry the fields.
- [x] 4.6 Change `category.DefaultNames` from `[]string` to a slice of
  `{Name, Icon, Color string}` with a fixed icon and palette token per
  seeded category (see design.md §6), and update `Service.SeedDefaults` to
  persist `icon` and `color` alongside `name` and `sort_order`.
- [x] 4.7 Extend `category` tests: field CRUD like the account tests, plus a
  seeding test asserting every seeded category comes back with a non-empty
  `icon` and `color`.

## 5. Frontend — shared entity-icon machinery

- [x] 5.1 Add `src/lib/entityColors.ts`: the palette token → `{ light, dark }`
  map for `blue, orange, aqua, yellow, magenta, green, violet, red` (values
  from design.md §2), plus a resolver that picks light/dark per active theme
  and returns undefined for an unknown token.
- [x] 5.2 Add `src/lib/entityIcons.ts`: the curated icon set as ordered
  groups, each `{ headingKey, icons: { token, Component }[] }`, with every
  `Component` a static named import from `lucide-react`; export a
  `token → Component | undefined` lookup and the group list.
- [x] 5.3 Add `src/components/EntityIcon.tsx`: props `{ icon?, color?, size? }`
  → coloured/neutral badge with optional glyph; renders `null` when both are
  unset; unknown icon token → no glyph, unknown colour token → treated as
  unset.
- [x] 5.4 Add `src/components/AccountLabel.tsx` and
  `src/components/CategoryLabel.tsx` rendering `<EntityIcon/>` then the
  name, with shared spacing/alignment.
- [x] 5.5 Add `src/components/IconColorPicker.tsx`: grouped icon grid beside
  the colour swatches, independent set/clear for each, i18n-keyed labels;
  controlled value `{ icon: string; color: string }` where `""` means
  unset.
- [x] 5.6 Add i18n keys to `src/i18n/locales/en.json` then
  `src/i18n/locales/de.json`: icon-group headings, picker labels ("Icon",
  "Colour", "None"/"Clear").

## 6. Frontend — wire the picker into forms

- [x] 6.1 `AccountForm.tsx`: add `icon` and `color` to `AccountFormValues`
  and `emptyAccountForm`, render `IconColorPicker`, and include the values
  in the submitted body (send `""` to clear on edit; omit / send absent for
  a never-set field on create).
- [x] 6.2 `accounts.$accountId.edit.tsx`: seed the form's `icon`/`color`
  from the loaded account.
- [x] 6.3 `categories.tsx`: add `icon`/`color` state to the inline create
  form and the rename form, render `IconColorPicker` in both, and send the
  values on `POST` / `PATCH /api/categories`.

## 7. Frontend — render badges on read surfaces

- [x] 7.1 `accounts.index.tsx`: replace the bare `account.title` render with
  `<AccountLabel account={account} />`.
- [x] 7.2 `AccountCard.tsx`: use `<AccountLabel />` before the title.
- [x] 7.3 `accounts.$accountId.index.tsx`: use `<AccountLabel />` in the
  detail header.
- [x] 7.4 `categories.tsx`: render `<CategoryLabel category={node} />` for
  each tree node in place of `{node.name}`.
- [x] 7.5 `entries.index.tsx`: show the account and category badge before
  their names in the ledger rows (not in the `<select>` filter).
- [x] 7.6 `reports.tsx`: show the account (and category, where shown) badge
  before names in the results table (not in the `<select>` filters).
- [x] 7.7 Confirm the entry form account/category `<select>` controls and
  the ledger/reports filter dropdowns remain text-only.

## 8. Verification

- [x] 8.1 `cd backend && go test ./...` passes.
- [x] 8.2 `cd frontend && pnpm lint && pnpm exec tsc && pnpm build` all
  pass; `out/index.html` is written.
- [ ] 8.3 Manual pass: create an account and a category with icon+colour,
  with colour only, with neither; verify badges on the overview, detail,
  home cards, category tree, ledger, and reports; verify dropdowns are
  unchanged; toggle the theme and confirm palette colours adapt; confirm a
  fresh user's seeded categories show badges.
- [x] 8.4 Update `frontend/AGENTS.md` (and `backend/AGENTS.md` if the seed
  section warrants it) to mention the entity-icon machinery and the
  icon/colour fields.
