## Context

Accounts (`internal/account`, `accounts` spec) and categories
(`internal/category`, `entry-categories` spec) are per-user records with a
text label and little else to tell them apart in a list. The frontend
already depends on `lucide-react` (used in `FeatureOverview.tsx` via static
named imports) and already has a validated categorical colour palette in the
`dataviz` skill (`references/palette.md`) that its hand-rolled charts draw
series colours from. This change adds an optional icon and an optional
colour to each account and each category and renders them as a badge before
the name on every client surface that shows one.

Constraints that shape the design:

- The backend owns all persistence and is the frontend's only backend; the
  API is spec-first (`openapi/openapi.yaml` hand-written, two generated
  artifacts committed, a CI `contract` job that fails on drift).
- The backend has no blob storage and no concept of an icon set or a
  palette — nor should it gain one.
- The frontend must keep `lucide-react` tree-shakeable; importing the whole
  icon set to resolve an arbitrary name would bloat the bundle.
- Several account/category pickers are native `<select>` elements; an
  `<option>` cannot contain an icon.
- Both new fields must be genuinely optional — a user who ignores them pays
  nothing.

## Goals / Non-Goals

**Goals:**

- Add optional `icon` and `color` to `Account` and `Category`, independent
  of each other, settable on create and update, clearable on update.
- Render an icon/colour badge before the name on the accounts overview,
  account detail, home account cards, the category tree, the entry ledger,
  and the reports tables.
- Ship a reusable icon/colour picker used by the account form and the
  category create/edit forms, with the icon grid grouped under headings.
- Seed the default categories with sensible icons and colours.
- Keep the backend ignorant of what any specific `icon`/`color` value means.

**Non-Goals:**

- Icons or colours on account **types**, tags, or entries.
- Any icon inside a native `<select>`/`<option>` — those stay text-only.
- User-uploaded icons/images, or free-form hex colours.
- A colour without a corresponding chart change — categories already feed
  charts; wiring category colour into chart series colour is a possible
  follow-up, not part of this change.
- Backfilling icons/colours onto accounts (there is no account seeder) or
  onto categories created before this change.

## Decisions

### 1. The backend stores `icon` and `color` as opaque short strings

Two nullable `text` columns on `accounts` and `categories`. The backend
validates **shape only** — `^[a-z0-9-]{1,40}$`, or empty — and otherwise
stores and echoes the value untouched. It never enumerates valid icon names
or palette tokens.

- **Why:** the icon set and the palette are frontend presentation concerns
  that will change (add an icon, retune a colour) far more often than the
  data model should. Keeping them out of `openapi/openapi.yaml` means those
  changes are frontend-only — no spec edit, no `go generate`, no
  `pnpm generate:api`, no `contract` job churn each time. It mirrors how
  `currency` is validated for shape, not against a canonical list.
- **Alternative considered — an `enum` in the OpenAPI schema:** every icon
  added later becomes a contract change touching three files plus CI.
  Rejected as long-term friction for a cosmetic field.
- **Alternative considered — a dedicated `icons`/`colors` lookup table:**
  pure overhead; there is no relational data hanging off an icon.
- **Consequence:** an `icon`/`color` value the current client doesn't
  recognise (older bundle, retired icon) is possible. The client handles it
  by falling back to rendering just the name (decision 4). The backend does
  not care.

### 2. `color` is a palette **token**, not a hex value

The stored string is a hue name from a fixed set — `blue`, `orange`,
`aqua`, `yellow`, `magenta`, `green`, `violet`, `red` — the eight slots of
the `dataviz` categorical palette. The frontend maps each token to a
light hex and a dark hex in one module.

| token | light | dark |
|-------|-------|------|
| blue | `#2a78d6` | `#3987e5` |
| orange | `#eb6834` | `#d95926` |
| aqua | `#1baf7a` | `#199e70` |
| yellow | `#eda100` | `#c98500` |
| magenta | `#e87ba4` | `#d55181` |
| green | `#008300` | `#008300` |
| violet | `#4a3aa7` | `#9085e9` |
| red | `#e34948` | `#e66767` |

