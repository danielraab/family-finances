## Context

`backend/internal/entry`'s `Entry`/`New`/`Update` types already have a
working precedent for a field that only applies to one `Kind`:
`CategoryID` is required for `transaction`, optional for
`balance_adjustment`, enforced in `validateNew`
(`backend/internal/entry/entry.go:370-381`). `counterparty` and `location`
follow that exact shape — both are `transaction`-only, rejected (or simply
never offered) on a `balance_adjustment`, for the same reason `category_id`
is: a balance adjustment is an absolute reading, not a real-world event
with a counterparty or a place.

Two existing patterns elsewhere in the codebase settle most of the design
by direct precedent rather than new invention:

- **`accounts.type`** (`backend/AGENTS.md`'s "Account types" section): a
  plain, backend-uninterpreted `text` column, with a read-only
  `GET /api/account-types` distinct-values endpoint feeding a frontend
  `<datalist>`. `counterparty` ports this exactly.
- **`icon`/`color`** on accounts/categories: opaque tokens the backend
  never interprets or validates beyond shape — "the meaning lives entirely
  in the frontend." `location` ports this same posture, except with no
  shape validation at all (not even icon/color's regex), since its two
  legitimate shapes (free text, or JSON coordinates) are frontend
  concerns, not backend ones.

## Goals / Non-Goals

**Goals:**
- Let a transaction record a free-text counterparty, suggested from the
  caller's own history, searchable via the existing `q` filter.
- Let a transaction record a location as either a typed address or GPS
  coordinates (device GPS or a dropped map pin), with no backend
  interpretation of which.
- Show, at a glance in the ledger, which entries carry a real coordinate
  (vs. a plain address or nothing), and let a viewer see it on a map with
  one click.

**Non-Goals:**
- No structured address fields (street/city/postcode) — `location` is one
  opaque string, exactly as specified.
- No reverse geocoding (turning coordinates into a place name) or forward
  geocoding/address search in the map picker — explicitly out of scope for
  this change.
- No two-party (`from` + `to`) modeling — `counterparty` is a single field,
  meaning "the other party," regardless of transaction direction.
- No counterparty or location on `balance_adjustment` entries.
- No validation, structured storage, or querying of `location` beyond
  "does it parse as `{lat, lng}` JSON" — no radius search, no map-based
  filtering of the ledger.

## Decisions

### `counterparty` is one field, not `from`/`to`

A single nullable string, meaning "the other party in the transaction" —
who was paid, or who paid — regardless of whether the amount is negative
or positive. This mirrors how a bank statement's payee/payer line usually
works: one slot, interpreted by the amount's sign. Two separate `from`/
`to` fields were considered and rejected during exploration — they raise
unanswered questions (is `to` required when `from` is set? can both apply
to one transaction?) that a single field sidesteps entirely, at no loss of
the requested capability.

### `location` is one opaque text column; the frontend decides what it means

The backend stores `location` as a plain nullable `text` column with no
validation beyond nothing (not even a trim requirement) — it is never
parsed, never format-checked, exactly like `icon`/`color`. The frontend
alone decides how to render it:

```
location value                        rendering
─────────────────────────────────────────────────────────────────────
null / ""                             nothing
parses as JSON, shape {lat, lng},     🌐 globe icon; click → read-only
  both numbers in valid range           map modal, static pin at (lat,lng)
anything else                         shown as plain text, no globe
```

This was chosen over a structured `latitude`/`longitude` column pair
(the initial design-phase idea) once the requirement clarified that a
plain typed address must also be storable in the *same* field — a
structured pair can't hold free text, and adding a second column
(`location_text`) for the address case would mean two fields where the
requirement asked for one. One opaque text column, sniffed on read, is the
only shape that satisfies "one field, three ways to fill it."

### The field always shows its raw stored value, including raw JSON

When GPS or the map picker sets a coordinate, the text input displays the
literal JSON string (e.g. `{"lat":48.2082,"lng":16.3738}`) rather than a
friendlier formatted string backed by a hidden value. This was a explicit
choice over a nicer display: the field is a single source of truth with no
parallel representation to keep in sync, at the cost of the JSON being
visible (and, if hand-edited, potentially broken — an invalid edit simply
falls through to "shown as plain text, no globe," never a hard error,
since the backend doesn't validate this field at all).

### Leaflet + OpenStreetMap tiles, accepted as a deliberate precedent

This frontend currently makes zero third-party network calls (self-hosted
fonts, no CDN scripts, same-origin API only — see `frontend/AGENTS.md`).
Rendering an actual map requires real tile imagery, which means an
external request to a tile provider; there is no self-hosted alternative
in scope for this change. This is accepted deliberately, not by default —
call it out in review rather than treat it as an ordinary dependency bump.
Leaflet is the library (not a heavier alternative like Mapbox GL) because
it's the smallest widely-used option that supports both an interactive,
draggable-pin picker and a simple static preview from the same API.

### Two separate map surfaces, not one shared component

- **Picker** (entry form, "Pick on map"): interactive — a draggable
  marker, click-to-place, a Confirm button that writes `{lat,lng}` back
  into the `location` field. Centers on the device's current GPS position
  if geolocation succeeds, otherwise a world view. No address search box.
- **Preview** (ledger's globe icon): read-only — a static marker at the
  entry's stored coordinate, no interaction, no Confirm button, opened by
  clicking the globe icon.

These are two components (`LocationPickerModal`, `LocationPreviewModal`),
not one generalized over an `editable` prop — the interaction models
(drag/click/confirm vs. nothing) are different enough that a shared
component would carry unused branches in the read-only case, the same
reasoning `design.md` in `add-tag-sharing` used to keep `TagLabel` distinct
from `CategoryLabel`.

### Ledger placement: counterparty rides under the title, the globe icon sits beside it

The ledger table already has six columns (Date, Account, Title, Category,
Tags, Amount) inside an `overflow-x-auto` wrapper. Rather than widen it
with a seventh/eighth column for counterparty and the globe icon, both
render inside the existing Title cell:

- `counterparty`, when present, renders as a second line under the title
  link — the same slot/style `created_by_name` already uses for "entry
  wasn't created by me."
- The globe icon renders inline, immediately after the title text (small,
  clickable, like a badge) — opening `LocationPreviewModal` on click.

This keeps the table shape unchanged and reuses an established pattern
(a secondary line under the title) instead of inventing a new column that
would mostly be empty.

### Update semantics: plain nullable strings, no `OptionalID` trick needed

`Entry.Update.Description` is already a plain `*string` (nil = untouched,
a non-nil pointer — including `*""`— sets/clears it), because there's no
third meaning to disambiguate. `counterparty` and `location` follow the
same shape, unlike `category_id`'s `OptionalID` (which exists specifically
to distinguish "leave alone" from "explicitly clear," relevant only
because category has permission/validity implications on the *new* value).
Neither `counterparty` nor `location` has that complication — validating
only applies at all when `Kind == transaction`, same as `Amount`.

## Risks / Trade-offs

- **[Risk]** A user could type non-JSON text that still happens to look
  like coordinates, or leave stray whitespace around valid JSON, and be
  confused why no globe icon appears → **Mitigation**: the parse-attempt
  is exact (`JSON.parse` after trim, checked for a `{lat, lng}` shape with
  both values in valid ranges) and consistent between the ledger and the
  form's own live-globe-preview (see tasks) — the same value always
  produces the same globe/no-globe outcome everywhere it's shown.
- **[Risk]** Leaflet + OSM tiles is a new external-network dependency in
  a codebase that otherwise makes none → **Mitigation**: named explicitly
  in the proposal and here, not absorbed silently; OSM's tile usage policy
  permits this scale of use; no self-hosted tile alternative is attempted
  in this change since it's substantially heavier (mirror-hosting map
  tiles) and out of proportion to the feature.
- **[Risk]** `navigator.geolocation` requires a secure context and
  explicit user permission, and can fail (denied, timeout, no GPS
  hardware) → **Mitigation**: "Use GPS" degrades to a visible inline error
  on failure, never a silent no-op; "Pick on map" remains available as a
  no-GPS-required fallback, and typing an address manually always works.

## Migration Plan

1. Additive migration `0024_entry_counterparty_location.sql`: add nullable
   `counterparty text` and `location text` columns to `entries`. No
   backfill, no constraint beyond nullability — both columns default to
   `NULL` for every existing row.
2. Backend: extend `entry.go` (`New`/`Update`/`Entry` fields, kind-gated
   validation), `service.go`, `store.go` (`ListInUseCounterparties`), both
   `storage/memory` and `storage/postgres` implementations, `handler.go`
   (new field wiring + `GET /api/entries/counterparties`), search query
   extension in `storage/postgres/entry.go`'s `Query` handling. Update
   `openapi/openapi.yaml`, regenerate `backend/openapi.yaml` and
   `frontend/src/api/schema.d.ts`.
3. Frontend: add the Leaflet dependency; build `LocationField` (text input
   + GPS button + Pick-on-map button), `LocationPickerModal`,
   `LocationPreviewModal`; wire `counterparty` + `location` into
   `entries.new.tsx` and `entries.$entryId.edit.tsx`; add the globe icon +
   counterparty second-line to `entries.index.tsx`'s title cell.
4. No data backfill needed — every existing entry simply has `counterparty
   = NULL, location = NULL` until edited.
5. Rollback: dropping the two columns and reverting the code is sufficient
   at any point pre-release; nothing else depends on their existence.

## Open Questions

None outstanding — field cardinality (one column, not two), kind-scoping,
suggestion-endpoint shape, search integration, location's opaque-JSON
storage, raw-JSON display, the map library choice, and ledger placement
were all resolved during exploration.
