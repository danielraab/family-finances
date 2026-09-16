import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../../api/client";
import type { components } from "../../api/schema";
import { formatAmount } from "../../lib/amount";
import { compact } from "../../lib/compact";
import type { DashboardCardConfig } from "../../lib/dashboardFilter";
import { cardReferencesResolve } from "../../lib/dashboardFilter";
import {
  cumulativePreviewDeltaByPeriod,
  fetchRecurringPreview,
  type RecurringTransactionPreviewItem,
  resolvePreviewCutoff,
  todayDateString,
} from "../../lib/recurringPreview";
import type { Account } from "../../lib/useAccountsWithBalances";
import { useRecurringPreviewHorizon } from "../../lib/useRecurringPreviewHorizon";
import { LineChart, type LineChartSeries } from "../charts/LineChart";
import { CardTitleLink } from "./CardTitleLink";
import { MissingReferenceCard } from "./MissingReferenceCard";

type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];
type BalancePoint = components["schemas"]["BalancePoint"];

// dataviz-skill categorical slot 1 (blue), matching the account-details
// page's own balance-chart colouring.
const BALANCE_STROKE = "stroke-[#2a78d6] dark:stroke-[#3987e5]";
const BALANCE_DOT = "fill-[#2a78d6] dark:fill-[#3987e5]";
// Muted variant (opacity-reduced) of the same hue for the projected
// balance line, mirroring accounts.$accountId.index.tsx's own constant.
const BALANCE_PROJECTED_STROKE = "stroke-[#2a78d6]/40 dark:stroke-[#3987e5]/50";

/**
 * A line_chart card: a running-balance line chart via
 * GET /api/entries/balance-series, scoped to config's optional single
 * account (omitted means every account the caller owns, summed per
 * currency). Only `unit=day` exists server-side, so — unlike BarChartCard
 * — there is no month/day toggle, just a previous/next-month pager. The
 * displayed month defaults to the current one and is never persisted —
 * reopening the dashboard always starts there, mirroring BarChartCard's
 * own year/month.
 */
export function LineChartCard({
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
  const [balancePoints, setBalancePoints] = useState<BalancePoint[] | null>(
    null,
  );
  const [previewItems, setPreviewItems] = useState<
    RecurringTransactionPreviewItem[]
  >([]);
  const recurringPreviewHorizon = useRecurringPreviewHorizon();

  const resolves = cardReferencesResolve(config, accounts, categories, tags);
  const filterKey = JSON.stringify(config);

  // biome-ignore lint/correctness/useExhaustiveDependencies: filterKey is config's stable stand-in; config itself is a new object identity each render.
  useEffect(() => {
    if (!resolves) return;
    let cancelled = false;
    setBalancePoints(null);
    api
      .GET("/api/entries/balance-series", {
        params: {
          query: {
            unit: "day",
            year,
            month,
            ...compact({
              account_id: config.account_id ? [config.account_id] : undefined,
            }),
          },
        },
      })
      .then(({ data }) => {
        if (!cancelled) setBalancePoints(data?.points ?? null);
      });
    return () => {
      cancelled = true;
    };
  }, [filterKey, year, month, resolves]);

  // The preview cutoff never depends on the displayed month — a line_chart
  // card has no date-range filter to intersect with, only the horizon
  // setting, mirroring BarChartCard's own cutoff resolution.
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
      to: previewCutoff,
    }).then(({ data }) => {
      if (!cancelled) setPreviewItems(data?.items ?? []);
    });
    return () => {
      cancelled = true;
    };
  }, [filterKey, previewCutoff, resolves]);

  if (!resolves) {
    return <MissingReferenceCard />;
  }

  const points = balancePoints ?? [];
  const todayStr = todayDateString(today);
  const projectedDeltaByPeriod =
    config.show_recurring_preview && previewCutoff
      ? cumulativePreviewDeltaByPeriod(
          previewItems,
          points.map((p) => p.period),
          todayStr,
          previewCutoff,
        )
      : {};

  const dataFor = (code: string) =>
    points.map((point, i) => {
      // The period is a local-day string ("YYYY-MM-DD"); read the day off
      // it directly rather than through a Date, which would shift under a
      // negative-offset browser timezone. The last point is the closing
      // boundary (the 1st of the next month) — label it with month + day
      // so it doesn't read as a duplicate "1".
      const day = Number(point.period.slice(8, 10));
      const isClosing = i === points.length - 1 && day === 1;
      const realValue =
        point.balances.find((b) => b.currency === code)?.amount ?? 0;
      const delta = projectedDeltaByPeriod[code]?.[i];
      return {
        category: isClosing
          ? new Date(`${point.period}T00:00:00`).toLocaleDateString(locale, {
              day: "numeric",
              month: "short",
            })
          : String(day),
        values: [realValue],
        ...(config.show_recurring_preview && delta !== undefined
          ? { projectedValues: [realValue + delta] }
          : {}),
      };
    });

  const formatFor = (code: string) => (v: number) =>
    formatAmount(v, code, displayedDecimalPlaces, locale);

  const currencies = [
    ...new Set(points.flatMap((p) => p.balances.map((b) => b.currency))),
  ].sort();

  function goBack() {
    if (month === 1) {
      setYear((y) => y - 1);
      setMonth(12);
    } else {
      setMonth((m) => m - 1);
    }
  }

  function goForward() {
    if (month === 12) {
      setYear((y) => y + 1);
      setMonth(1);
    } else {
      setMonth((m) => m + 1);
    }
  }

  const periodLabel = new Date(year, month - 1, 1).toLocaleDateString(locale, {
    month: "long",
    year: "numeric",
  });

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
            aria-label={t("dashboard.lineChart.previous")}
            className="rounded-md px-2 py-1 font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
          >
            ◀
          </button>
          <span className="min-w-32 text-center font-medium tabular-nums">
            {periodLabel}
          </span>
          <button
            type="button"
            onClick={goForward}
            aria-label={t("dashboard.lineChart.next")}
            className="rounded-md px-2 py-1 font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
          >
            ▶
          </button>
        </div>
      </div>

      {balancePoints !== null &&
        currencies.map((code) => {
          const series: LineChartSeries[] = [
            {
              label: t("accounts.details.balanceChart.series"),
              strokeClassName: BALANCE_STROKE,
              dotClassName: BALANCE_DOT,
              ...(config.show_recurring_preview
                ? {
                    projectedStrokeClassName: BALANCE_PROJECTED_STROKE,
                    projectedLabel: t("dashboard.lineChart.projectedBalance"),
                  }
                : {}),
            },
          ];
          return (
            <div key={code} className="flex flex-col gap-2">
              {currencies.length > 1 && (
                <h3 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
                  {code}
                </h3>
              )}
              <LineChart
                series={series}
                data={dataFor(code)}
                formatValue={formatFor(code)}
              />
            </div>
          );
        })}
    </div>
  );
}