- **Why a token, not hex:** a token adapts to light/dark (a raw hex can't);
  it is guaranteed legible in both themes because the palette is validated
  for exactly that; and it is the same identity a chart series uses, so a
  later "colour category slices by the category's colour" change is a
  direct lookup with no colour math. A token also keeps the opaque-string
  contract (decision 1) — it's still just `^[a-z0-9-]{1,40}$`.
- **Alternative considered — free-form hex picker:** more expressive, but
  users can pick unreadable or theme-broken colours, and the value can't
  harmonise with charts. Rejected.
- **Palette size:** start with exactly the eight validated slots. Adding a
  neutral ("slate") later is a one-line frontend change precisely because
  of decision 1.

### 3. The frontend resolves icons through an explicit curated map, grouped

A single module declares the icon set as groups, each group a heading key
plus an ordered list of `{ token, Component }` where `Component` is a
**statically imported** `lucide-react` icon:

```
Banking      wallet · piggy-bank · landmark · credit-card · banknote · coins · vault
Income       briefcase · hand-coins · trending-up · gift · receipt
Home & bills house · zap · droplet · flame · wifi · phone · wrench · trash-2
Food         shopping-cart · shopping-bag · utensils · coffee · apple · beer
Transport    car · bus · train · plane · bike · fuel · circle-parking
Life         heart · activity · gamepad-2 · film · music · dumbbell · book-open · graduation-cap · paw-print · baby
Transfers    arrow-left-right · repeat · split · send
```

(~45 icons; final list tuned during implementation.)

- **Why explicit static imports:** it's the only option that keeps
  `lucide-react` tree-shaken to just the icons actually referenced.
  `import * as icons` ships all ~1,600; `lucide-react`'s dynamic import
  helper code-splits but renders asynchronously (icon pops in a frame
  late). `FeatureOverview.tsx` already uses static named imports — same
  pattern.
- **Why grouped:** ~45 icons in a flat grid is a hunt; headings ("Banking",
  "Food", …) let a user jump to the right neighbourhood. Group headings are
  i18n keys.
- **The map is also the allow-list:** the picker only offers what's in the
  map; a stored token not in the map hits the fallback (decision 4).

### 4. `<EntityIcon>` and the label components; unknown-value fallback

- `IconColorPicker` opens from a trigger styled like a form field (showing
  the current `EntityIcon` badge, or a dashed placeholder when unset) into a
  `@headlessui/react` `Popover` panel — the colour swatch row above the
  grouped, scrollable icon grid. A `Popover` (not inline, not a `Dialog`) so
  it composes inside the `/categories` edit `Dialog` without a
  dialog-in-dialog; `PopoverPanel anchor="bottom start"` portals the panel
  so the enclosing form's `overflow` never clips it.
- `<EntityIcon icon color size>` renders the badge: a rounded square filled
  with the token's theme-appropriate colour (or a neutral surface tint when
  `color` is unset) containing the `lucide` icon in `currentColor` white/ink
  (or nothing when `icon` is unset — a plain colour chip). If **both** are
  unset it renders nothing.
- `<AccountLabel account>` / `<CategoryLabel category>` render
  `<EntityIcon …/>` then the name, with consistent gap and alignment. These
  are the components every surface uses.
- **Fallback:** an `icon` token not in the curated map → no glyph (chip only
  if `color` is set, else nothing). A `color` token not in the palette map →
  treated as unset. The name always renders. No error, no console noise —
  an older client simply shows less decoration.

### 5. Native `<select>` surfaces stay text-only

The entry form's account picker and category picker, and the account filter
dropdowns on the entry ledger and reports, are native `<select>`. An
`<option>` cannot hold markup, so these keep showing the bare name.

