# Design

## Context

The entry ledger already maps URL search state into `GET /api/entries`, keeps that state in local storage, and resets cursor-paginated results whenever the request key changes. Backend listing uses one `entry.Filter` and two storage implementations; both list through account-oriented entry legs so a self-transfer can be returned with opposite signs. See `proposal.md` and the two delta specs for the requested behavior.

## Goals / Non-Goals

**Goals:**

- Preserve the existing fixed-four-decimal storage contract while presenting usable major-unit inputs in the ledger.
- Keep amount filtering consistent across Postgres, in-memory tests, sorting, and self-transfer leg orientation.
- Make copied/reloaded ledger URLs and filter restoration reproduce a selected amount range exactly.

**Non-Goals:**

- Extending amount bounds to entry summaries, reports, flow summaries, dashboard cards, or recurring-transaction queries.
- Filtering by absolute magnitude, currency conversion, or a balance adjustment's absolute balance reading.
- Adding a database index before a measured need; the filter composes with existing list predicates and its query plan should be evaluated against real production-like data first.

## Decisions

### Use `amount_from` and `amount_to` as signed stored-scale integers

The API carries entries' `amount` as an integer at a fixed scale, so the two new query parameters use that same representation and parse with signed 64-bit integer validation. Their names parallel the inclusive date bounds and make each optional endpoint clear.

The frontend will accept full-precision major-unit text using the existing amount conversion helper, then serialize the converted integer into the URL. This keeps API filtering exact and avoids locale-dependent number parsing in shareable URLs. Decimal strings in API query parameters were considered, but would create a second server-side decimal parser and risk disagreement with entry create/edit precision rules.

### Bound account-oriented entry legs rather than underlying entry rows

The Postgres store's `entry_legs` view and the memory store's generated legs already define what one row in the ledger means. Add lower/upper predicates to the shared list-filter path over each leg's signed amount. Therefore a self-transfer's sender and recipient rows are tested against their own signs, and an adjustment is tested against its computed delta. Filtering raw entry rows would make a recipient's visible positive leg obey the sender's negative amount, contradicting both the displayed ledger row and amount sorting.

### Validate inverted bounds at the HTTP boundary

After parsing both optional parameters, the handler rejects `amount_from > amount_to` as an invalid request. This makes the API behavior deterministic for all clients and prevents a UI or manually authored URL from silently producing an empty list. The client can avoid emitting an inverted range but does not replace this backend validation.

### Model the pair as one UI filter

`amount_from` and `amount_to` are two fields composing one numeric range. The ledger's active-filter badge increments once if either is present; clear all removes both; both fields participate in the request key and persisted search state. This matches the existing treatment of a date range's multiple URL keys.

## Risks / Trade-offs

- [A ledger spanning currencies compares raw nominal amounts rather than economic value] → This is consistent with existing amount sorting and is made explicit in the API's stored-scale contract; no conversion rate exists in the product.
- [A balance adjustment visually emphasizes its absolute reading while its filter uses the delta] → Describe the filter as signed amount bounds and cover the delta rule in contract tests.
- [Floating-point conversion in the browser] → Reuse the established full-precision conversion behavior used by entry forms and retain integer values in URLs/API requests.

## Migration Plan

The API change is additive: clients without amount parameters retain identical results. Deploy the backend and frontend together in the existing single-image release. Rollback is safe because the new frontend only sends optional query parameters; if rolling back the backend independently, deploy the prior frontend as well to avoid unsupported filters being ignored.
