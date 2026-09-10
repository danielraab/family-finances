import { createFileRoute, Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { AccountLabel } from "../components/AccountLabel";
import { useAuth } from "../components/AuthProvider";
import {
  LineChart,
  type LineChartSeries,
} from "../components/charts/LineChart";
import { FlowChart } from "../components/FlowChart";
import {
  amountColorClass,
  formatAmount,
  formatSignedAmount,
} from "../lib/amount";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";

export const Route = createFileRoute("/accounts/$accountId/")({
  component: AccountDetails,
});

type Account = components["schemas"]["Account"];
type Entry = components["schemas"]["Entry"];
type BalancePoint = components["schemas"]["BalancePoint"];

// Only one chart is on screen at a time; the switcher below picks which.
type ChartView = "flow" | "balance";

const RECENT_LIMIT = 5;

// dataviz-skill categorical slot 1 (blue) — validated for the lightness
// band, chroma floor, and >=3:1 contrast against both surfaces. A single
// series needs no CVD-pair check.
const BALANCE_STROKE = "stroke-[#2a78d6] dark:stroke-[#3987e5]";
const BALANCE_DOT = "fill-[#2a78d6] dark:fill-[#3987e5]";

function AccountDetails() {
  const { accountId } = Route.useParams();
  const { t, i18n } = useTranslation();
  const { user } = useAuth();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();
  const [account, setAccount] = useState<Account | null | undefined>(undefined);
  const [balance, setBalance] = useState<number | null>(null);
  const [recent, setRecent] = useState<Entry[] | null>(null);
  const [chartView, setChartView] = useState<ChartView>("flow");
  // First of the month the balance line is showing — a single Date so
  // previous/next roll the year over for free.
  const [balanceMonth, setBalanceMonth] = useState(() => {
    const now = new Date();
    return new Date(now.getFullYear(), now.getMonth(), 1);
  });
  const [balancePoints, setBalancePoints] = useState<BalancePoint[] | null>(
    null,
  );

  useEffect(() => {
    let cancelled = false;
    api
      .GET("/api/accounts/{id}", { params: { path: { id: accountId } } })
      .then(({ data }) => {
        if (!cancelled) setAccount(data ?? null);
      });
    api
      .GET("/api/accounts/{id}/balance", {
        params: { path: { id: accountId } },
      })
      .then(({ data }) => {
        if (!cancelled) setBalance(data?.balance ?? null);
      });
    api
      .GET("/api/entries", {
        params: {
          query: {
            account_id: [accountId],
            sort: "booking_timestamp",
            dir: "desc",
            limit: RECENT_LIMIT,
          },
        },
      })
      .then(({ data }) => {
        if (!cancelled) setRecent(data?.items ?? []);
      });
    return () => {
      cancelled = true;
    };
  }, [accountId]);

  // Each chart fetches only while it's the one on screen (mounted by the
  // switcher below), so opening the page loads a single series, not both.
  useEffect(() => {
    if (chartView !== "balance") return;
    let cancelled = false;
    setBalancePoints(null);
    api
      .GET("/api/entries/balance-series", {
        params: {
          query: {
            account_id: [accountId],
            unit: "day",
            year: balanceMonth.getFullYear(),
            month: balanceMonth.getMonth() + 1,
          },
        },
      })
      .then(({ data }) => {
        if (!cancelled) setBalancePoints(data?.points ?? null);
      });
    return () => {
      cancelled = true;
    };
  }, [accountId, balanceMonth, chartView]);

  if (account === undefined) {
    return null;
  }
  if (account === null) {
    return (
      <section className="mx-auto flex w-full max-w-3xl flex-col gap-4 px-6 py-12 sm:px-10">
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("accounts.notFound")}
        </p>
      </section>
    );
  }

  const balanceSeries: LineChartSeries[] = [
    {
      label: t("accounts.details.balanceChart.series"),
      strokeClassName: BALANCE_STROKE,
      dotClassName: BALANCE_DOT,
    },
  ];
  const points = balancePoints ?? [];
  const balanceData = points.map((point, i) => {
    // The period is a local-day string ("YYYY-MM-DD"); read the day off it
    // directly rather than through a Date, which would shift under a
    // negative-offset browser timezone. The last point is the closing
    // boundary (the 1st of the next month) — label it with month + day so
    // it doesn't read as a duplicate "1".
    const day = Number(point.period.slice(8, 10));
    const isClosing = i === points.length - 1 && day === 1;
    return {
      category: isClosing
        ? new Date(`${point.period}T00:00:00`).toLocaleDateString(
            i18n.resolvedLanguage,
            { day: "numeric", month: "short" },
          )
        : String(day),
      values: [
        point.balances.find((b) => b.currency === account.currency)?.amount ??
          0,
      ],
    };
  });

  return (
    <section className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-6 py-12 sm:px-10">
      <header className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-semibold tracking-tight">
            <AccountLabel account={account} iconSize={26} />
          </h1>
          {account.description && (
            <p className="text-sm text-zinc-500 dark:text-zinc-400">
              {account.description}
            </p>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Link
            to="/accounts/$accountId/sharing"
            params={{ accountId }}
            className="rounded-md border border-black/15 px-3 py-2 text-sm font-medium transition-colors hover:bg-black/[.04] dark:border-white/15 dark:hover:bg-white/[.06]"
          >
            {t("accounts.share")}
          </Link>
          {account.permission === "owner" && (
            <Link
              to="/accounts/$accountId/edit"
              params={{ accountId }}
              className="rounded-md border border-black/15 px-3 py-2 text-sm font-medium transition-colors hover:bg-black/[.04] dark:border-white/15 dark:hover:bg-white/[.06]"
            >
              {t("accounts.details.edit")}
            </Link>
          )}
        </div>
      </header>

      <dl className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-3">
        <div>
          <dt className="text-zinc-500 dark:text-zinc-400">
            {t("accounts.details.balance")}
          </dt>
          <dd
            className={`font-mono text-lg tabular-nums ${
              balance === null ? "" : amountColorClass(balance)
            }`}
          >
            {balance === null
              ? "…"
              : formatAmount(
                  balance,
                  account.currency,
                  displayedDecimalPlaces,
                  i18n.resolvedLanguage ?? "en",
                )}
          </dd>
        </div>
        <div>
          <dt className="text-zinc-500 dark:text-zinc-400">
            {t("accounts.form.currency")}
          </dt>
          <dd>{account.currency}</dd>
        </div>
        <div>
          <dt className="text-zinc-500 dark:text-zinc-400">
            {t("accounts.form.openingDate")}
          </dt>
          <dd>{account.opening_date}</dd>
        </div>
        {account.closing_date && (
          <div>
            <dt className="text-zinc-500 dark:text-zinc-400">
              {t("accounts.form.closingDate")}
            </dt>
            <dd>{account.closing_date}</dd>
          </div>
        )}
        {account.financial_institute && (
          <div>
            <dt className="text-zinc-500 dark:text-zinc-400">
              {t("accounts.form.financialInstitute")}
            </dt>
            <dd>{account.financial_institute}</dd>
          </div>
        )}
        {account.disabled && (
          <div>
            <dt className="text-zinc-500 dark:text-zinc-400">
              {t("accounts.details.status")}
            </dt>
            <dd>{t("accounts.status.disabled")}</dd>
          </div>
        )}
      </dl>

      <section className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-0.5 rounded-md bg-black/[.04] p-0.5 text-sm dark:bg-white/[.06]">
            {(["flow", "balance"] as const).map((view) => (
              <button
                key={view}
                type="button"
                aria-pressed={chartView === view}
                onClick={() => setChartView(view)}
                className={`rounded px-3 py-1 font-medium transition-colors ${
                  chartView === view
                    ? "bg-white text-black shadow-sm dark:bg-neutral-700 dark:text-white"
                    : "text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"
                }`}
              >
                {view === "flow"
                  ? t("flowChart.title")
                  : t("accounts.details.balanceChart.title")}
              </button>
            ))}
          </div>

          {chartView === "balance" && (
            <div className="flex items-center gap-2 text-sm">
              <button
                type="button"
                onClick={() =>
                  setBalanceMonth(
                    (d) => new Date(d.getFullYear(), d.getMonth() - 1, 1),
                  )
                }
                aria-label={t("accounts.details.balanceChart.previousMonth")}
                className="rounded-md px-2 py-1 font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
              >
                ◀
              </button>
              <span className="min-w-32 text-center font-medium tabular-nums">
                {balanceMonth.toLocaleDateString(i18n.resolvedLanguage, {
                  month: "long",
                  year: "numeric",
                })}
              </span>
              <button
                type="button"
                onClick={() =>
                  setBalanceMonth(
                    (d) => new Date(d.getFullYear(), d.getMonth() + 1, 1),
                  )
                }
                aria-label={t("accounts.details.balanceChart.nextMonth")}
                className="rounded-md px-2 py-1 font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
              >
                ▶
              </button>
            </div>
          )}
        </div>

        {chartView === "flow" ? (
          <FlowChart
            accountIds={[accountId]}
            currency={account.currency}
            displayedDecimalPlaces={displayedDecimalPlaces}
          />
        ) : (
          balancePoints !== null && (
            <LineChart
              series={balanceSeries}
              data={balanceData}
              formatValue={(v) =>
                formatAmount(
                  v,
                  account.currency,
                  displayedDecimalPlaces,
                  i18n.resolvedLanguage ?? "en",
                )
              }
            />
          )
        )}
      </section>

      <section className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
            {t("accounts.details.recentEntries")}
          </h2>
          <div className="flex items-center gap-4">
            <Link
              to="/entries"
              search={{ account_id: accountId }}
              className="text-sm font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
            >
              {t("accounts.details.seeAll")}
            </Link>
            {account.permission !== "view" && (
              <Link
                to="/entries/new"
                search={{ account_id: accountId }}
                className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
              >
                {t("entries.create")}
              </Link>
            )}
          </div>
        </div>

        {recent === null ? null : recent.length === 0 ? (
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            {t("accounts.details.noEntries")}
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {recent.map((entry) => (
              <li key={entry.id}>
                <Link
                  to="/entries/$entryId/edit"
                  params={{ entryId: entry.id }}
                  className="flex items-center justify-between gap-4 rounded-lg border border-black/10 px-4 py-3 transition-colors hover:bg-black/[.02] dark:border-white/10 dark:hover:bg-white/[.04]"
                >
                  <div className="flex flex-col gap-0.5">
                    <span className="font-medium">{entry.title}</span>
                    <span className="text-xs text-zinc-500 dark:text-zinc-400">
                      {new Date(entry.booking_timestamp).toLocaleDateString(
                        i18n.resolvedLanguage,
                      )}
                      {entry.created_by !== user?.id &&
                        entry.created_by_name && (
                          <>
                            {" · "}
                            {t("entries.createdBy", {
                              name: entry.created_by_name,
                            })}
                          </>
                        )}
                    </span>
                  </div>
                  <span className="flex flex-col items-end gap-0.5">
                    <span
                      className={`font-mono text-sm tabular-nums ${amountColorClass(
                        entry.kind === "balance_adjustment"
                          ? (entry.balance ?? 0)
                          : entry.amount,
                      )} ${entry.kind === "balance_adjustment" ? "underline" : ""}`}
                    >
                      {formatAmount(
                        entry.kind === "balance_adjustment"
                          ? (entry.balance ?? 0)
                          : entry.amount,
                        account.currency,
                        displayedDecimalPlaces,
                        i18n.resolvedLanguage ?? "en",
                      )}
                    </span>
                    {entry.kind === "balance_adjustment" && (
                      <span className="font-mono text-xs tabular-nums text-zinc-500 dark:text-zinc-400">
                        {formatSignedAmount(
                          entry.amount,
                          account.currency,
                          displayedDecimalPlaces,
                          i18n.resolvedLanguage ?? "en",
                        )}
                      </span>
                    )}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>
    </section>
  );
}
