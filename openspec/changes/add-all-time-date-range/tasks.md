## 1. The `all_time` preset key

- [ ] 1.1 In `frontend/src/lib/dateRangePresets.ts`, add `"all_time"` to
      the `PresetKey` union and to `DATE_RANGE_PRESET_KEYS`, last —
      after `this_year`, so it renders immediately above the separately
      rendered `Custom` entry (design.md decision 5).
- [ ] 1.2 Add `all_time: "allTime"` to `PRESET_I18N_KEYS`.
- [ ] 1.3 Widen `resolvePreset`'s return type to
      `{ from?: string; to?: string }` and add its `case "all_time"`
      returning `{ from: undefined, to: undefined }` (design.md decision
      2). Leave the other ten arms as they are.
- [ ] 1.4 Confirm `matchPreset` still type-checks and now returns
      `"all_time"` for a pair of undefined bounds — no code change
      expected, just the comparison holding with optional bounds.

## 2. An absent default resolves to All time

- [ ] 2.1 In `resolveEffectiveRange`, change the final branch (no
      `range`, no `from`/`to`, no page default) to return
      `{ selectedKey: "all_time", from: undefined, to: undefined }`
      instead of `CUSTOM_RANGE_KEY` (design.md decision 3), and update
      the function's doc comment accordingly.
- [ ] 2.2 Check the three consumers that pass no default —
      `routes/reports.tsx` (both `resolveEffectiveRange` calls),
      `lib/dashboardFilter.ts` (`resolveCardDateRange`,
      `resolveCardRangeToDateString`) and
      `components/dashboard/CardFormDialog.tsx` — and confirm none of
      them branches on `selectedKey`, so the applied range is byte-for-byte
      what it was.

## 3. Clearing the last bound under Custom

- [ ] 3.1 In `frontend/src/components/DateRangeFilter.tsx`, make
      `changeFrom` and `changeTo` emit
      `{ range: "all_time", from: undefined, to: undefined }` when the
      filter is in Custom and the edit leaves both bounds empty
      (design.md decision 4). Clearing one of two set bounds keeps
      today's behaviour.
- [ ] 3.2 Leave `selectPreset`'s Custom branch as it is — carrying the
      effective bounds in still does the right thing from All time,
      landing on Custom with two empty editable fields.

## 4. Translations

- [ ] 4.1 Add `dateRangeFilter.presets.allTime` to
      `frontend/src/i18n/locales/en.json` as "All time" and to `de.json`
      as "Gesamter Zeitraum" — the same strings the existing
      `dateRangeFilter.placeholder` already carries (design.md decision
      6). Keep `placeholder` in both files; Custom with no bounds is
      still reachable.

## 5. Verification

- [ ] 5.1 `cd frontend && pnpm lint && pnpm exec tsc --noEmit && pnpm
      build` — the typecheck is the one that matters here, since
      widening `resolvePreset`'s return type is the change most likely
      to ripple, and CI's `frontend` job does not run `tsc`.
- [ ] 5.2 Drive the filter in a browser against a stubbed `/api` and
      check, on `/entries` (default "Last 2 weeks"): selecting "All time"
      writes `range=all_time` and lists entries older than two weeks;
      reloading that URL keeps All time selected; the two date inputs are
      empty and disabled; selecting Custom from there gives two empty
      editable fields; and clearing the last date under Custom lands back
      on All time rather than on Last 2 weeks.
- [ ] 5.3 On `/reports` with a bare URL, check the dropdown reads "All
      time" rather than "Custom", and that generating a report returns
      the same unrestricted results as before.
- [ ] 5.4 In the dashboard's card form, check a `query_stat` or
      `entry_list` card saved with All time selected renders unrestricted,
      and that an existing card with no stored range is unaffected.
