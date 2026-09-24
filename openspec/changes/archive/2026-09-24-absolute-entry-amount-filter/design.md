# Design

## Context

The initial amount-range feature passes signed stored-scale bounds from the
ledger directly into `entry.Filter`. Both storage implementations compare them
against each account-oriented ledger leg's signed `amount`; a self-transfer's
two visible legs therefore currently match differently. The route also owns
two duplicate input/draft-state blocks. See `proposal.md` and its delta specs.

## Goals / Non-Goals

**Goals:**

- Make a magnitude range yield identical inclusion decisions for positive and
  negative occurrences of the same amount.
- Keep full stored precision in the API and URL while accepting major-unit
  input in the ledger.
- Centralize the paired amount controls' positive-input behavior without
  changing the filter panel's shared layout.

**Non-Goals:**

- Change amount sort order or the signed amount displayed in ledger results.
- Add amount filtering to reports, summaries, dashboard cards, or recurring
  transaction queries.
- Add currency conversion or filter against a balance-adjustment's balance
  reading.

## Decisions

### Redefine the existing bounds as non-negative magnitudes

`amount_from` and `amount_to` remain the URL and API parameter names, but the
backend accepts only non-negative stored-scale values and applies them to each
ledger leg's magnitude. This is a deliberate breaking contract change because
the signed behavior has not reached production. Reusing the parameters keeps
shared URLs and persisted filter state concise and avoids a temporary
dual-semantics API.

Adding separate absolute-bound parameters while retaining signed bounds was
rejected: it would leave two competing range concepts in the same endpoint and
give the ledger no reason to expose signed filtering.

### Compare bounds without calculating an absolute `int64`

For an upper bound `to`, a leg qualifies when `-to <= amount <= to`. For a
positive lower bound `from`, it qualifies when `amount <= -from OR amount >=
from`; a zero lower bound imposes no lower predicate. These equivalent signed
comparisons avoid an `abs()` overflow for the minimum signed 64-bit amount,
while applying consistently to PostgreSQL and the in-memory implementation.

Using SQL `ABS(amount)` and Go `math.Abs` was rejected because their edge-case
behavior differs or overflows at the signed integer minimum. Filtering the
underlying entry instead of its account-oriented leg was also rejected: it
would make one side of a self-transfer use a value different from its rendered
amount.

### Extract a controlled AmountRangeFilter component

`AmountRangeFilter` will receive the optional stored-scale bounds and emit
their pair through one change callback. It will render the existing translated
start/end labels and filter-panel control styling, own its textual drafts, and
convert with the existing fixed-scale amount helpers. Its drafts synchronize
from supplied bounds so restored, cleared, and externally navigated URL state
is reflected in the controls.

On entry, the component removes a typed or pasted minus sign from the displayed
draft and serializes the magnitude. Empty or not-yet-parseable drafts remove
the corresponding bound; a parsed value is converted to its absolute
stored-scale integer before emitting. Keeping this at the component boundary
makes every rendered amount range positive without changing the general amount
parser used by signed entry and balance inputs.

Changing `inputToAmount` globally was rejected because transaction and balance
entry forms intentionally have distinct signed behavior.

### Retain range validation at the HTTP boundary

The handler will reject negative, malformed, and inverted ranges before they
reach service/storage code. The frontend's normalization makes normal UI
requests valid but does not protect manual API callers or handcrafted URLs.

## Risks / Trade-offs

- [Existing development-browser storage contains signed bounds] → The feature
  is unshipped; users can clear the range, and the backend rejects negative
  manually restored values rather than silently reinterpret them.
- [Negative pasted text may be briefly surprising] → The component immediately
  displays its positive magnitude, matching the field's documented semantics.
- [Magnitude comparisons remain nominal across currencies] → Preserve existing
  stored-scale behavior; the product has no exchange-rate model.

## Migration Plan

1. Update the OpenAPI description and regenerate both committed API artifacts.
2. Deploy backend and frontend together in the existing single-image release.
3. Roll back both packages together if necessary; no data migration is needed.
