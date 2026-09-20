export type WeekStart = "monday" | "sunday";

/**
 * Named date-range presets, in display order. Keys are the literal `range`
 * URL value (see web-client-date-range-filter) — never renamed casually,
 * since a bookmarked/shared `?range=<key>` link depends on it.
 */
export type PresetKey =
  | "today"
  | "last_7_days"
  | "last_14_days"
  | "last_30_days"
  | "this_week"
  | "last_week"
  | "last_2_weeks"
  | "this_month"
  | "last_month"
  | "this_year"
  | "all_time";

export const DATE_RANGE_PRESET_KEYS: PresetKey[] = [
  "today",
  "last_7_days",
  "last_14_days",
  "last_30_days",
  "this_week",
  "last_week",
  "last_2_weeks",
  "this_month",
  "last_month",
  "this_year",
  "all_time",
];

/** Maps each preset key to the camelCase suffix of its
 * `dateRangeFilter.presets.*` i18n key. */
export const PRESET_I18N_KEYS: Record<PresetKey, string> = {
  today: "today",
  last_7_days: "last7Days",
  last_14_days: "last14Days",
  last_30_days: "last30Days",
  this_week: "thisWeek",
  last_week: "lastWeek",
  last_2_weeks: "last2Weeks",
  this_month: "thisMonth",
  last_month: "lastMonth",
  this_year: "thisYear",
  all_time: "allTime",
};

export const CUSTOM_RANGE_KEY = "custom";

/** The one preset resolving to neither bound — named because the filter
 * and `resolveEffectiveRange` both reach for it directly, as the key for
 * "no date restriction at all". */
export const ALL_TIME_KEY = "all_time" satisfies PresetKey;

export function isPresetKey(v: string | undefined): v is PresetKey {
  return v !== undefined && DATE_RANGE_PRESET_KEYS.includes(v as PresetKey);
}

function pad2(n: number): string {
  return String(n).padStart(2, "0");
}

/** Formats a Date's local year/month/day as `YYYY-MM-DD` — never uses UTC
 * getters, so the result matches what the visitor sees on their own
 * calendar regardless of timezone offset. */
function toDateString(d: Date): string {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
}

function addDays(d: Date, days: number): Date {
  const copy = new Date(d.getFullYear(), d.getMonth(), d.getDate());
  copy.setDate(copy.getDate() + days);
  return copy;
}

/** The most recent occurrence of the configured week-start day, on or
 * before `today`. */
function currentWeekStart(today: Date, weekStart: WeekStart): Date {
  const targetDow = weekStart === "monday" ? 1 : 0;
  const diff = (today.getDay() - targetDow + 7) % 7;
  return addDays(today, -diff);
}

/** Resolves a preset key to a concrete `{ from, to }` date-string pair, as
 * of `today`. See design.md's preset formula table. Both bounds are
 * nullable because `all_time` resolves to neither — every other preset
 * returns two concrete dates. The keys are always present (rather than
 * optional) so the result spreads straight into an `EffectiveDateRange`
 * under `exactOptionalPropertyTypes`. */
