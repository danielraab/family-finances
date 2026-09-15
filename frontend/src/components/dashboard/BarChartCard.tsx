import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../../api/client";
import type { components } from "../../api/schema";
import { formatAmount } from "../../lib/amount";
import { compact } from "../../lib/compact";
import type { DashboardCardConfig } from "../../lib/dashboardFilter";
import { cardReferencesResolve } from "../../lib/dashboardFilter";
import {
  fetchRecurringPreview,
  type RecurringTransactionPreviewItem,
  resolvePreviewCutoff,
  todayDateString,
} from "../../lib/recurringPreview";
import type { Account } from "../../lib/useAccountsWithBalances";
import { useRecurringPreviewHorizon } from "../../lib/useRecurringPreviewHorizon";
import { BarChart, type BarChartSeries } from "../charts/BarChart";
import { CardTitleLink } from "./CardTitleLink";
import { MissingReferenceCard } from "./MissingReferenceCard";

type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];
type FlowBucket = components["schemas"]["FlowBucket"];

// dataviz-skill categorical slots 6 (green) / 8 (red), matching FlowChart's
// own income/outcome colouring.
const INCOME_FILL = "fill-[#008300] dark:fill-[#008300]";
const OUTCOME_FILL = "fill-[#e34948] dark:fill-[#e66767]";
// Muted variants (opacity-reduced) of the same hues for a stacked
// "projected" segment — same colour identity as the real segment, just
// visually receded. TODO: validate this pairing with the dataviz skill's
// scripts/validate_palette.js before shipping (see tasks.md 10.2).
const INCOME_PROJECTED_FILL = "fill-[#008300]/35 dark:fill-[#008300]/45";
const OUTCOME_PROJECTED_FILL = "fill-[#e34948]/35 dark:fill-[#e66767]/45";

/** Buckets preview items into the same period keys flow-summary's real
 * buckets use: a day bucket's key is the item's own date; a month
 * bucket's key is the first of that month — mirroring FlowBucket.period's
 * "first calendar day of the period" shape. */
function previewBucketKey(dateStr: string, unit: "month" | "day"): string {
  return unit === "day" ? dateStr : `${dateStr.slice(0, 7)}-01`;
}

/** Sums previewItems' amounts into { [bucketKey]: { [currency]: {income, outcome} } },
 * split by sign the same way flow-summary splits a real entry's amount.
 * Overdue items (booking_timestamp before today) are excluded — they'd
 * otherwise stack a projected amount onto an already-complete historical
 * bucket, which reads as wrong rather than useful. */
function bucketPreviewItems(
  items: RecurringTransactionPreviewItem[],
  unit: "month" | "day",
  today: string,
): Record<string, Record<string, { income: number; outcome: number }>> {
  const out: Record<
    string,
    Record<string, { income: number; outcome: number }>
  > = {};
  for (const item of items) {
    if (item.booking_timestamp < today) continue;
    const key = previewBucketKey(item.booking_timestamp, unit);
    const currency = item.account_currency ?? "";
    out[key] ??= {};
    out[key][currency] ??= { income: 0, outcome: 0 };
    if (item.amount > 0) {
      out[key][currency].income += item.amount;
    } else {
      out[key][currency].outcome += -item.amount;
    }
  }
  return out;
}

/**
 * A bar_chart card: an income/outcome bar chart via
 * GET /api/entries/flow-summary, scoped to config's account/category/tag
 * filter. unit=month renders a year of monthly bars with a prev/next-year
 * pager; unit=day renders a month of daily bars with a prev/next-month
 * pager. The displayed year/month defaults to the current one and is
 * never persisted — reopening the dashboard always starts there, same as
 * the previous fixed FlowChart did.
 */
