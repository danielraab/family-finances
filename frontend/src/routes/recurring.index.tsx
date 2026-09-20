import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { useAuth } from "../components/AuthProvider";
import { CategoryLabel } from "../components/CategoryLabel";
import { FilterPanel, filterControlClass } from "../components/FilterPanel";
import { SelfTransferIcon } from "../components/SelfTransferIcon";
import { useSummaryModals } from "../components/summary/useSummaryModals";
import { amountColorClass, formatAmount } from "../lib/amount";
import { flattenCategoryTree } from "../lib/categoryTree";
import {
  CUSTOM_PRESET_KEY,
  matchPreset,
  perMonthAmount,
} from "../lib/recurrence";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";

type Account = components["schemas"]["Account"];
type Category = components["schemas"]["Category"];
type RecurringTransaction = components["schemas"]["RecurringTransaction"];
type CurrencySum = components["schemas"]["CurrencySum"];
type Tag = components["schemas"]["Tag"];

type RecurringSearch = {
  account_id?: string | undefined;
  category_id?: string | undefined;
  tag_id?: string | undefined;
  include_self_transfer?: boolean | undefined;
  self_transfer_both_legs?: boolean | undefined;
};

function asString(v: unknown): string | undefined {
  return typeof v === "string" && v !== "" ? v : undefined;
}

/**
 * How many filters the visitor has actually applied, for the panel's badge
 * and for deciding whether "Clear all filters" is offered — one per
 * control, the same way /entries counts its own. The revealed both-legs
 * flag is folded into the checkbox that reveals it, since it can never be
 * set on its own.
 */
function activeFilterCount(search: RecurringSearch): number {
  return [
    search.account_id !== undefined,
    search.category_id !== undefined,
    search.tag_id !== undefined,
    search.include_self_transfer === true,
  ].filter(Boolean).length;
}

export const Route = createFileRoute("/recurring/")({
  // Every filter lives in the URL, the way /entries keeps its own state,
  // so a filtered list can be bookmarked, shared and restored by the back
  // button. An absent flag means false in both boolean cases.
  validateSearch: (search: Record<string, unknown>): RecurringSearch => ({
    account_id: asString(search["account_id"]),
    category_id: asString(search["category_id"]),
    tag_id: asString(search["tag_id"]),
    include_self_transfer:
      search["include_self_transfer"] === true ||
      search["include_self_transfer"] === "true"
        ? true
        : undefined,
    self_transfer_both_legs:
      search["self_transfer_both_legs"] === true ||
      search["self_transfer_both_legs"] === "true"
        ? true
        : undefined,
  }),
  component: RecurringTransactionsList,
});

