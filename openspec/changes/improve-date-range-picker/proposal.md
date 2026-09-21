## Why

The shared date-range filter still exposes Custom mode as two independent
native date inputs. That makes the relationship between start and end hard to
see, offers no visual overview of the chosen interval, and allows the visitor
to create a range whose start is later than its end. The current compact
popover also does not adapt its interaction model to a narrow mobile viewport.

The filter should make the valid path the easiest path: presets remain quick,
a custom interval is selected in one calendar, and the component never emits
an inverted range. Open-ended ranges are legitimate and must remain explicit,
easy choices rather than accidental empty fields.

## What Changes

- Replace the preset select and two native date inputs with a purpose-built
  range-picker panel: quick preset actions plus one calendar for Custom mode.
- Select both endpoints in the same calendar. The first date starts a range;
  the second date completes it only when it is on or after the start. Choosing
  an earlier second date starts a new range instead of producing an invalid
  pair. Equal start and end remains valid.
- Keep unbounded filtering first-class through explicit "Use start only",
  "Use end only", and "All time" actions. Either bound or both bounds may
  therefore remain absent.
- Stage edits inside the open picker and write the URL/card-form value only
  when the visitor activates Apply. Closing the panel discards the draft, so
  consumers never observe a half-completed or inverted pair.
- Present the control as an anchored popover with two visible months on wide
  screens and as a full-width bottom-sheet dialog with one visible month on
  narrow screens. Both forms share the same selection semantics and state.
- Improve summaries and accessibility: localized display dates, connected
  range highlighting, keyboard navigation, focus restoration, minimum mobile
  touch targets, and an accessible description of open-ended selections.

## Non-goals

- No backend filtering, database, OpenAPI, preset date-math, or URL schema
  changes. Applied values remain `range=<preset>` or explicit `from`/`to`.
- No removal or renaming of existing presets, including All time and Custom.
- No global date-picker abstraction for unrelated forms.
- No change to page-specific default ranges.
- No attempt to repair a manually constructed invalid URL silently. The picker
  will surface it as invalid and require a valid selection before Apply, while
  all states produced by the UI are valid by construction.

## Capabilities

### Modified Capabilities

- `web-client-date-range-filter`: Custom mode becomes a single calendar range
  picker with staged application, explicit open-bound actions, a non-inverted
  range invariant, localized summaries, and responsive desktop/mobile
  presentations.

## Impact

- `frontend/src/components/DateRangeFilter.tsx` — panel state, preset actions,
  responsive presentation, Apply/Reset controls, and integration with the new
  calendar component.
- New focused frontend components/helpers for calendar rendering, range draft
  transitions, localized display formatting, and validation.
- `frontend/src/i18n/locales/{en,de}.json` — labels and accessible
  descriptions for Apply, Reset, open-bound actions, month navigation, and
  selection guidance.
- `frontend/package.json` and `pnpm-lock.yaml` if the implementation adopts
  the accessible calendar dependency selected in design.md.
- No backend, API contract, generated API type, or stored-data changes.
