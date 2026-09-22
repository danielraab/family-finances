## 1. The `all_time` preset key

- [x] 1.1 In `frontend/src/lib/dateRangePresets.ts`, add `"all_time"` to
      the `PresetKey` union and to `DATE_RANGE_PRESET_KEYS`, last —
      after `this_year`, so it renders immediately above the separately
      rendered `Custom` entry (design.md decision 5).
- [x] 1.2 Add `all_time: "allTime"` to `PRESET_I18N_KEYS`.
- [x] 1.3 Widen `resolvePreset`'s return type to
      `{ from: string | undefined; to: string | undefined }` (keys
      present, values nullable — `exactOptionalPropertyTypes` is on and
      the result spreads into an `EffectiveDateRange`) and add its
      `case "all_time"`
      returning `{ from: undefined, to: undefined }` (design.md decision
      2). Leave the other ten arms as they are.
- [x] 1.4 Confirm `matchPreset` still type-checks and now returns
      `"all_time"` for a pair of undefined bounds — no code change
      expected, just the comparison holding with optional bounds.

## 2. An absent default resolves to All time

- [x] 2.1 In `resolveEffectiveRange`, change the final branch (no
      `range`, no `from`/`to`, no page default) to return
      `{ selectedKey: "all_time", from: undefined, to: undefined }`
      instead of `CUSTOM_RANGE_KEY` (design.md decision 3), and update
      the function's doc comment accordingly.
- [x] 2.2 Check the three consumers that pass no default —
      `routes/reports.tsx` (both `resolveEffectiveRange` calls),
      `lib/dashboardFilter.ts` (`resolveCardDateRange`,
      `resolveCardRangeToDateString`) and
      `components/dashboard/CardFormDialog.tsx` — and confirm none of
      them branches on `selectedKey`, so the applied range is byte-for-byte
      what it was.

## 3. Clearing the last bound under Custom

- [x] 3.1 In `frontend/src/components/DateRangeFilter.tsx`, make
      `changeFrom` and `changeTo` emit
      `{ range: "all_time", from: undefined, to: undefined }` when the
      filter is in Custom and the edit leaves both bounds empty
      (design.md decision 4). Clearing one of two set bounds keeps
      today's behaviour.
- [x] 3.2 Selecting Custom from All time also has to survive a page
      default — see section 4, which is what actually makes it land on
      Custom with two empty editable fields.

## 4. Selecting Custom with nothing to carry in

Found while verifying task 3 in a browser, not anticipated by the original
proposal — see design.md decision 7.

- [x] 4.1 In `resolveEffectiveRange`, suppress the page default whenever
      any date-range parameter is present, so a `range` value that is not
      a known preset key resolves to Custom with the given bounds rather
      than falling through to the default.
- [x] 4.2 In `selectPreset`, write `range: "custom"` when Custom is
      selected with no bounds to carry in, and keep today's
      bounds-carrying behaviour when there are some.
- [x] 4.3 In `changeFrom`/`changeTo`, clear `range` when a bound is set
      under Custom, so the marker never coexists with explicit bounds.

## 5. Translations

- [x] 5.1 Add `dateRangeFilter.presets.allTime` to
      `frontend/src/i18n/locales/en.json` as "All time" and to `de.json`
      as "Gesamter Zeitraum" — the same strings the existing
      `dateRangeFilter.placeholder` already carries (design.md decision
      6). Keep `placeholder` in both files; Custom with no bounds is
      still reachable.

## 6. Verification

- [x] 6.1 `cd frontend && pnpm lint && pnpm exec tsc --noEmit && pnpm
      build` — the typecheck is the one that matters here, since
      widening `resolvePreset`'s return type is the change most likely
      to ripple, and CI's `frontend` job does not run `tsc`.
- [x] 6.2 Drive the filter in a browser against a stubbed `/api` and
      check, on `/entries` (default "Last 2 weeks"): selecting "All time"
      writes `range=all_time` and lists entries older than two weeks;
      reloading that URL keeps All time selected; the two date inputs are
      empty and disabled; selecting Custom from there gives two empty
      editable fields; and clearing the last date under Custom lands back
      on All time rather than on Last 2 weeks.
- [x] 6.3 On `/reports` with a bare URL, check the dropdown reads "All
      time" rather than "Custom", and that generating a report returns
      the same unrestricted results as before.
- [x] 6.4 In the dashboard's card form, check a `query_stat` or
      `entry_list` card saved with All time selected renders unrestricted,
      and that an existing card with no stored range is unaffected.
