## 1. Range draft model and validation

- [x] 1.1 Add focused date helpers that parse/format local-calendar ISO dates
      without UTC conversion, compare optional bounds, and expose the
      `from <= to` validity rule.
- [x] 1.2 Implement the pure Custom-selection state machine from design.md:
      first click starts, same/later click completes, an earlier second click
      restarts, and a click after completion begins a new selection.
- [x] 1.3 Represent named preset, Custom closed range, start-only, end-only,
      All time, selection phase, and an invalid externally supplied pair in one
      draft type derived from `resolveEffectiveRange`.
- [x] 1.4 Convert a valid draft back into one atomic `DateRangeValue` patch:
      named presets write `range`, Custom writes optional `from`/`to`,
      and each representation clears the other fields.

## 2. Accessible calendar view

- [x] 2.1 Add the latest available `@daypicker/react` version with pnpm and commit the
      lockfile change; verify its current React 19 and TypeScript API from the
      official documentation during implementation.
- [x] 2.2 Create the locally styled calendar component with German/English
      locale data, localized month/weekday labels, previous/next navigation,
      today/start/end/in-range states, and continuous highlighting across week
      and month boundaries.
- [x] 2.3 Wire pointer and keyboard selection through the same pure transition
      helper, preserving equal-bound ranges and preventing inverted output.
- [x] 2.4 Add accessible names and non-color indicators for navigation,
      selected endpoints, in-range days, and the current day.

## 3. Shared picker content and staged Apply

- [x] 3.1 Refactor `DateRangeFilter` so opening creates a local draft and
      closing without Apply discards it; Apply emits one `onChange` patch,
      closes, and restores focus.
- [x] 3.2 Replace the preset `select` with ordered quick actions generated
      from `DATE_RANGE_PRESET_KEYS`; selecting one updates the draft and
      calendar preview but does not call `onChange` before Apply.
- [x] 3.3 Add the Custom section, localized range summary, selection guidance,
      Reset/All-time behavior, "Use start only", and "Use end only" actions.
- [x] 3.4 Detect an externally supplied `from > to` pair, display the
      localized validation message, and disable Apply until the draft becomes
      valid. Do not rewrite the value merely by opening the picker.
- [x] 3.5 Keep trigger summaries localized while retaining ISO strings in URL
      and dashboard-card values.

## 4. Responsive presentation

- [x] 4.1 Render the shared picker content in a collision-aware anchored
      popover at `sm` and above, normally displaying two consecutive months
      and falling back to one when available width requires it.
- [x] 4.2 Render it below `sm` in a Headless UI Dialog bottom sheet with
      backdrop, close button, focus trap, one visible month, scrollable body,
      sticky Apply footer, and safe-area padding.
- [x] 4.3 Make mobile preset actions horizontally scrollable, keep every
      interactive target at least 44 CSS pixels, and verify the sheet remains
      usable in short/landscape viewports.
- [x] 4.4 Preserve the existing `fullWidth` trigger behavior and verify the
      filter still fits the entries grid, reports row, and dashboard form.

## 5. Translations

- [x] 5.1 Add English source strings for Apply, Reset, Custom, selection
      guidance, open-bound actions, invalid-order validation, month navigation,
      and screen-reader selection descriptions.
- [x] 5.2 Add complete German translations for the same keys and verify no
      user-facing calendar or responsive-shell text is hardcoded.

## 6. Verification

- [x] 6.1 From `frontend/`, run `pnpm lint`, `pnpm exec tsc --noEmit`,
      and `pnpm build`.
- [ ] 6.2 Browser-check desktop at representative widths: presets preview
      without applying, Apply writes one URL update, dismissal discards edits,
      two-month navigation works, ranges cross week/month boundaries, and the
      popover never overflows the viewport.
- [ ] 6.3 Browser-check mobile at 375px width and a short landscape viewport:
      one-month bottom sheet, focus trap/restoration, scrollable content,
      sticky safe-area-aware Apply, horizontal presets, and 44px targets.
- [ ] 6.4 Verify every allowed bound shape on `/entries` and `/reports`:
      both bounds, start only, end only, and All time; reload each resulting
      URL and confirm the same effective filter and summary return.
- [ ] 6.5 Verify ordering cases: end after start and equal dates apply; an
      earlier second click restarts selection; an injected `from > to` URL
      shows validation and cannot be applied unchanged.
- [ ] 6.6 Verify a dashboard `query_stat` or `entry_list` card can save and
      reopen every allowed bound shape, with no API or stored-config changes.
- [ ] 6.7 Check keyboard-only and screen-reader-oriented behavior in both
      shells: day traversal, range endpoints, month navigation, Apply disabled
      state, Escape/close behavior, and focus return.
