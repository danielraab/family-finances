# web-client-date-range-filter Specification

## Purpose

A shared date-range filter control used identically wherever a page filters
by a `from`/`to` date range (`/entries`, `/reports`). It offers a dropdown of
named presets plus a `Custom` mode, collapses into a single trigger
summarizing the current selection, encodes its state in the URL as either a
preset key or explicit bounds, and lets a consuming page supply its own
default effective range. See `web-client-entries` and `web-client-reports`
for how each page wires the control in, and `user-settings` for the
`week_start` preference that week-anchored presets use.

## Requirements

### Requirement: A shared date-range filter offers named presets and a Custom mode

The client SHALL offer one shared date-range filter control, used
identically wherever a page filters by a `from`/`to` date range, presenting
a dropdown of named presets — `Today`, `Last 7 days`, `Last 14 days`,
`Last 30 days`, `This week`, `Last week`, `Last 2 weeks`, `This month`,
`Last month`, `This year`, `All time` — plus a `Custom` entry. `All time`
SHALL be listed last among the named presets, immediately before `Custom`.
Selecting a named preset SHALL resolve it to a concrete `from`/`to` date
pair as of the current moment, except `All time`, which SHALL resolve to
neither bound; selecting `Custom` SHALL let the visitor set `from` and/or
`to` directly, each independently optional.

Preset date math SHALL be: `Today` is the current date for both bounds;
`Last 7/14/30 days` end on the current date and start 6/13/29 days earlier;
`This week` spans the configured week-start day through 6 days later;
`Last week` spans the 7 days immediately before the current week; `Last 2
weeks` is `Last week` and `This week` concatenated (14 days, starting 7 days
before the current week-start day and ending 6 days after it); `This month`
and `Last month` span the full current/previous calendar month; `This year`
spans the full current calendar year; `All time` resolves to no `from` and
no `to`, applying no date restriction at all. A preset whose resolved range
includes today or later (`This week`, `Last 2 weeks`, `This month`, `This
year`) SHALL be allowed to resolve an `end` date later than today — this is
expected, not an error condition, since a matching entry dated in the future
is valid.

#### Scenario: Selecting a preset resolves concrete dates

- **WHEN** an authenticated visitor selects "Last 30 days" from the filter
- **THEN** the filter's `from` resolves to 29 days before today and `to`
  resolves to today

#### Scenario: Selecting All time removes both bounds

- **WHEN** an authenticated visitor selects "All time" from the filter
- **THEN** the URL carries `range=all_time`, any previous `from`/`to`
  parameters are removed, and the page fetches its data with no date
  restriction

#### Scenario: All time survives a reload on a page with a narrower default

- **WHEN** an authenticated visitor opens a page whose default is "Last 2
  weeks" at a URL carrying `range=all_time`
- **THEN** "All time" is shown selected and no date filter is applied, the
  page's own default having no effect

#### Scenario: This week's range can include future dates

- **WHEN** an authenticated visitor selects "This week" on a day other than
  the last day of the configured week
- **THEN** the resolved `to` date is later than today

#### Scenario: Last 2 weeks spans last week and this week

- **WHEN** an authenticated visitor selects "Last 2 weeks"
- **THEN** the resolved range starts on the day the previous configured
  week began and ends on the last day of the current configured week

#### Scenario: Custom mode accepts either bound alone

- **WHEN** an authenticated visitor selects "Custom" and sets only a `from`
  date, leaving `to` empty
- **THEN** the filter applies an open-ended range starting at that date,
  the same as today's unfiltered `to` behavior

### Requirement: Filter state encodes as a preset key or explicit bounds, never both

The filter's state SHALL be represented as a URL search parameter `range`
holding the selected preset's key when a named preset is active, or as
`from`/`to` search parameters holding explicit date strings when `Custom` is
active. These two representations SHALL be mutually exclusive: applying a
named preset SHALL clear any existing `from`/`to` parameters, and setting a
`from` or `to` value directly SHALL clear any existing `range` parameter and
switch the filter to `Custom`. A `from`/`to` URL parameter pair that predates
this filter SHALL continue to be interpreted exactly as before (as an
explicit `Custom` range).

When `Custom` is selected while neither bound is set — and so there is
nothing for `from`/`to` to carry — the filter SHALL write `range=custom`
to mark the mode, since a URL carrying no date-range parameter at all is
instead read as "apply this page's default". Any `range` value that is not
a recognized preset key SHALL be interpreted as `Custom` with whatever
`from`/`to` are present, and the presence of any date-range parameter SHALL
suppress the page's default. Setting either bound SHALL clear that marker,
so a `range` value and explicit bounds never coexist.

#### Scenario: Applying a preset clears explicit bounds

- **WHEN** an authenticated visitor has `from`/`to` set via Custom and then
  selects a named preset
- **THEN** the URL's `from` and `to` parameters are removed and a `range`
  parameter with the preset's key is set

#### Scenario: Editing a date field clears the active preset

- **WHEN** an authenticated visitor has a named preset active and edits the
  `from` or `to` date input directly
- **THEN** the URL's `range` parameter is removed, the filter switches to
  Custom, and the edited `from`/`to` parameter is set

#### Scenario: Selecting Custom from All time marks the mode in the URL

- **WHEN** an authenticated visitor on a page defaulting to "Last 2 weeks"
  has All time active and selects "Custom"
- **THEN** the URL carries `range=custom` with no `from`/`to`, the filter
  shows Custom with two empty editable inputs, and the page's own default
  is not re-applied

#### Scenario: Setting a bound clears the Custom marker

- **WHEN** an authenticated visitor with `range=custom` in the URL sets a
  `from` date
