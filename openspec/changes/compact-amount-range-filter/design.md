# Design

## Context

The current `AmountRangeFilter` keeps both labels visible in the filter grid
and uses `amountToInput`, which intentionally emits all four stored decimal
places for edit forms. The date range already uses a compact trigger and an
overlay for its related controls.

## Goals / Non-Goals

**Goals:**
- Treat the two amount bounds as one compact filter-panel control.
- Preserve exact stored-scale values while removing visually redundant zeroes.

**Non-Goals:**
- Change amount parsing, URL serialization, or filtering semantics.
- Localize numeric separators differently from existing amount inputs.

## Decisions

### Use a Popover trigger and panel

`AmountRangeFilter` will use the project's Headless UI popover pattern: a
compact field-styled trigger summarizes the active bounds, and its anchored
panel contains both labels and inputs. This keeps the pair together without
requiring a second filter-panel row.

### Trim only display trailing zeroes

A component-local formatter will derive major units from stored scale, render
up to four decimal places, then trim terminal zeroes and a now-empty decimal
point. Parsing and emitted URL values continue to use `inputToAmount`, so no
precision is lost.

## Risks / Trade-offs

- [A compact summary is less explicit when no bounds are set] → Use the
  translated amount-range placeholder on the trigger.
