import { api } from "../api/client";
import type { components } from "../api/schema";
import { compact } from "./compact";

export type RecurringPreviewHorizon =
  components["schemas"]["RecurringPreviewHorizon"];
export type RecurringTransactionPreviewItem =
  components["schemas"]["RecurringTransactionPreviewItem"];

/** Every horizon preset, in display order — mirrors DATE_RANGE_PRESET_KEYS'
 * shape for a fixed, named-only (no free-form) enum. */
export const RECURRING_PREVIEW_HORIZON_KEYS: RecurringPreviewHorizon[] = [
  "1_month",
  "2_months",
  "3_months",
  "end_of_this_month",
  "end_of_next_month",
  "end_of_this_year",
];

/** Maps each horizon key to the camelCase suffix of its
 * `settings.profile.recurringPreviewHorizon.options.*` i18n key. */
export const RECURRING_PREVIEW_HORIZON_I18N_KEYS: Record<
  RecurringPreviewHorizon,
  string
> = {
  "1_month": "oneMonth",
  "2_months": "twoMonths",
  "3_months": "threeMonths",
  end_of_this_month: "endOfThisMonth",
  end_of_next_month: "endOfNextMonth",
  end_of_this_year: "endOfThisYear",
};

/** Mirrors the backend's hardcoded default (see internal/settings). */
export const DEFAULT_RECURRING_PREVIEW_HORIZON: RecurringPreviewHorizon =
  "end_of_this_month";

function pad2(n: number): string {
  return String(n).padStart(2, "0");
}

/** Formats a Date's local year/month/day as `YYYY-MM-DD` — same convention
 * as dateRangePresets.ts's own toDateString. */
function toDateString(d: Date): string {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
}

/** `today`'s own local calendar date as `YYYY-MM-DD` — for comparing
 * against a resolved cutoff to decide whether an Upcoming block has
 * anything left to show at all (see design.md: a cutoff already before
 * today means the block should not render). */
export function todayDateString(today: Date): string {
  return toDateString(today);
}

/** Resolves a recurring-preview horizon to a concrete calendar date
 * (`YYYY-MM-DD`), as of `today` — the client-side half of the
 * recurring-transaction-preview design's cutoff formula. The backend never
 * interprets a horizon key itself, only the resolved date this produces. */
export function resolveRecurringPreviewHorizon(
  horizon: RecurringPreviewHorizon,
  today: Date,
): string {
  switch (horizon) {
    case "1_month":
      return toDateString(
        new Date(today.getFullYear(), today.getMonth() + 1, today.getDate()),
      );
    case "2_months":
      return toDateString(
        new Date(today.getFullYear(), today.getMonth() + 2, today.getDate()),
      );
    case "3_months":
      return toDateString(
        new Date(today.getFullYear(), today.getMonth() + 3, today.getDate()),
      );
    case "end_of_this_month":
      return toDateString(
        new Date(today.getFullYear(), today.getMonth() + 1, 0),
      );
    case "end_of_next_month":
      return toDateString(
        new Date(today.getFullYear(), today.getMonth() + 2, 0),
      );
    case "end_of_this_year":
      return toDateString(new Date(today.getFullYear(), 11, 31));
  }
}

/** Resolves the effective preview cutoff for a surface: the earlier of the
 * surface's own resolved date-range `to` (when it has one — bar_chart cards
 * never do) and the horizon setting resolved to a date. See
 * design.md's cutoff-formula decision. */
export function resolvePreviewCutoff(
  filterTo: string | undefined,
  horizon: RecurringPreviewHorizon,
  today: Date,
): string {
  const horizonDate = resolveRecurringPreviewHorizon(horizon, today);
  if (filterTo && filterTo < horizonDate) return filterTo;
  return horizonDate;
}

function toRangeEnd(date: string): string {
  return new Date(`${date}T23:59:59.999Z`).toISOString();
}

