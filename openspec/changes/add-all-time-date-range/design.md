## Context

`frontend/src/lib/dateRangePresets.ts` owns the whole preset vocabulary:
`PresetKey`, the display-ordered `DATE_RANGE_PRESET_KEYS`, the i18n-suffix
map, `resolvePreset` (key + week start + today to a concrete
`{ from, to }`), `matchPreset` (the reverse lookup), and
`resolveEffectiveRange` (a route's raw `range`/`from`/`to` plus the page's
default to the range actually applied). `DateRangeFilter.tsx` renders it
and writes URL patches; three surfaces consume it —
`routes/entries.index.tsx` (default `last_2_weeks`),
`routes/reports.tsx` (no default) and
`components/dashboard/CardFormDialog.tsx` (no default, persisting the
choice into a card's config).

Two facts shape everything below.

First, a page's default is *not* written into the URL — it is a fallback
applied when `range`, `from` and `to` are all absent (see the archived
`add-date-range-presets` design, decision 2). So "no parameters" already
means "use the default", and an unbounded range that writes no parameters
is indistinguishable from it.

Second, the unbounded state is already implemented and already labelled.
`resolveEffectiveRange` returns `{ selectedKey: "custom", from: undefined,
to: undefined }` for it, both routes omit `from`/`to` from their queries
when undefined, and the trigger renders `dateRangeFilter.placeholder`,
which reads "All time" in English and "Gesamter Zeitraum" in German. This
change does not create a new filtering behaviour; it gives the existing one
a name in the dropdown and a key in the URL.

## Goals / Non-Goals

**Goals:**
- One click from any preset to an unbounded range, on every surface that
  embeds the filter.
- An unbounded range that survives a reload and a bookmark on a page whose
  default is narrower than everything.
- No new ambiguity: exactly one representation of "everything", displayed
  consistently wherever the effective range has no bounds.

**Non-Goals:**
- No backend or contract work — see proposal.md's Non-goals.
- No new "earliest entry" lookup. All time means "send no bounds", not
  "resolve to the date of the visitor's oldest entry". The backend already
  treats an absent bound as unbounded, and a resolved-to-real-dates variant
  would need a new endpoint and would go stale the moment an older entry
  is imported.
- No change to what `Custom` with both fields empty *does* — it stays a
  reachable, valid, unbounded state. It is simply no longer the only way
  to get there.

## Decisions

### 1. All time is a preset key, not a "clear both dates" action

`all_time` joins `PresetKey` and encodes as `?range=all_time`, the same as
`this_month` or `last_week`.

The alternative — a dropdown entry that just patches `from` and `to` to
`undefined` — was rejected because it cannot be represented. On `/entries`
that patch produces a URL with no date-range parameter at all, which
`resolveEffectiveRange` reads as "fall back to `last_2_weeks`". The
visitor picks "All time" and the ledger shows the last two weeks. Only a
positive key distinguishes an explicit choice from an absent one.

```
URL state                    effective range on /entries
─────────────────────────────────────────────────────────
(nothing)                 →  last_2_weeks   (page default)
?range=all_time           →  no bounds      (explicit)
?from=2026-01-01          →  custom, from only
```

### 2. `resolvePreset` returns nullable bounds

Today its signature promises two concrete strings and its `switch`
returns a pair in all ten arms. `all_time` returns
`{ from: undefined, to: undefined }`, so the return type becomes
`{ from: string | undefined; to: string | undefined }` — keys always
present rather than optional, since `exactOptionalPropertyTypes` is on and
the result is spread straight into an `EffectiveDateRange`.

The ripple is contained: `resolvePreset` is called only from `matchPreset`
and `resolveEffectiveRange`, both in the same file, and
`EffectiveDateRange` already declares `from`/`to` as `string | undefined`.
Every consumer downstream (`entries.index.tsx`, `reports.tsx`,
`dashboardFilter.ts`) already handles an undefined bound, because Custom
has always been able to produce one.

`matchPreset`'s scan keeps working unchanged: it compares
`resolved.from === from && resolved.to === to`, so a pair of undefined
bounds now matches `all_time` and nothing else — a Custom range with only
one bound set still has one defined side and cannot collide.
(`matchPreset` is currently exported but unreferenced; this change leaves
it correct rather than growing a second reverse lookup beside it.)

### 3. An absent default resolves to All time, not Custom

`resolveEffectiveRange`'s final branch — no `range`, no `from`/`to`, no
page default — returns `selectedKey: "custom"` today. It becomes
`selectedKey: "all_time"`, with both bounds still undefined.

This is a display change only. The applied range is identical (no bounds),
so `/reports` fetches exactly what it fetches today and a dashboard card
with no stored `range` renders exactly as before. What changes is that the
dropdown now names the state it is in, agreeing with a trigger that
already reads "All time".

Keeping `custom` here was the alternative: it would leave `/reports`
showing "Custom" with two empty fields while the trigger beside it says
"All time", which is the confusion this change exists to remove.

### 4. Clearing the last bound under Custom selects All time

In `changeFrom`/`changeTo`, when the filter is in Custom and the edit
leaves *both* bounds empty, the patch becomes
`{ range: "all_time", from: undefined, to: undefined }` instead of
`{ from: undefined }`.

Without this, Custom's empty state stays unrepresentable in the URL and
`/entries` keeps snapping back to "Last 2 weeks" when the visitor clears
the second date — the bug in proposal.md's Why. With it, there is one
representation of "everything" no matter which route the visitor took to
get there.

The visible consequence is that the two inputs disable themselves as the
last date leaves them, because a named preset is now active. That reads as
abrupt at the keystroke, but it is honest: nothing is being filtered, and
the state is now the same one the dropdown offers directly. Selecting
`Custom` again re-enables both fields, empty, since `selectPreset` carries
the effective bounds in and there are none.

### 5. All time sits last, before Custom

`DATE_RANGE_PRESET_KEYS` runs day-relative, then week-relative, then month,
then year — narrow to wide. `all_time` is the widest, so it goes after
`this_year`, immediately above the `Custom` entry that the component
renders separately.

Above `Today` was the alternative. It puts the least-used option in the
position the eye lands on first and breaks the widening order the list
already has.

### 6. One new i18n key, deliberately duplicating the placeholder

`dateRangeFilter.presets.allTime` is added with the same text as the
existing `dateRangeFilter.placeholder` ("All time" / "Gesamter Zeitraum").
Both name the same effective range — the preset label in the dropdown and
on the trigger, the placeholder on the trigger while Custom is open with
no bounds set — so showing identical text is correct, not a collision.
The placeholder stays: Custom with both fields empty remains reachable —
selecting `Custom` while All time is active lands exactly there (see
decision 7, which is what makes that state representable at all).

### 7. Selecting Custom with no bounds to carry writes `range=custom`

`selectPreset(CUSTOM_RANGE_KEY)` carries the outgoing preset's resolved
bounds into `from`/`to`. From `all_time` there are none to carry, so the
patch clears every date-range parameter — and a URL with no date-range
parameter is exactly how a page says "use my default". On `/entries` the
visitor would pick `Custom` and be thrown back to Last 2 weeks, with the
inputs disabled again. (Browser-verified before the fix: selecting Custom
from All time landed on `/entries` with `last_2_weeks` selected.)

So when `Custom` is selected with neither bound to carry, the filter
writes `range=custom`. `resolveEffectiveRange` treats *any* `range` value
that is not a known preset key as Custom with the given bounds, and — the
part that matters — stops short of the page default whenever any
date-range parameter is present at all.

```
URL state                    /entries dropdown + bounds
──────────────────────────────────────────────────────────
(nothing)                 →  Last 2 weeks, two dates
?range=all_time           →  All time, two empty disabled inputs
?range=custom             →  Custom, two empty editable inputs
?from=2026-01-01          →  Custom, from only
```

The marker never coexists with a bound: editing either date input under
Custom clears `range` in the same patch, so the two representations stay
mutually exclusive the moment there is anything to represent. This is the
same hole decision 4 closes from the other side — every route into "no
bounds" now has a URL that survives a page default.

## Risks / Trade-offs

- **An unbounded `/entries` query on a large history.** The ledger pages at
  30 rows and the list endpoint is already reachable unbounded from
  `/reports`, so this adds no new query shape — but it does make the
  unbounded shape one click away on the page with the most rows. Acceptable
  for a family-scale dataset; if it ever bites, it bites at the endpoint,
  not at the filter.
- **Two labels for one state.** Decision 6 accepts that "All time" can
  appear as either a selected preset or a Custom placeholder. The
  alternative — removing the empty-Custom state entirely — would mean
  forcing a bound whenever Custom is selected, which is a bigger and worse
  change.
- **The disabling-as-you-clear moment** in decision 4. Documented above;
  judged better than a silent snap-back to a range the visitor did not ask
  for.
- **Two URLs for one effective range.** `?range=all_time` and
  `?range=custom` both apply no date filter; they differ only in whether
  the inputs are editable. That is the point — the mode is part of the
  state the URL has to carry — but it does mean a card saved from the
  Custom-with-no-bounds state stores `preset: "custom"`, which resolves
  through the same unknown-key path and filters nothing, exactly as an
  empty stored range does today.

## Migration Plan

None. No stored data, no URL, and no card config changes meaning. Existing
`?range=<key>` and `?from`/`?to` links resolve exactly as before, a card
whose stored `range` is empty keeps resolving to no bounds, and a card
saved with `preset: "all_time"` resolves through the same path as every
other preset key.

## Open Questions

None outstanding. The three calls raised in exploration — whether a
no-default page shows All time, whether clearing the last bound snaps to
it, and where it sits in the list — are settled as decisions 3, 4 and 5.
