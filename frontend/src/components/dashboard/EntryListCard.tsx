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
  resolveCardRangeToDateString,
} from "../../lib/dashboardFilter";
import type { WeekStart } from "../../lib/dateRangePresets";
import {
  fetchRecurringPreview,
  type RecurringTransactionPreviewItem,
  resolvePreviewCutoff,
  todayDateString,
} from "../../lib/recurringPreview";
import type { Account } from "../../lib/useAccountsWithBalances";
import { useRecurringPreviewHorizon } from "../../lib/useRecurringPreviewHorizon";
import { AccountLabel } from "../AccountLabel";
import { useAuth } from "../AuthProvider";
import { useSummaryModals } from "../summary/useSummaryModals";
import { UpcomingBlock } from "../UpcomingBlock";
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
  const { user } = useAuth();
  const [items, setItems] = useState<Entry[] | null>(null);
  const [previewItems, setPreviewItems] = useState<
    RecurringTransactionPreviewItem[] | null
  >(null);
  const recurringPreviewHorizon = useRecurringPreviewHorizon();
  const { openEntry, openRecurring, summaryModals } = useSummaryModals({
    accounts,
    categories,
    tags,
    userId: user?.id,
    displayedDecimalPlaces,
  });

  const resolves = cardReferencesResolve(config, accounts, categories, tags);
  const queryKey = JSON.stringify(config);

  const today = new Date();
  const previewCutoff = config.show_recurring_preview
    ? resolvePreviewCutoff(
        resolveCardRangeToDateString(config.range, weekStart),
        recurringPreviewHorizon,
        today,
      )
    : undefined;
  const showUpcoming =
    previewCutoff !== undefined && previewCutoff >= todayDateString(today);

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

  const previewKey = JSON.stringify({ queryKey, previewCutoff, showUpcoming });

  // biome-ignore lint/correctness/useExhaustiveDependencies: previewKey is the stable dependency for the values it embeds.
  useEffect(() => {
    if (!resolves || !showUpcoming || previewCutoff === undefined) {
      setPreviewItems(null);
      return;
    }
    let cancelled = false;
    setPreviewItems(null);
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
  }, [previewKey, resolves, showUpcoming, previewCutoff]);

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

      {showUpcoming && (
        <UpcomingBlock
          items={previewItems}
          accounts={accounts}
          displayedDecimalPlaces={displayedDecimalPlaces}
          locale={locale}
          onOpenRecurring={openRecurring}
          embedded
        />
      )}

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
              // A self-transfer entry can appear twice in an unfiltered
              // card — once per account it touches (see account-entries) —
              // sharing the same id but a different account_id.
              <li
                key={`${entry.id}-${entry.account_id}`}
                className="flex items-center justify-between gap-2 py-1.5 text-sm"
              >
                <div className="flex min-w-0 flex-col">
                  <button
                    type="button"
                    onClick={() => openEntry(entry)}
                    className="truncate text-left font-medium underline-offset-2 hover:underline"
                  >
                    {entry.title}
                  </button>
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

      {summaryModals}
    </div>
  );
}
