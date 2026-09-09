## Why

Accounts and categories are today distinguishable only by their text label.
On the accounts overview, the category tree, the entry ledger, and the
reports tables they read as an undifferentiated list, and there is no quick
visual anchor for "my everyday checking account" or "Groceries". Giving each
account and each category an optional icon and an optional colour — shown as
a small badge before the name everywhere they appear — makes those surfaces
scannable at a glance, at essentially no cost to users who don't want it
(both fields stay optional and default to nothing).

## What Changes

- `accounts` gains two optional fields, `icon` and `color`, on the `Account`
  resource. Both are short opaque strings the backend stores and echoes but
  never interprets (validated only for shape: `^[a-z0-9-]{1,40}$`). They are
  independent — either, both, or neither may be set. On update, an empty
  string clears a field.
- `entry-categories` gains the same two optional fields, `icon` and `color`,
  on the `Category` resource, with identical semantics.
- The seeded starter categories every new user gets (Salary, Groceries,
  Rent, …) each ship with a sensible default `icon` and `color`, so a fresh
  account looks considered rather than blank. Account types are **not**
  touched — icon and colour live on the account itself, not its type — and
  there is no account seeder, so no default-account changes.
- New web-client capability for **entity icon rendering**: a curated,
  grouped icon set (drawn from `lucide-react`, already a dependency) and a
  fixed colour palette (the `dataviz` categorical palette), a shared
  `<EntityIcon>` badge + label component used wherever an account or
  category name is shown, an `IconColorPicker` (icon grid grouped under
  headings, colour swatches beside it) reused by the account and category
  forms, and graceful fallback to just the name when a stored `icon`/`color`
  value is unknown to the current client.
- Wherever the web client shows an account or category **by name** — the
  accounts overview, account detail, home account cards, the category tree,
  the entry ledger, the reports tables — the name SHALL be prefixed with the
  entity's icon/colour badge when one is set. Native `<select>` / `<option>`
  controls are the sole exception: they stay text-only (an `<option>` cannot
  hold an icon), so the entry form's account and category pickers and the
  filter dropdowns are unchanged.
- The account create/edit form and the category create and edit forms each
  gain the `IconColorPicker`.

## Capabilities

### New Capabilities

- `web-client-entity-icons`: the shared client-side machinery for entity
  icons and colours — the curated grouped icon set, the fixed colour
  palette with light/dark values, the `<EntityIcon>` badge/label component
  and its unknown-value fallback, the reusable `IconColorPicker`, and the
  rule that account/category names render icon-before-label on every surface
  except native `<select>`/`<option>`.

### Modified Capabilities

- `accounts`: `Account` gains optional `icon` and `color` (opaque short
  strings, shape-validated, independent, cleared by `""` on update);
  returned on every `Account` response and accepted on create and update.
- `entry-categories`: `Category` gains optional `icon` and `color` with the
  same semantics; the seeded default-categories requirement is amended so
  each seeded category also carries a fixed default `icon` and `color`.
- `web-client-accounts`: the account create/edit form includes the
  icon/colour picker; the accounts overview, detail page, and home account
  cards render the account's icon/colour badge before its title.
- `web-client-categories`: the category create form and the edit form
  include the icon/colour picker; the category tree renders each node's
  icon/colour badge before its name.

## Impact

- **Backend**: new migration adding nullable `icon` / `color` `text` columns
  to `accounts` and `categories`; `internal/account` and `internal/category`
  `New`/`Update` structs, validation, and both storage backends (Postgres +
  in-memory) carry the two fields; `category.DefaultNames` (a `[]string`)
  becomes a slice of `{Name, Icon, Color}` and `Service.SeedDefaults` writes
  the icon/colour too.
- **API contract**: `openapi/openapi.yaml` — `Account`, `AccountCreate`,
  `AccountUpdate`, `Category`, and `CategoryWrite` schemas each gain `icon`
  and `color`; regenerate `backend/openapi.yaml` and
  `frontend/src/api/schema.d.ts` in the same change.
- **Frontend**: new modules for the curated icon set, the colour palette,
  `<EntityIcon>` / entity label components, and `IconColorPicker`;
  `AccountForm.tsx` and `categories.tsx` wire the picker; the accounts
  overview, account detail, `AccountCard.tsx`, the category tree, the entry
  ledger rows, and the reports tables switch to the shared label component;
  new i18n keys (icon-group headings, picker labels) in `en.json` / `de.json`.
- **No new runtime dependencies** — `lucide-react` and the `dataviz` palette
  are already in the repo.
