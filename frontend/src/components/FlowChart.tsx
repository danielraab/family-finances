import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { formatAmount } from "../lib/amount";
import { BarChart, type BarChartSeries } from "./charts/BarChart";

type FlowBucket = components["schemas"]["FlowBucket"];

// dataviz-skill categorical slots 6 (green) / 8 (red) — validated together
// for CVD/normal-vision separation; see frontend/AGENTS.md's chart
// convention. Same semantics as the app's existing positive/negative
// amount coloring, applied to chart fills instead of text.
const INCOME_FILL = "fill-[#008300] dark:fill-[#008300]";
const OUTCOME_FILL = "fill-[#e34948] dark:fill-[#e66767]";

/**
 * An income/outcome bar chart for one calendar year — two bars (income,
 * outcome) per month — fetched from `GET /api/entries/flow-summary`. Owns
 * its own year state and a previous/next-year pager.
 *
 * Not under `components/charts/`, which is presentational-only by
 * convention (nothing fetched inside): this composes `charts/BarChart`.
 *
 * - `accountIds` omitted → every account the caller owns (the endpoint's
 *   own default). Given → only those accounts.
 * - `currency` given → a single chart filtered to that currency (the
 *   account-details case). Omitted → one chart per currency present in the
 *   response, stacked, each headed by its code (the `/home` case).
 */
export function FlowChart({
  accountIds,
  currency,
  displayedDecimalPlaces,
}: {
  accountIds?: string[];
  currency?: string;
  displayedDecimalPlaces: number;
}) {
  const { t, i18n } = useTranslation();
  const [chartYear, setChartYear] = useState(() => new Date().getFullYear());
  const [buckets, setBuckets] = useState<FlowBucket[] | null>(null);

  // A stable stand-in for `accountIds` (a fresh array identity each render).
  const accountKey = (accountIds ?? []).join(",");

  // biome-ignore lint/correctness/useExhaustiveDependencies: accountKey is the stable dependency; accountIds itself is a new array each render.
  useEffect(() => {
    let cancelled = false;
    setBuckets(null);
    api
      .GET("/api/entries/flow-summary", {
        params: {
          query: {
            unit: "month",
            year: chartYear,
            ...(accountIds ? { account_id: accountIds } : {}),
          },
        },
      })
      .then(({ data }) => {
        if (!cancelled) setBuckets(data?.buckets ?? null);
      });
    return () => {
      cancelled = true;
    };
  }, [accountKey, chartYear]);

  const series: BarChartSeries[] = [
    { label: t("flowChart.income"), fillClassName: INCOME_FILL },
    { label: t("flowChart.outcome"), fillClassName: OUTCOME_FILL },
  ];

  const dataFor = (code: string) =>
    (buckets ?? []).map((bucket) => ({
      category: new Date(bucket.period).toLocaleDateString(
        i18n.resolvedLanguage,
        { month: "short" },
      ),
      values: [
        bucket.income.find((s) => s.currency === code)?.amount ?? 0,
        bucket.outcome.find((s) => s.currency === code)?.amount ?? 0,
      ],
    }));

  const formatFor = (code: string) => (v: number) =>
    formatAmount(
      v,
      code,
      displayedDecimalPlaces,
      i18n.resolvedLanguage ?? "en",
    );

  const currencies =
    currency !== undefined
      ? [currency]
      : [
          ...new Set(
            (buckets ?? []).flatMap((b) =>
              [...b.income, ...b.outcome].map((s) => s.currency),
            ),
          ),
        ].sort();

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-end gap-2 text-sm">
        <button
          type="button"
          onClick={() => setChartYear((y) => y - 1)}
          aria-label={t("flowChart.previousYear")}
          className="rounded-md px-2 py-1 font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
        >
          ◀
        </button>
        <span className="min-w-10 text-center font-medium tabular-nums">
          {chartYear}
        </span>
        <button
          type="button"
          onClick={() => setChartYear((y) => y + 1)}
          aria-label={t("flowChart.nextYear")}
          className="rounded-md px-2 py-1 font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
        >
          ▶
        </button>
      </div>

      {buckets !== null &&
        currencies.map((code) => (
          <div key={code} className="flex flex-col gap-2">
            {currency === undefined && (
              <h3 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
                {code}
              </h3>
            )}
            <BarChart
              series={series}
              data={dataFor(code)}
              formatValue={formatFor(code)}
            />
          </div>
        ))}
    </div>
  );
}