export type PreviewFilterInput = {
  accountIds?: string[] | undefined;
  categoryId?: string | undefined;
  categoryMode?: "exact" | "subtree" | undefined;
  tagId?: string | undefined;
  /** The resolved cutoff date (`YYYY-MM-DD`), from resolvePreviewCutoff. */
  to: string;
};

/** Fetches GET /api/recurring-transactions/preview for the given filter —
 * the one place every consuming surface (`/entries`, `/reports`, the
 * dashboard's entry_list/bar_chart cards) builds this request, so the
 * cutoff-to-RFC3339 conversion and filter shape live in exactly one place. */
export function fetchRecurringPreview(input: PreviewFilterInput) {
  return api.GET("/api/recurring-transactions/preview", {
    params: {
      query: {
        ...compact({
          account_id:
            input.accountIds && input.accountIds.length > 0
              ? input.accountIds
              : undefined,
          category_id: input.categoryId,
          category_mode: input.categoryMode,
          tag_id: input.tagId,
        }),
        to: toRangeEnd(input.to),
      },
    },
  });
}

/** A stable React key for a virtual preview row — never a real entity id,
 * since a projected occurrence is never persisted. The account is part of
 * the key because a self-transfer projects one row per account it touches,
 * and those rows share both the template id and the date. */
export function previewItemKey(item: RecurringTransactionPreviewItem): string {
  return `${item.recurring_transaction_id}-${item.account_id}-${item.booking_timestamp}`;
}

/**
 * Turns a flat preview item list into a per-currency cumulative delta
 * aligned to a balance chart's own day points (e.g. BalancePoint.period),
 * for overlaying a projected balance line on top of a real one (LineChart's
 * projected-segment capability). `periods` must be ascending "YYYY-MM-DD"
 * strings.
 *
 * Index i of a currency's array is `undefined` when periods[i] is before
 * today (nothing to project — the real and projected value are the same,
 * and the chart should draw no dashed segment there) or after `cutoff`
 * (the fetch that produced `items` was itself bounded by that same cutoff,
 * so nothing beyond it is known — holding the last cumulative value flat
 * forever would misrepresent "no further data" as "no further activity");
 * otherwise it's the sum of that currency's non-overdue items whose
 * booking_timestamp falls in the half-open interval [today, periods[i]) —
 * mirroring GET /api/entries/balance-series' own point semantics, where an
 * entry booked exactly on a point's day only moves the *next* point, not
 * the one that opens it. At periods[i] === today, that interval is empty,
 * so the result is always 0 there — the anchor a projected line's dashed
 * segment continues from, by construction rather than a rendering special
 * case. Overdue items (booking_timestamp < today) are excluded entirely,
 * mirroring BarChartCard.tsx's bucketPreviewItems.
 */
export function cumulativePreviewDeltaByPeriod(
  items: RecurringTransactionPreviewItem[],
  periods: string[],
  today: string,
  cutoff: string,
): Record<string, (number | undefined)[]> {
  const currencies = new Set(
    items
      .filter((item) => item.booking_timestamp >= today)
      .map((item) => item.account_currency ?? ""),
  );

  const result: Record<string, (number | undefined)[]> = {};
  for (const currency of currencies) {
    let cumulative = 0;
    let itemIndex = 0;
    const sorted = items
      .filter(
        (item) =>
          (item.account_currency ?? "") === currency &&
          item.booking_timestamp >= today,
      )
      .sort((a, b) => a.booking_timestamp.localeCompare(b.booking_timestamp));

    result[currency] = periods.map((period) => {
      if (period < today || period > cutoff) return undefined;
      while (
        itemIndex < sorted.length &&
        (sorted[itemIndex] as RecurringTransactionPreviewItem)
          .booking_timestamp < period
      ) {
        cumulative += (sorted[itemIndex] as RecurringTransactionPreviewItem)
          .amount;
        itemIndex++;
      }
      return cumulative;
    });
  }
  return result;
}
