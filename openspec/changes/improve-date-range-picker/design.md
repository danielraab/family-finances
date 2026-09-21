## Context

`DateRangeFilter.tsx` is shared by the entry ledger, reports, and dashboard
card configuration. It resolves the caller's raw `range`/`from`/`to`
value through `dateRangePresets.ts`, renders one trigger, and currently opens
a Headless UI popover containing a preset `select` plus two native
`input[type=date]` fields. Edits patch the caller immediately.

The data model already supports every required result:

| State | URL or card value |
|---|---|
| Named preset | `range=<preset key>` |
| Closed interval | `from=<date>&to=<date>` |
| Open end | `from=<date>` |
| Open start | `to=<date>` |
| Explicitly unbounded | `range=all_time` |

The change is therefore a frontend interaction redesign, not a new filtering
model. It must preserve those representations because bookmarked URLs,
page-specific defaults, and saved dashboard cards depend on them.

## Goals / Non-Goals

**Goals:**
- Select a closed interval in one calendar with a visible, continuous range.
- Make it impossible for the component to emit `from > to`.
- Keep open-start, open-end, and unbounded ranges explicit and convenient.
- Provide a compact desktop interaction and a touch-friendly mobile one with
  identical semantics.
- Apply a complete change atomically instead of exposing intermediate clicks to
  every consumer.

**Non-Goals:**
- No backend or OpenAPI work.
- No changes to preset calculations or URL parameter names.
- No general-purpose date/time framework.
- No silently swapping invalid values, which would change visitor input without
  explaining it.

## Decisions

### 1. One draft range, committed only by Apply

Opening the control creates a local draft from the effective range. Calendar
clicks, preset clicks, month navigation, Reset, and open-bound actions update
only that draft. Apply emits exactly one existing `DateRangeValue` patch and
closes the surface. Closing with Escape, the close button, or an outside click
discards the draft and restores focus to the trigger.

This replaces today's immediate URL updates. A two-click range selection would
otherwise expose a transient start-only filter after the first click, trigger
unnecessary data fetching, and make cancellation impossible. Atomic Apply also
lets both route-backed consumers and the dashboard form use precisely the same
component contract.

Preset actions still resolve through `resolvePreset`. Apply encodes a named
preset as `range=<key>`; a custom draft encodes explicit bounds and clears
`range`.

### 2. Calendar selection is a small state machine

The Custom draft has two optional bounds plus a selection phase:

- With no pending start, clicking a day sets `from` and waits for an end.
- Clicking the same or a later day completes `to`.
- Clicking an earlier day replaces `from` and continues waiting for an end.
  It never creates, swaps, or emits an inverted pair.
- Starting a new selection after a completed interval replaces the old range.
- "Use start only" keeps the selected start and clears the end.
- "Use end only" treats the currently selected day as `to` and clears
  `from`.
- All time clears both bounds and uses the existing `all_time` preset.

A pure helper owns these transitions and the `from <= to` predicate so visual
events, keyboard events, and Apply cannot drift into different rules.
ISO `YYYY-MM-DD` strings remain the comparison representation; their
lexicographic order is chronological and avoids timezone conversion.

### 3. Invalid external values are visible, not silently rewritten

A manually edited or legacy URL can still contain both bounds in reverse order.
The picker opens with those values represented in its summary, shows a localized
inline validation message, and disables Apply until the visitor selects a valid
range or an open-bound/all-time action. The existing URL is not rewritten merely
by opening the control.

This keeps the UI invariant strong without inventing an implicit correction
rule. All values created through presets or the calendar are valid by
construction.

### 4. Wide screens use a popover; narrow screens use a bottom sheet

At the `sm` breakpoint and above, the trigger opens an anchored popover. It
shows two consecutive months side by side when space permits, giving useful
context for ranges across a month boundary.

Below `sm`, the same trigger opens a Headless UI Dialog rendered as a
full-width bottom sheet with a backdrop, rounded top corners, a drag-handle
affordance, one visible month, and a sticky Apply footer respecting safe-area
insets. Month navigation uses buttons and may additionally support a horizontal
swipe, but swipe is an enhancement rather than the only navigation method.

The render shells differ; the draft state, calendar semantics, presets, and
Apply encoding are shared.

### 5. Presets become quick actions without changing their vocabulary

The long select becomes a compact grid on desktop and a horizontally scrollable
chip row on mobile. The order remains `DATE_RANGE_PRESET_KEYS`; no preset is
removed. Activating a preset selects it in the draft and shows its resolved
range in the calendar/summary. It does not apply until Apply.

Custom is represented by entering calendar selection rather than by a separate
select option. The panel labels the section "Custom" and automatically marks it
active after the visitor changes a calendar date or chooses an open-bound
action.

### 6. Use an accessible calendar primitive, styled locally

Add `@daypicker/react` for month grids, keyboard navigation, locale-aware
weekday/month labels, and range rendering. The application owns the state
machine and URL encoding; the dependency is only the accessible calendar view.
Its baseline structural stylesheet is imported, then its variables and
surrounding controls are adapted to the existing Tailwind visual language.

Building the entire calendar grid directly was rejected. Correct roving focus,
screen-reader labels, outside-month days, week boundaries, and range keyboard
selection are substantial accessibility work unrelated to the finance domain.

Implementation must verify the dependency's current documented React API when
the change is executed and pin the selected version through pnpm.

### 7. Display formatting is localized; stored values remain ISO

The trigger and draft summary use `Intl.DateTimeFormat` with the active i18n
language, producing familiar localized dates such as `21.09.2026` in German.
Values passed to callers remain local-calendar ISO strings exactly as today.
No `new Date("YYYY-MM-DD")` UTC parsing is used; helpers parse year, month, and
day into local calendar dates.

Summaries remain semantically distinct:

- closed: "12.09.2026 – 21.09.2026"
- open end: "From 12.09.2026"
- open start: "Until 21.09.2026"
- unbounded: "All time"
- preset: the preset's localized label

### 8. Accessibility and touch behavior are part of the contract

Every calendar day and control is keyboard reachable. The current day, selected
start, selected end, in-range days, and unavailable Apply action are conveyed
without relying on color alone. Month navigation has localized accessible
names. Focus enters the heading/first meaningful control, stays trapped in the
mobile dialog, and returns to the trigger on close.

Mobile interactive targets are at least 44 CSS pixels. The Apply footer remains
reachable above the device safe area, while calendar content may scroll within
the sheet.

## Risks / Trade-offs

- **New frontend dependency.** The accessible calendar avoids bespoke
  accessibility bugs but adds bundle and maintenance cost. Keep its usage behind
  one local component.
- **Apply changes today's immediate behavior.** Visitors gain cancelability and
  avoid intermediate fetches, but one extra confirmation is required. Presets
  and custom ranges use the same predictable rule.
- **Popover width.** Two months may not fit beside every desktop trigger.
  Collision-aware positioning and the one-month fallback prevent viewport
  overflow.
- **Mobile sheet height.** Small landscape screens may not fit all controls.
  Make the body scroll and keep only header/footer sticky.
- **No frontend test runner.** Pure helpers can be kept separately testable, but
  until a runner is introduced this change relies on typechecking, lint/build,
  and focused browser verification.

## Migration Plan

No migration. Existing URLs and saved dashboard configurations use unchanged
keys and bounds. The first time the redesigned control opens it derives a draft
from the same effective range used today.

## Open Questions

None. Exploration settled the interaction model: one calendar, explicit open
bounds, atomic Apply, two desktop months, and a one-month mobile bottom sheet.
