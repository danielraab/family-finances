## MODIFIED Requirements

### Requirement: A shared date-range filter offers named presets and a Custom mode

The client SHALL offer one shared date-range filter control wherever a page or
dashboard form filters by optional `from`/`to` bounds. It SHALL present the
existing named presets as quick actions and SHALL provide a Custom section in
which both endpoints are selected from one calendar rather than two independent
native date inputs. Preset vocabulary, resolution, and URL encoding SHALL remain
unchanged.

Selecting a preset SHALL update a local draft and preview its effective range.
Selecting a calendar day SHALL switch the draft to Custom. Neither interaction
SHALL change the consuming page or form until Apply is activated.

#### Scenario: A preset previews before it applies

- **WHEN** a visitor opens the filter and selects "Last 30 days"
- **THEN** the picker previews that preset's resolved range while the currently
  applied page filter remains unchanged
- **AND WHEN** the visitor activates Apply
- **THEN** the filter writes `range=last_30_days` and closes

#### Scenario: Custom uses one calendar

- **WHEN** a visitor chooses a start and end for a Custom range
- **THEN** both endpoints and every included date are shown in one connected
  calendar selection rather than separate date inputs

### Requirement: The filter collapses to a single trigger summarizing the current selection

The filter SHALL render as one compact trigger summarizing the applied
selection. A named preset SHALL use its translated label. A Custom closed range
SHALL show both dates in the visitor's locale; a start-only range SHALL read
"from" that localized date; an end-only range SHALL read "until" that localized
date; and an unbounded range SHALL use the All time label. The underlying URL
values SHALL remain ISO `YYYY-MM-DD` strings.

Activating the trigger SHALL open the responsive picker. Closing the picker
without Apply SHALL discard its draft and leave the trigger, URL, and applied
filter unchanged.

#### Scenario: A German Custom summary uses local display formatting

- **WHEN** the applied Custom bounds are `2026-09-12` and `2026-09-21`
  and the interface language is German
- **THEN** the trigger summarizes them as
  `12.09.2026 – 21.09.2026`
- **AND** the URL retains the original ISO values

#### Scenario: Dismissing a draft makes no change

- **WHEN** a visitor changes dates inside the open picker and closes it without
  Apply
- **THEN** the prior applied summary and filter remain in effect

### Requirement: The calendar reflects the effective range under every mode

Within the opened panel, the filter SHALL show a calendar and a range summary
for every mode. Named presets SHALL display their resolved dates without making
those dates directly editable as Custom bounds. All time SHALL display no
selected dates. Entering Custom calendar selection SHALL make the chosen dates
the draft's explicit optional bounds.

A wide presentation SHOULD show two consecutive months when they fit. A narrow
presentation SHALL show one month at a time. Month navigation SHALL not alter
the selected draft.

#### Scenario: A preset range is visible in the calendar

- **WHEN** the visitor opens the picker with "Last month" applied
- **THEN** its first and last day and the dates between them are visibly
  represented in the calendar

#### Scenario: All time has no selected calendar dates

- **WHEN** the visitor opens the picker with All time applied
- **THEN** no calendar date is selected and the All time preset is active

## ADDED Requirements

### Requirement: Calendar selection never applies an inverted range

The component SHALL never emit a Custom value whose defined `from` is later
than its defined `to`. With no pending start, the first calendar activation
SHALL select the start. Activating the same or a later date SHALL complete the
range. Activating an earlier date while waiting for the end SHALL replace the
start and continue waiting rather than creating or swapping an invalid pair.
Equal start and end dates SHALL be valid.

A manually constructed input value whose `from` is later than `to` SHALL
show localized validation and SHALL keep Apply disabled until the draft is
valid. Merely opening the picker SHALL NOT rewrite that external value.

#### Scenario: A later date completes the range

- **WHEN** the visitor selects 12 September and then 21 September
- **THEN** Custom contains `from=2026-09-12` and `to=2026-09-21`
- **AND** every date in that inclusive interval is highlighted

#### Scenario: An earlier second date restarts selection

- **WHEN** the pending start is 21 September and the visitor next activates
  12 September
- **THEN** 12 September becomes the new pending start
- **AND** the component does not create `from=2026-09-21&to=2026-09-12`

#### Scenario: The same date is a valid closed range

- **WHEN** the visitor activates the same calendar date as both endpoints
- **THEN** Apply is enabled and both explicit bounds use that date

#### Scenario: Invalid URL bounds cannot be reapplied unchanged

- **WHEN** the filter receives `from=2026-09-21&to=2026-09-12`
- **THEN** it explains that the end cannot be before the start
- **AND** Apply remains disabled until the visitor chooses a valid or
  open-ended range

### Requirement: Custom selection preserves all open-bound forms

The Custom picker SHALL allow a visitor to apply only a start, only an end, or
both endpoints. "Use start only" SHALL keep the selected start and omit
`to`. "Use end only" SHALL use the selected day as `to` and omit `from`.
All time SHALL remain the representation with neither bound.

#### Scenario: Apply an open end

- **WHEN** the visitor selects 12 September and activates "Use start only"
- **THEN** Apply emits `from=2026-09-12` with no `to` or `range`

#### Scenario: Apply an open start

- **WHEN** the visitor selects 21 September and activates "Use end only"
- **THEN** Apply emits `to=2026-09-21` with no `from` or `range`

#### Scenario: Select neither bound

- **WHEN** the visitor activates All time and then Apply
- **THEN** Apply emits `range=all_time` with no explicit bounds

### Requirement: The picker adapts to desktop and mobile interaction

At the `sm` breakpoint and above, the picker SHALL open as a
collision-aware anchored popover and SHOULD show two consecutive months where
the viewport permits. Below `sm`, it SHALL open as a modal bottom sheet with
a backdrop, one visible month, a close control, a scrollable body, and a sticky
Apply footer that respects the device safe area. Both presentations SHALL share
the same draft and selection behavior.

Mobile interactive targets SHALL be at least 44 CSS pixels. Keyboard focus
SHALL remain within the modal sheet, Escape SHALL dismiss without applying, and
focus SHALL return to the trigger after either dismissal or Apply. Calendar
navigation and selection SHALL be operable by keyboard and exposed with
localized accessible names.

#### Scenario: A narrow viewport uses the bottom sheet

- **WHEN** the viewport is narrower than `sm` and the visitor activates the
  trigger
- **THEN** a one-month modal bottom sheet opens with a sticky Apply action
- **AND** the background is inert until the sheet closes

#### Scenario: A wide viewport uses the anchored picker

- **WHEN** the viewport is at least `sm` and has room for two months
- **THEN** the picker opens anchored to its trigger and displays two consecutive
  month grids without overflowing the viewport

#### Scenario: Mobile dismissal restores focus

- **WHEN** a keyboard visitor closes the mobile sheet without Apply
- **THEN** the draft is discarded and focus returns to the date-filter trigger

## RENAMED Requirements

- FROM: `### Requirement: The date inputs stay visible and reflect the resolved range under every mode`
- TO: `### Requirement: The calendar reflects the effective range under every mode`