- **Why not swap them for a headless `Listbox`:** that's 4–5 controls
  (including the category-tree `<select>` fed by `flattenCategoryTree`)
  worth of interaction rework for a cosmetic gain in a transient menu.
  Out of proportion to the value; explicitly deferred.
- The spec states the exception so the inconsistency is intentional and
  documented, not a bug.

### 6. Seeded default categories carry icon + colour

`category.DefaultNames` (`[]string`) becomes `[]struct{ Name, Icon, Color string }`,
e.g. `Salary → hand-coins / green`, `Groceries → shopping-cart / orange`,
`Rent → house / blue`, `Utilities → zap / yellow`, `Transportation → car / aqua`,
`Entertainment → gamepad-2 / magenta`, `Health → heart / red`,
`Other → circle-dashed / violet`. `Service.SeedDefaults` writes the two new
fields alongside `name`/`sort_order`. The `entry-categories` seeding
requirement is amended to say each seeded category also has a fixed default
icon and colour (still English-independent, still non-recurring, still
freely editable afterwards).

- Account types are **not** seeded with icon/colour — they don't have the
  fields. There is no account seeder, so no default account gets a badge.

### 7. One migration, both tables

A single migration (`00NN_account_category_icon_color.sql`) adds
`icon text` and `color text` (both nullable, no default) to `accounts` and
to `categories`. Backwards-compatible: existing rows get `NULL`, which the
API represents as the field being absent (`omitempty`), which the client
renders as no badge.

## Risks / Trade-offs

- **[Contract omits the valid value sets]** → An out-of-date client can
  store a token a newer client shows and an older one doesn't. Mitigated by
  the always-render-the-name fallback (decision 4); the failure mode is
  "less decoration", never a broken view. Accepted deliberately as the cost
  of keeping icon/palette churn out of the API contract.
- **[Curated icon set won't contain everyone's ideal icon]** → ~45 icons is
  a compromise. Adding one is a one-line change to the curated map with no
  backend or contract impact, so the set can grow based on feedback.
- **[Dropdowns look inconsistent with the rest of the UI]** → Named as an
  explicit spec exception. Revisitable as a later `Listbox` migration if it
  grates in practice.
- **[Colour-only badges (no icon) could read as noise]** → The picker
  presents icon and colour together and a colour chip with no icon is a
  deliberate, supported state (useful for categories); if it proves ugly in
  the tree, the label component can require an icon for the chip to show —
  a component-local tweak, no data change.
- **[i18n lag]** → New group-heading and picker keys land in `en.json`
  first; `de.json` added same change. CI's `i18n-coverage` is informational
  only, so a brief lag wouldn't block, but there's no reason to incur it.

## Migration Plan

1. Edit `openapi/openapi.yaml` (`Account`, `AccountCreate`, `AccountUpdate`,
   `Category`, `CategoryWrite` → add `icon`, `color`); run
   `cd backend && go generate ./...` and `cd frontend && pnpm generate:api`;
   commit all three files together.
2. Add the migration; extend `internal/account` and `internal/category`
   (`New`/`Update`, shape validation, Postgres + in-memory stores).
3. Change `category.DefaultNames` to the struct slice; update
   `Service.SeedDefaults`.
4. Frontend: curated icon map, palette module, `<EntityIcon>` +
   label components, `IconColorPicker`; wire into `AccountForm.tsx` and
   `categories.tsx`; switch the six read surfaces to the label components;
   add i18n keys.
5. `cd backend && go test ./...`; `cd frontend && pnpm lint && pnpm exec tsc
   && pnpm build`.

**Rollback:** the fields are additive and optional. Reverting the frontend
leaves harmless unused columns; the migration can be left in place (down is
just `DROP COLUMN` on four columns if truly needed).

## Open Questions

- Final curated icon list and the exact default per seeded category —
  settle during implementation, not blocking.
- Whether to include one neutral palette token from the start or wait for a
  concrete need. Leaning wait.