export function resolvePreset(
  key: PresetKey,
  weekStart: WeekStart,
  today: Date,
): { from: string | undefined; to: string | undefined } {
  switch (key) {
    case "today":
      return { from: toDateString(today), to: toDateString(today) };
    case "last_7_days":
      return {
        from: toDateString(addDays(today, -6)),
        to: toDateString(today),
      };
    case "last_14_days":
      return {
        from: toDateString(addDays(today, -13)),
        to: toDateString(today),
      };
    case "last_30_days":
      return {
        from: toDateString(addDays(today, -29)),
        to: toDateString(today),
      };
    case "this_week": {
      const start = currentWeekStart(today, weekStart);
      return {
        from: toDateString(start),
        to: toDateString(addDays(start, 6)),
      };
    }
    case "last_week": {
      const start = addDays(currentWeekStart(today, weekStart), -7);
      return {
        from: toDateString(start),
        to: toDateString(addDays(start, 6)),
      };
    }
    case "last_2_weeks": {
      const start = addDays(currentWeekStart(today, weekStart), -7);
      return {
        from: toDateString(start),
        to: toDateString(addDays(start, 13)),
      };
    }
    case "this_month": {
      const start = new Date(today.getFullYear(), today.getMonth(), 1);
      const end = new Date(today.getFullYear(), today.getMonth() + 1, 0);
      return { from: toDateString(start), to: toDateString(end) };
    }
    case "last_month": {
      const start = new Date(today.getFullYear(), today.getMonth() - 1, 1);
      const end = new Date(today.getFullYear(), today.getMonth(), 0);
      return { from: toDateString(start), to: toDateString(end) };
    }
    case "this_year": {
      const start = new Date(today.getFullYear(), 0, 1);
      const end = new Date(today.getFullYear(), 11, 31);
      return { from: toDateString(start), to: toDateString(end) };
    }
    case "all_time":
      return { from: undefined, to: undefined };
  }
}

/** Reverse-lookup: finds the preset (if any) whose resolved bounds exactly
 * match `from`/`to` — a pair of undefined bounds matches `all_time`, the
 * one preset that resolves to neither. Used only to decide the dropdown's
 * initial selection when a page's implicit default is in effect — an
 * explicit `from`/`to` URL pair is always shown as Custom regardless of
 * whether it happens to match a preset (see web-client-date-range-filter's
 * "Filter state encodes as a preset key or explicit bounds" requirement),
 * so this is never applied to URL-supplied bounds. */
export function matchPreset(
  from: string | undefined,
  to: string | undefined,
  weekStart: WeekStart,
  today: Date,
): PresetKey | typeof CUSTOM_RANGE_KEY {
  for (const key of DATE_RANGE_PRESET_KEYS) {
    const resolved = resolvePreset(key, weekStart, today);
    if (resolved.from === from && resolved.to === to) return key;
  }
  return CUSTOM_RANGE_KEY;
}

/** A route's current date-range URL state — `range` and `from`/`to` are
 * mutually exclusive (see web-client-date-range-filter). */
export type DateRangeValue = {
  range?: string | undefined;
  from?: string | undefined;
  to?: string | undefined;
};

export type EffectiveDateRange = {
  selectedKey: PresetKey | typeof CUSTOM_RANGE_KEY;
  from: string | undefined;
  to: string | undefined;
};

/** Resolves a route's raw `value` into the concrete range actually applied
 * and the preset the dropdown should show as selected. `defaultPreset` is
 * used only when `value` has neither `range` nor `from`/`to` at all; with
 * no default either, the range is unrestricted and the dropdown shows
 * `all_time`, the preset that names exactly that — see "A consuming page
 * supplies its own default effective range". */
export function resolveEffectiveRange(
  value: DateRangeValue,
  weekStart: WeekStart,
  defaultPreset: PresetKey | undefined,
  today: Date,
): EffectiveDateRange {
  if (isPresetKey(value.range)) {
    const resolved = resolvePreset(value.range, weekStart, today);
    return { selectedKey: value.range, ...resolved };
  }
  // Any date-range parameter at all — including a bare `range=custom` with
  // no bounds yet — means the visitor has chosen something, so the page's
  // default must not overrule it. Without this, selecting Custom with
  // nothing to carry in would clear every parameter and a page like
  // /entries would silently snap back to its own default.
  if (
    value.range !== undefined ||
    value.from !== undefined ||
    value.to !== undefined
  ) {
    return { selectedKey: CUSTOM_RANGE_KEY, from: value.from, to: value.to };
  }
  if (defaultPreset) {
    const resolved = resolvePreset(defaultPreset, weekStart, today);
    return { selectedKey: defaultPreset, ...resolved };
  }
  return { selectedKey: ALL_TIME_KEY, from: undefined, to: undefined };
}