function RecurringTransactionsList() {
  const { t, i18n } = useTranslation();
  const search = Route.useSearch();
  const navigate = useNavigate({ from: "/recurring" });
  const includeSelfTransfer = search.include_self_transfer === true;
  const bothLegs =
    includeSelfTransfer && search.self_transfer_both_legs === true;
  const { user } = useAuth();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();

  const [items, setItems] = useState<RecurringTransaction[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [sums, setSums] = useState<CurrencySum[]>([]);
  const [loading, setLoading] = useState(true);

  const {
    account_id: accountID,
    category_id: categoryID,
    tag_id: tagID,
  } = search;

  useEffect(() => {
    // The list and the summary always go out with the same filters, so the
    // totals below the table are always the totals of the rows in it.
    // account_id is repeatable on the wire; this page sends at most one.
    const query = {
      ...(accountID ? { account_id: [accountID] } : {}),
      ...(categoryID ? { category_id: categoryID } : {}),
      ...(tagID ? { tag_id: tagID } : {}),
      include_self_transfer: includeSelfTransfer,
      self_transfer_both_legs: bothLegs,
    };
    let cancelled = false;
    setLoading(true);
    Promise.all([
      api.GET("/api/recurring-transactions", { params: { query } }),
      api.GET("/api/recurring-transactions/summary", { params: { query } }),
      api.GET("/api/accounts"),
      api.GET("/api/categories"),
      api.GET("/api/tags"),
    ]).then(([list, summary, accts, cats, tgs]) => {
      if (cancelled) return;
      setItems(list.data ?? []);
      setSums(summary.data?.sums ?? []);
      setAccounts(accts.data ?? []);
      setCategories(cats.data ?? []);
      setTags(tgs.data ?? []);
      setLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, [accountID, categoryID, tagID, includeSelfTransfer, bothLegs]);

  // Every control patches the search rather than replacing it, so setting
  // one filter never silently drops another.
  function patchSearch(patch: Partial<RecurringSearch>) {
    navigate({ search: (prev) => ({ ...prev, ...patch }) });
  }

  // Drops every filter in one navigation.
  function clearAllFilters() {
    navigate({ search: {} });
  }

  const categoryById = new Map(categories.map((c) => [c.id, c]));
  const categoryOptions = flattenCategoryTree(categories);
  const { openRecurring, summaryModals } = useSummaryModals({
    accounts,
    categories,
    tags,
    userId: user?.id,
    displayedDecimalPlaces,
  });

  return (
    <section className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-6 py-12 sm:px-10">
      <header className="flex items-center justify-between gap-4">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("recurring.title")}
        </h1>
        <Link
          to="/recurring/new"
          aria-label={t("recurring.create")}
          title={t("recurring.create")}
          className="flex shrink-0 items-center gap-1.5 rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          <Plus size={16} aria-hidden="true" />
          <span className="hidden sm:inline">{t("recurring.create")}</span>
        </Link>
      </header>

      <FilterPanel
        id="recurring-filter-controls"
        activeCount={activeFilterCount(search)}
        onClearAll={clearAllFilters}
      >
        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("recurring.filters.account")}
          <select
            className={filterControlClass}
            value={search.account_id ?? ""}
            onChange={(e) =>
              patchSearch({ account_id: e.target.value || undefined })
            }
          >
            <option value="">{t("recurring.filters.allAccounts")}</option>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.title}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("recurring.filters.category")}
          <select
            className={filterControlClass}
            value={search.category_id ?? ""}
            onChange={(e) =>
              patchSearch({ category_id: e.target.value || undefined })
            }
          >
            <option value="">{t("recurring.filters.allCategories")}</option>
            {categoryOptions.map((c) => (
              <option key={c.id} value={c.id}>
                {c.label}
                {c.shared &&
                  ` — ${t("categories.shared.badgeTitle", { owner: c.ownerName ?? "" })}`}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("recurring.filters.tag")}
          <select
            className={filterControlClass}
            value={search.tag_id ?? ""}
            onChange={(e) =>
              patchSearch({ tag_id: e.target.value || undefined })
            }
          >
            <option value="">{t("recurring.filters.allTags")}</option>
            {tags.map((tag) => (
              <option key={tag.id} value={tag.id}>
                {tag.name}
                {tag.shared &&
                  ` — ${t("tags.shared.badgeTitle", { owner: tag.owner_name ?? "" })}`}
              </option>
            ))}
          </select>
        </label>

        <label className="flex items-center gap-2 pb-1.5 text-sm">
          <input
            type="checkbox"
            checked={includeSelfTransfer}
            onChange={(e) =>
              patchSearch({
                include_self_transfer: e.target.checked ? true : undefined,
                // Unchecking the first clears the second, so the two can
                // never be left in the meaningless "both sides, transfers
                // hidden" combination.
                self_transfer_both_legs: undefined,
              })
            }
          />
          {t("recurring.filters.includeSelfTransfer")}
        </label>
        {includeSelfTransfer && (
          <label className="flex items-center gap-2 pb-1.5 text-sm">
            <input
              type="checkbox"
              checked={bothLegs}
              onChange={(e) =>
                patchSearch({
                  self_transfer_both_legs: e.target.checked ? true : undefined,
                })
              }
            />
            {t("recurring.filters.selfTransferBothLegs")}
          </label>
        )}
      </FilterPanel>

      <div className="overflow-x-auto rounded-lg border border-black/10 dark:border-white/10">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-black/10 text-xs uppercase text-zinc-500 dark:border-white/10 dark:text-zinc-400">
            <tr>
              <th className="px-3 py-2 font-medium">
                {t("recurring.columns.title")}
              </th>
              <th className="px-3 py-2 font-medium">
                {t("recurring.columns.category")}
              </th>
              <th className="px-3 py-2 font-medium">
                {t("recurring.columns.frequency")}
              </th>
              <th className="px-3 py-2 text-right font-medium">
                {t("recurring.columns.amount")}
              </th>
              <th className="px-3 py-2 text-right font-medium">
                {t("recurring.columns.perYearAmount")}
              </th>
              <th className="px-3 py-2 text-right font-medium">
                {t("recurring.columns.perMonthAmount")}
              </th>
              <th className="px-3 py-2 font-medium" />
            </tr>
          </thead>
          <tbody>
            {items.map((rt) => {
              const category = rt.category_id
                ? categoryById.get(rt.category_id)
                : undefined;
              return (
                <tr
                  // Both sides of one transfer share an id, so the leg has
                  // to be part of the key.
                  key={`${rt.id}-${rt.native}`}
                  className={`border-b border-black/5 last:border-0 dark:border-white/5 ${
                    rt.ended ? "opacity-50" : ""
                  }`}
                >
                  <td className="px-3 py-2">
                    <button
                      type="button"
                      onClick={() => openRecurring(rt.id)}
                      className="text-left font-medium underline-offset-2 hover:underline"
                    >
                      {rt.title}
                    </button>
                    {rt.kind === "self_transfer" && (
                      <span className="ml-2 inline-flex items-center gap-1 whitespace-nowrap align-middle text-xs font-normal text-zinc-500 dark:text-zinc-400">
                        <SelfTransferIcon width={14} height={14} />
                        {/* Which side this row is reads off its own amount:
                            money leaving goes to the other account, money
                            arriving comes from it. */}
                        {t(
                          rt.amount < 0
                            ? "recurring.selfTransferTo"
                            : "recurring.selfTransferFrom",
                          { account: rt.to_account_name ?? "" },
                        )}
                      </span>
                    )}
                    {rt.ended && (
                      <span className="ml-2 rounded-full bg-black/[.06] px-2 py-0.5 text-xs font-medium text-zinc-500 dark:bg-white/10 dark:text-zinc-400">
                        {t("recurring.ended")}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                    {category ? (
                      <CategoryLabel category={category} iconSize={16} />
                    ) : (
                      <span aria-hidden>—</span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                    {(() => {
                      const presetKey = matchPreset(
                        rt.interval_unit,
                        rt.interval_count,
                      );
                      return presetKey === CUSTOM_PRESET_KEY
                        ? t("recurring.customFrequency", {
                            count: rt.interval_count,
                            unit: t(`recurring.units.${rt.interval_unit}`),
                          })
                        : t(`recurring.presets.${presetKey}`);
                    })()}
                  </td>
                  <td
                    className={`whitespace-nowrap px-3 py-2 text-right font-mono tabular-nums ${amountColorClass(rt.amount)}`}
                  >
                    {formatAmount(
                      rt.amount,
                      rt.account_currency ?? "",
                      displayedDecimalPlaces,
                      i18n.resolvedLanguage ?? "en",
                    )}
                  </td>
                  <td
                    className={`whitespace-nowrap px-3 py-2 text-right font-mono tabular-nums ${amountColorClass(rt.per_year_amount)}`}
                  >
                    {formatAmount(
                      rt.per_year_amount,
                      rt.account_currency ?? "",
                      displayedDecimalPlaces,
                      i18n.resolvedLanguage ?? "en",
                    )}
                  </td>
                  {/* Coloured by the per-year amount it is derived from, not
                      by itself: the two always share a sign, and this way a
                      per-year amount small enough to round to zero per month
                      can't render a lone grey cell beside a red one. */}
                  <td
                    className={`whitespace-nowrap px-3 py-2 text-right font-mono tabular-nums ${amountColorClass(rt.per_year_amount)}`}
                  >
                    {formatAmount(
                      perMonthAmount(rt.per_year_amount),
                      rt.account_currency ?? "",
                      displayedDecimalPlaces,
                      i18n.resolvedLanguage ?? "en",
                    )}
                  </td>
                  <td className="px-3 py-2 text-right">
                    <Link
                      to="/entries/new"
                      search={{ recurring_transaction_id: rt.id }}
                      aria-label={t("recurring.createTransaction")}
                      title={t("recurring.createTransaction")}
                      className="inline-flex items-center gap-1.5 whitespace-nowrap rounded-md border border-black/15 px-2.5 py-1.5 text-xs font-medium transition-colors hover:bg-black/[.04] dark:border-white/15 dark:hover:bg-white/[.06]"
                    >
                      <Plus size={14} aria-hidden="true" />
                      <span className="hidden sm:inline">
                        {t("recurring.createTransaction")}
                      </span>
                    </Link>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {!loading && items.length === 0 && (
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t(
            activeFilterCount(search) > 0
              ? "recurring.emptyFiltered"
              : "recurring.empty",
          )}
        </p>
      )}

      {sums.length > 0 && (
        <div className="grid grid-cols-1 gap-4 rounded-lg border border-black/10 p-4 text-sm sm:grid-cols-2 dark:border-white/10">
          <TotalGroup
            label={t("recurring.totalPerYear")}
            sums={sums}
            amountOf={(sum) => sum.amount}
            displayedDecimalPlaces={displayedDecimalPlaces}
            locale={i18n.resolvedLanguage ?? "en"}
          />
          <TotalGroup
            label={t("recurring.totalPerMonth")}
            sums={sums}
            amountOf={(sum) => perMonthAmount(sum.amount)}
            displayedDecimalPlaces={displayedDecimalPlaces}
            locale={i18n.resolvedLanguage ?? "en"}
          />
        </div>
      )}

      {summaryModals}
    </section>
  );
}

/**
 * One labelled group of per-currency totals in the list's footer. The
 * page renders two of these side by side — per year, straight from
 * `GET /api/recurring-transactions/summary`, and per month, each of those
 * same totals divided by twelve (see `perMonthAmount`).
 */
function TotalGroup({
  label,
  sums,
  amountOf,
  displayedDecimalPlaces,
  locale,
}: {
  label: string;
  sums: CurrencySum[];
  amountOf: (sum: CurrencySum) => number;
  displayedDecimalPlaces: number;
  locale: string;
}) {
  return (
    <div className="flex flex-col gap-1">
      <span className="font-medium text-zinc-500 dark:text-zinc-400">
        {label}
      </span>
      <div className="flex flex-wrap gap-x-6 gap-y-1">
        {sums.map((sum) => (
          <span
            key={sum.currency}
            className={`whitespace-nowrap font-mono text-base tabular-nums ${amountColorClass(amountOf(sum))}`}
          >
            {formatAmount(
              amountOf(sum),
              sum.currency,
              displayedDecimalPlaces,
              locale,
            )}
          </span>
        ))}
      </div>
    </div>
  );
}
