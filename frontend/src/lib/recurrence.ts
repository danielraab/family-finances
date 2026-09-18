import type { components } from "../api/schema";

type IntervalUnit = components["schemas"]["IntervalUnit"];

/**
 * A curated set of common recurrence shortcuts, each mapping to the
 * underlying {interval_unit, interval_count} pair the API actually stores —
 * see backend/openspec design.md's "interval unit + count, not named
 * frequencies" decision. Purely a frontend convenience: picking a preset
 * just fills in the two raw fields; there is no `frequency` concept on the
 * wire. "Custom" isn't a real preset — it's the fallback UI state that
 * reveals the raw interval_unit/interval_count inputs directly.
 */
export const RECURRENCE_PRESETS: {
  key: string;
  unit: IntervalUnit;
  count: number;
}[] = [
  { key: "weekly", unit: "week", count: 1 },
  { key: "everyTwoWeeks", unit: "week", count: 2 },
  { key: "monthly", unit: "month", count: 1 },
  { key: "everyTwoMonths", unit: "month", count: 2 },
  { key: "quarterly", unit: "month", count: 3 },
  { key: "everySixMonths", unit: "month", count: 6 },
  { key: "yearly", unit: "year", count: 1 },
];

export const CUSTOM_PRESET_KEY = "custom";

/** Finds the preset key matching unit/count exactly, or "custom" otherwise. */
export function matchPreset(unit: IntervalUnit, count: number): string {
  return (
    RECURRENCE_PRESETS.find((p) => p.unit === unit && p.count === count)?.key ??
    CUSTOM_PRESET_KEY
  );
}

/**
 * A recurring transaction's monthly share of its annualized amount —
 * `per_year_amount` divided by twelve, at the same fixed 4-decimal-place
 * integer scale.
 *
 * Derived on the client rather than sent by the backend: `per_year_amount`
 * is itself computed on every read and never stored, and this is that same
 * number divided by a constant. Rounding is half away from zero, mirroring
 * the backend's own `PerYearAmount`, so a negative (expense) amount rounds
 * symmetrically to a positive one. At the stored scale that rounding is
 * accurate to 0.0001 of a currency unit — two orders of magnitude below
 * the display default, so it can never move a rendered digit.
 */
export function perMonthAmount(perYearAmount: number): number {
  const monthly = perYearAmount / 12;
  const rounded = Math.sign(monthly) * Math.round(Math.abs(monthly));
  // `-0` would otherwise reach Intl and render as "-€0.00" for a
  // per-year amount small enough to round away.
  return rounded === 0 ? 0 : rounded;
}
