import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../../api/client";
import type { components } from "../../api/schema";
import { amountColorClass, formatAmount } from "../../lib/amount";
import { compact } from "../../lib/compact";
import {
  buildCardFilterQuery,
  cardReferencesResolve,
  type DashboardCardConfig,
} from "../../lib/dashboardFilter";
import type { WeekStart } from "../../lib/dateRangePresets";
import type { Account } from "../../lib/useAccountsWithBalances";
import { AccountLabel } from "../AccountLabel";
import { CardTitleLink } from "./CardTitleLink";
import { MissingReferenceCard } from "./MissingReferenceCard";

type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];
type Entry = components["schemas"]["Entry"];

const PAGE_SIZE = 10;

/**
 * An entry_list card: the most recent entries matching config's inline
 * filter (fixed at 10 — no configurable page size), via the same
 * GET /api/entries listing /reports uses, sorted booking_timestamp desc.
 */
export function EntryListCard({
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
  const [items, setItems] = useState<Entry[] | null>(null);

  const resolves = cardReferencesResolve(config, accounts, categories, tags);
  const queryKey = JSON.stringify(config);

  // biome-ignore lint/correctness/useExhaustiveDependencies: queryKey is config's stable stand-in; config itself is a new object identity each render.
  useEffect(() => {
    if (!resolves) return;
    let cancelled = false;
    setItems(null);
    api
      .GET("/api/entries", {
        params: {
          query: compact({
            ...buildCardFilterQuery(config, weekStart),
            sort: "booking_timestamp" as const,
            dir: "desc" as const,
            limit: PAGE_SIZE,
          }),
        },
      })
      .then(({ data }) => {
        if (cancelled) return;
        setItems(data?.items ?? []);
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

      {items === null ? (
        <span className="text-sm text-zinc-500 dark:text-zinc-400">…</span>
      ) : items.length === 0 ? (
        <span className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("dashboard.entryListEmpty")}
        </span>
      ) : (
        <ul className="flex flex-col divide-y divide-black/5 dark:divide-white/5">
          {items.map((entry) => {
            const account = accounts.find((a) => a.id === entry.account_id);
            return (
              <li
                key={entry.id}
                className="flex items-center justify-between gap-2 py-1.5 text-sm"
              >
                <div className="flex min-w-0 flex-col">
                  <span className="truncate font-medium">{entry.title}</span>
                  <span className="flex items-center gap-1 text-xs text-zinc-500 dark:text-zinc-400">
                    {new Date(entry.booking_timestamp).toLocaleDateString(
                      locale,
                    )}
                    {account && (
                      <>
                        {" · "}
                        <AccountLabel account={account} iconSize={14} />
                      </>
                    )}
                  </span>
                </div>
                <span
                  className={`shrink-0 font-mono text-sm tabular-nums ${amountColorClass(entry.amount)}`}
                >
                  {formatAmount(
                    entry.amount,
                    entry.account_currency ?? "",
                    displayedDecimalPlaces,
                    locale,
                  )}
                </span>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