export function BarChartCard({
  config,
  accounts,
  categories,
  tags,
  displayedDecimalPlaces,
  locale,
}: {
  config: DashboardCardConfig;
  accounts: Account[];
  categories: Category[];
  tags: Tag[];
  displayedDecimalPlaces: number;
  locale: string;
}) {
  const { t } = useTranslation();
  const today = new Date();
  const [year, setYear] = useState(today.getFullYear());
  const [month, setMonth] = useState(today.getMonth() + 1);
  const [buckets, setBuckets] = useState<FlowBucket[] | null>(null);
  const [previewItems, setPreviewItems] = useState<
    RecurringTransactionPreviewItem[]
  >([]);
  const recurringPreviewHorizon = useRecurringPreviewHorizon();

  const unit: "month" | "day" = config.unit === "day" ? "day" : "month";
  const resolves = cardReferencesResolve(config, accounts, categories, tags);
  const filterKey = JSON.stringify(config);

  // The preview cutoff never depends on which period is currently
  // displayed — a bar_chart card has no date-range filter to intersect
  // with (see design.md), only the horizon setting.
  const previewCutoff = config.show_recurring_preview
    ? resolvePreviewCutoff(undefined, recurringPreviewHorizon, today)
    : undefined;

  // biome-ignore lint/correctness/useExhaustiveDependencies: filterKey is config's stable stand-in; config itself is a new object identity each render.
  useEffect(() => {
    if (!resolves || !previewCutoff) {
      setPreviewItems([]);
      return;
    }
    let cancelled = false;
    fetchRecurringPreview({
      accountIds: config.account_id ? [config.account_id] : undefined,
      categoryId: config.category_id,
      categoryMode:
        config.category_id && config.include_subcategories === false
          ? "exact"
          : undefined,
      tagId: config.tag_id,
      to: previewCutoff,
    }).then(({ data }) => {
      if (!cancelled) setPreviewItems(data?.items ?? []);
    });
    return () => {
      cancelled = true;
    };
  }, [filterKey, previewCutoff, resolves]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: filterKey is config's stable stand-in; config itself is a new object identity each render.
  useEffect(() => {
    if (!resolves) return;
    let cancelled = false;
    setBuckets(null);
    api
      .GET("/api/entries/flow-summary", {
        params: {
          query: {
            unit,
            year,
            ...compact({
              month: unit === "day" ? month : undefined,
              account_id: config.account_id ? [config.account_id] : undefined,
              category_id: config.category_id,
              category_mode:
                config.category_id && config.include_subcategories === false
                  ? ("exact" as const)
                  : undefined,
              tag_id: config.tag_id,
            }),
          },
        },
      })
      .then(({ data }) => {
        if (!cancelled) setBuckets(data?.buckets ?? null);
      });
    return () => {
      cancelled = true;
    };
  }, [filterKey, unit, year, month, resolves]);

  if (!resolves) {
    return <MissingReferenceCard />;
  }

  const series: BarChartSeries[] = config.show_recurring_preview
    ? [
        {
          label: t("flowChart.income"),
          fillClassName: INCOME_FILL,
          projectedFillClassName: INCOME_PROJECTED_FILL,
          projectedLabel: t("dashboard.barChart.projectedIncome"),
        },
        {
          label: t("flowChart.outcome"),
          fillClassName: OUTCOME_FILL,
          projectedFillClassName: OUTCOME_PROJECTED_FILL,
          projectedLabel: t("dashboard.barChart.projectedOutcome"),
        },
      ]
    : [
        { label: t("flowChart.income"), fillClassName: INCOME_FILL },
        { label: t("flowChart.outcome"), fillClassName: OUTCOME_FILL },
      ];

  const previewByBucket = config.show_recurring_preview
    ? bucketPreviewItems(previewItems, unit, todayDateString(today))
    : {};

  const dataFor = (code: string) =>
    (buckets ?? []).map((bucket) => {
      const projected = previewByBucket[bucket.period]?.[code];
      return {
        category: new Date(bucket.period).toLocaleDateString(
          locale,
          unit === "day" ? { day: "numeric" } : { month: "short" },
        ),
        values: [
          bucket.income.find((s) => s.currency === code)?.amount ?? 0,
          bucket.outcome.find((s) => s.currency === code)?.amount ?? 0,
        ],
        ...(config.show_recurring_preview
          ? {
              projectedValues: [
                projected?.income ?? 0,
                projected?.outcome ?? 0,
              ],
            }
          : {}),
      };
    });

  const formatFor = (code: string) => (v: number) =>
    formatAmount(v, code, displayedDecimalPlaces, locale);

  const currencies = [
    ...new Set([
      ...(buckets ?? []).flatMap((b) =>
        [...b.income, ...b.outcome].map((s) => s.currency),
      ),
      // A currency with purely projected (no real) activity in the
      // displayed period still needs its own bar row.
      ...previewItems
        .filter((item) => item.account_currency)
        .map((item) => item.account_currency as string),
    ]),
  ].sort();

  function goBack() {
    if (unit === "day") {
      if (month === 1) {
        setYear((y) => y - 1);
        setMonth(12);
      } else {
        setMonth((m) => m - 1);
      }
    } else {
      setYear((y) => y - 1);
    }
  }

  function goForward() {
    if (unit === "day") {
      if (month === 12) {
        setYear((y) => y + 1);
        setMonth(1);
      } else {
        setMonth((m) => m + 1);
      }
    } else {
      setYear((y) => y + 1);
    }
  }

  const periodLabel =
    unit === "day"
      ? new Date(year, month - 1, 1).toLocaleDateString(locale, {
          month: "long",
          year: "numeric",
        })
      : String(year);

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-black/10 p-4 dark:border-white/10">
      <div className="flex items-center justify-between gap-2">
        <CardTitleLink
          config={config}
          accounts={accounts}
          categories={categories}
          tags={tags}
          allLabel={t("dashboard.filterAllAccounts")}
        />
        <div className="flex items-center gap-2 text-sm">
          <button
            type="button"
            onClick={goBack}
            aria-label={t("dashboard.barChart.previous")}
            className="rounded-md px-2 py-1 font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
          >
            ◀
          </button>
          <span className="min-w-16 text-center font-medium tabular-nums">
            {periodLabel}
          </span>
          <button
            type="button"
            onClick={goForward}
            aria-label={t("dashboard.barChart.next")}
            className="rounded-md px-2 py-1 font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
          >
            ▶
          </button>
        </div>
      </div>

      {buckets !== null &&
        currencies.map((code) => (
          <div key={code} className="flex flex-col gap-2">
            {currencies.length > 1 && (
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