- **THEN** the URL carries that `from` alone, with no `range` parameter

#### Scenario: A pre-existing from/to link still works

- **WHEN** an authenticated visitor opens a URL containing only `from` and
  `to` parameters from before this filter existed
- **THEN** the filter shows Custom selected with those exact bounds applied

### Requirement: Clearing the last remaining bound under Custom selects All time

The filter SHALL switch to the `All time` preset — writing `range=all_time`
into the URL and removing `from`/`to`, rather than leaving the URL with no
date-range parameter at all — when `Custom` is active and the visitor
clears a date input such that neither `from` nor `to` holds a value any
more. This keeps the unbounded range representable on a page whose default
is narrower, where an absent parameter would instead re-apply that default.

Clearing one of two set bounds SHALL NOT trigger this — the filter stays in
`Custom` with the remaining bound applied as an open-ended range.

#### Scenario: Clearing the second date switches to All time

- **WHEN** an authenticated visitor on a page defaulting to "Last 2 weeks"
  has Custom active with only a `from` date set, and clears it
- **THEN** the filter shows "All time" selected, the URL carries
  `range=all_time` with no `from`/`to`, and the list is not restricted by
  date

#### Scenario: Clearing one of two bounds stays in Custom

- **WHEN** an authenticated visitor has Custom active with both `from` and
  `to` set, and clears the `to` date
- **THEN** the filter stays in Custom with the `from` bound applied, and
  the URL carries `from` with no `range`

### Requirement: The filter collapses to a single trigger summarizing the current selection

The filter SHALL render as one compact trigger control, not as separate
always-visible fields, so it does not dominate a filter row of several
controls. The trigger SHALL show: the active preset's label, when a named
preset is active; the `from`–`to` range, when Custom is active with both
bounds set; a "from only" or "until only" label, when Custom is active with
exactly one bound set; or a placeholder text, when Custom is active with
neither bound set. Activating the trigger SHALL open a panel containing the
preset dropdown and the two date inputs, labeled From and To.

#### Scenario: The trigger shows the active preset's label

- **WHEN** an authenticated visitor has "Last month" active
- **THEN** the collapsed trigger reads "Last month", not raw dates

#### Scenario: The trigger shows a placeholder when nothing is set

- **WHEN** an authenticated visitor has Custom active with both From and To
  empty
- **THEN** the collapsed trigger shows placeholder text, not a blank
  control

### Requirement: The date inputs stay visible and reflect the resolved range under every mode

Within the opened panel, the filter SHALL always show two date inputs,
labeled From and To, regardless of whether a named preset or Custom is
active. When a named preset is active, both inputs SHALL display that
preset's resolved dates and SHALL be disabled — under `All time`, which
resolves to no bounds, both inputs SHALL be empty and disabled. When Custom
is active, both inputs SHALL be enabled and editable, reflecting (and
driving) the `from`/`to` values directly.

#### Scenario: A preset's resolved dates are visible

- **WHEN** an authenticated visitor opens the panel with "Last month"
  active
- **THEN** the From and To inputs display the first and last day of the
  previous calendar month, and neither input can be edited

#### Scenario: All time shows two empty, disabled inputs

- **WHEN** an authenticated visitor opens the panel with "All time" active
- **THEN** both the From and To inputs are empty and neither can be edited

#### Scenario: Custom inputs are editable

- **WHEN** an authenticated visitor opens the panel and selects "Custom"
- **THEN** the From and To inputs become editable

### Requirement: A consuming page supplies its own default effective range

A page embedding the filter SHALL supply its own default: either a preset
key to use as the effective range when neither `range` nor `from`/`to` is
present in the URL, or no default at all (no date filter applied). This
default SHALL NOT be written into the URL merely by being in effect — it
SHALL apply only to the effective range used for fetching data and to which
option the dropdown displays as selected on load. The dropdown's displayed
selection SHALL always be computed by matching the current effective
`from`/`to` against every preset's resolved bounds; when no preset matches,
`Custom` SHALL be shown. A page with no default SHALL therefore show
`All time` selected, that preset being the one whose resolved bounds match
the unrestricted range it applies. The first explicit change the visitor
makes SHALL write a concrete `range` or `from`/`to` value into the URL.

#### Scenario: A page with a default preset shows it selected without a URL parameter

- **WHEN** an authenticated visitor opens a page whose default is "Last 2
  weeks" with no `range`/`from`/`to` parameter in the URL
- **THEN** "Last 2 weeks" is shown selected, its resolved dates are shown in
  the disabled From/To inputs, and the URL is not modified

#### Scenario: A page with no default shows All time

- **WHEN** an authenticated visitor opens a page whose default is "no date
  filter" with no `range`/`from`/`to` parameter in the URL
- **THEN** "All time" is shown selected with both From and To inputs empty
  and disabled, no date filter is applied to the fetched data, and the URL
  is not modified

### Requirement: Week-anchored presets use the visitor's configured week start

`This week`, `Last week`, and `Last 2 weeks` SHALL compute their week
boundaries using the authenticated visitor's `week_start` setting (`monday`
or `sunday`). Every other preset SHALL be unaffected by this setting.

#### Scenario: A Sunday-start visitor's "This week" begins on Sunday

- **WHEN** an authenticated visitor whose `week_start` setting is `sunday`
  selects "This week"
- **THEN** the resolved range starts on the most recent Sunday on or before
  today and ends 6 days later

#### Scenario: A non-week preset ignores week start

- **WHEN** an authenticated visitor selects "This month" or "Last 30 days"
- **THEN** the resolved range is identical regardless of their `week_start`
  setting
