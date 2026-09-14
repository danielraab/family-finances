import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../../api/client";
import type { components } from "../../api/schema";
import { amountColorClass, formatAmount } from "../../lib/amount";
import {
  buildCardFilterQuery,
  cardReferencesResolve,
  type DashboardCardConfig,
} from "../../lib/dashboardFilter";
import type { WeekStart } from "../../lib/dateRangePresets";
import type { Account } from "../../lib/useAccountsWithBalances";
import { CardTitleLink } from "./CardTitleLink";
import { MissingReferenceCard } from "./MissingReferenceCard";

type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];
type CurrencySum = components["schemas"]["CurrencySum"];

/**
 * A query_stat card: the per-currency sum (and count) of entries matching
 * config's inline filter, via GET /api/entries/summary — the same
 * endpoint /reports uses for its own totals.
 */
export function QueryStatCard({
  config,
  accounts,
  categories,
  tags,
  weekStart,
  displayedDecimalPlaces,
  locale,
}: {
  config: DashboardCardConfig;
  accounts: Account[];
  categories: Category[];
  tags: Tag[];
  weekStart: WeekStart;
  displayedDecimalPlaces: number;
  locale: string;
}) {
  const { t } = useTranslation();
  const [sums, setSums] = useState<CurrencySum[] | null>(null);
  const [count, setCount] = useState(0);

  const resolves = cardReferencesResolve(config, accounts, categories, tags);
  const queryKey = JSON.stringify(config);

  // biome-ignore lint/correctness/useExhaustiveDependencies: queryKey is config's stable stand-in; config itself is a new object identity each render.
  useEffect(() => {
    if (!resolves) return;
    let cancelled = false;
    setSums(null);
    api
      .GET("/api/entries/summary", {
        params: { query: buildCardFilterQuery(config, weekStart) },
      })
      .then(({ data }) => {
        if (cancelled) return;
        setSums(data?.sums ?? []);
        setCount(data?.count ?? 0);
      });
    return () => {
      cancelled = true;
    };
  }, [queryKey, weekStart, resolves]);

  if (!resolves) {
    return <MissingReferenceCard />;
  }

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-black/10 p-4 dark:border-white/10">
      <CardTitleLink
        config={config}
        accounts={accounts}
        categories={categories}
        tags={tags}
        allLabel={t("dashboard.filterAllAccounts")}
      />
      {sums === null ? (
        <span className="text-lg">…</span>
      ) : sums.length === 0 ? (
        <span className="text-lg text-zinc-400 dark:text-zinc-500">—</span>
      ) : (
        <div className="flex flex-col gap-1">
          {sums.map((s) => (
            <span
              key={s.currency}
              className={`font-mono text-lg tabular-nums ${amountColorClass(s.amount)}`}
            >
              {formatAmount(
                s.amount,
                s.currency,
                displayedDecimalPlaces,
                locale,
              )}
            </span>
          ))}
        </div>
      )}
      <span className="text-xs text-zinc-500 dark:text-zinc-400">
        {t("dashboard.entryCount", { count })}
      </span>
    </div>
  );
}
