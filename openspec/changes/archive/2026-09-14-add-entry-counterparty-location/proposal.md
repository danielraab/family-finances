## Why

Entries currently have no place to record *who the money moved with* (a
payer or payee — a shop, a person, an employer) or *where the transaction
happened*. Both are common on paper/bank statements and useful for
searching and recalling entries later, but neither fits an existing field:
`title`/`description` are freeform prose, not structured enough to
autocomplete against, and there is nowhere to put a coordinate at all.

## What Changes

- Add `counterparty` to entries: a nullable, free-text column, own column
  on `entries` (not folded into `description`), scoped to `kind =
  transaction` — a balance adjustment has no counterparty, mirroring how
  `category_id` is already required-for-transaction/optional-for-adjustment
  and `amount`/`balance` are already kind-exclusive. Suggested values come
  from a new read-only endpoint, `GET /api/entries/counterparties`,
  structurally identical to the existing `GET /api/account-types`
  (distinct, trimmed, case-insensitively sorted values from the caller's
  own entries). `counterparty` also joins the existing free-text `q` search
  (today `title OR description`) as a third `OR`'d column — no new filter
  parameter.
- Add `location` to entries: a nullable, free-text column, also `kind =
  transaction`-only, that the backend never interprets — same posture as
  `icon`/`color` on accounts/categories. The value is either a plain
  address string the user typed, or a JSON string `{"lat":…,"lng":…}`
  produced by the device's GPS or a pin dropped on a map; the frontend
  decides which it's looking at by attempting to parse it.
- Entry form (`/entries/new`, `/entries/{id}/edit`) gains a `counterparty`
  text input with a `<datalist>` (same UI shape as the account `type`
  field), and a `location` text input with two adjacent buttons: **Use
  GPS** (`navigator.geolocation.getCurrentPosition`, writes the JSON
  coordinate string directly into the field) and **Pick on map** (opens an
  interactive Leaflet/OpenStreetMap modal with a draggable pin, centered on
  the device's current position if available, else a world view; no
  address search; confirming writes the same JSON shape back into the
  field). The field stays a plain, directly-editable text input at all
  times — coordinates show as raw JSON when that's what's stored.
- The entry ledger (`/entries`) shows a globe icon on any row whose
  `location` parses as valid `{lat, lng}` JSON; clicking it opens a small,
  read-only modal with a Leaflet/OpenStreetMap view centered on that point
  with a static pin. A row whose `location` is a plain address string (or
  is empty) shows no globe icon.
- **New dependency**: Leaflet + OpenStreetMap tiles, the frontend's first
  external-network runtime dependency (every other asset — fonts, icons,
  API calls — is self-hosted or same-origin today) and its first
  non-self-hosted JS library. Called out explicitly rather than absorbed
  silently, since it's a precedent, not just an add.

## Capabilities

### Modified Capabilities

- `account-entries`: entries gain `counterparty` and `location`, both
  nullable and `kind = transaction`-only; `GET /api/entries/counterparties`
  is added; free-text search (`q`) additionally matches `counterparty`.
- `web-client-entries`: the create/edit entry form gains the counterparty
  and location inputs (with GPS capture and map-pin picking); the ledger
  gains a location globe icon with a read-only map-preview modal.

## Impact

- **Backend**: migration `0024_entry_counterparty_location.sql` adds two
  nullable `text` columns to `entries`; `backend/internal/entry/` gains the
  new fields (kind-gated validation mirroring `category_id`), a
  `ListInUseCounterparties` store method + `GET /api/entries/counterparties`
  handler mirroring `backend/internal/account`'s `ListInUseTypes`/`GET
  /api/account-types`, and the `q` search extended to include
  `counterparty`. `openapi/openapi.yaml` gains the new fields/endpoint,
  synced to `backend/openapi.yaml`.
- **Frontend**: new `Leaflet`/`react-leaflet` (or Leaflet used directly)
  dependency plus its CSS; new `LocationField` component (text input + GPS
  button + map-picker modal, used by both `entries.new.tsx` and
  `entries.$entryId.edit.tsx`); new read-only `LocationPreviewModal` used
  by the ledger; `counterparty` input added to both entry forms;
  `frontend/src/api/schema.d.ts` regenerated; i18n strings added.
- **No changes to existing tables or endpoints beyond the additive
  columns/field** — `balance_adjustment` entries and every existing
  `account-entries` behavior are unaffected.
