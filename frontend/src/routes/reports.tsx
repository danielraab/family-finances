import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { AccountLabel } from "../components/AccountLabel";
import { useAuth } from "../components/AuthProvider";
import { DateRangeFilter } from "../components/DateRangeFilter";
import { RecurringTransactionBadge } from "../components/RecurringTransactionBadge";
import { SelfTransferBadge } from "../components/SelfTransferBadge";
import { useSummaryModals } from "../components/summary/useSummaryModals";
import { UpcomingBlock } from "../components/UpcomingBlock";
import { amountColorClass, formatAmount } from "../lib/amount";
import { flattenCategoryTree } from "../lib/categoryTree";
import { compact } from "../lib/compact";
import { resolveEffectiveRange } from "../lib/dateRangePresets";
import {
  fetchRecurringPreview,
  type RecurringTransactionPreviewItem,
  resolvePreviewCutoff,
  todayDateString,
} from "../lib/recurringPreview";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";
import { usePersistedListFilters } from "../lib/usePersistedListFilters";
import { useRecurringPreviewHorizon } from "../lib/useRecurringPreviewHorizon";
import { useWeekStart } from "../lib/useWeekStart";

type Account = components["schemas"]["Account"];
type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];
type Entry = components["schemas"]["Entry"];
type CurrencySum = components["schemas"]["CurrencySum"];
type Sort = "booking_timestamp" | "amount";
type Dir = "asc" | "desc";

type ReportsSearch = {
  category_id?: string | undefined;
  include_subcategories?: boolean | undefined;
  tag_id?: string | undefined;
  account_id?: string | undefined;
  range?: string | undefined;
  from?: string | undefined;
  to?: string | undefined;
  show_recurring?: boolean | undefined;
  sort?: Sort | undefined;
  dir?: Dir | undefined;
};

/** The filters "Generate report" was last activated with — a snapshot,
 * distinct from the live (URL-backed) draft controls, so changing a filter
 * never triggers a fetch on its own. */
type GeneratedFilter = {
  categoryId?: string | undefined;
  includeSubcategories: boolean;
  tagId?: string | undefined;
  accountId?: string | undefined;
  from?: string | undefined;
  to?: string | undefined;
  showRecurring: boolean;
  /** The resolved preview cutoff, only set when showRecurring is on AND it
   * resolves on/after today — see design.md's "a cutoff already before
   * today means the block simply doesn't render" rule. Drives both
   * whether the preview is fetched and whether the Upcoming block
   * renders. */
  previewCutoff?: string | undefined;
};

const PAGE_SIZE = 30;

// The most recently applied filter state, persisted per-browser so
// arriving at a bare /reports (a sidebar click, or any other ordinary
// navigation) restores it instead of resetting to the default view. See
// web-client-reports' "Returning to the report restores the last-applied
// filters" requirement. Only the draft controls are restored — "Generate
// report" still requires an explicit click, exactly as for a bookmarked
// URL today.
const LAST_FILTERS_KEY = "ff:reports-last-filters";

function asString(v: unknown): string | undefined {
  return typeof v === "string" && v !== "" ? v : undefined;
}

export const Route = createFileRoute("/reports")({
  validateSearch: (search: Record<string, unknown>): ReportsSearch => ({
    category_id: asString(search["category_id"]),
    include_subcategories:
      typeof search["include_subcategories"] === "boolean"
        ? search["include_subcategories"]
        : undefined,
    tag_id: asString(search["tag_id"]),
    account_id: asString(search["account_id"]),
    range: asString(search["range"]),
    from: asString(search["from"]),
    to: asString(search["to"]),
    show_recurring:
      typeof search["show_recurring"] === "boolean"
        ? search["show_recurring"]
        : undefined,
    sort: search["sort"] === "amount" ? "amount" : undefined,
    dir: search["dir"] === "asc" ? "asc" : undefined,
  }),
  component: ReportsPage,
});

function toRangeStart(date: string): string {
  return new Date(`${date}T00:00:00.000Z`).toISOString();
}
function toRangeEnd(date: string): string {
  return new Date(`${date}T23:59:59.999Z`).toISOString();
}

function buildEntriesQuery(
  gf: GeneratedFilter,
  sort: Sort,
  dir: Dir,
  after?: string,
) {
  return compact({
    account_id: gf.accountId ? [gf.accountId] : undefined,
    category_id: gf.categoryId,
    category_mode:
      gf.categoryId && !gf.includeSubcategories
        ? ("exact" as const)
        : undefined,
    tag_id: gf.tagId,
    from: gf.from ? toRangeStart(gf.from) : undefined,
    to: gf.to ? toRangeEnd(gf.to) : undefined,
    sort,
    dir,
    after,
    limit: PAGE_SIZE,
  });
}

function buildSummaryQuery(gf: GeneratedFilter) {
  return compact({
    account_id: gf.accountId ? [gf.accountId] : undefined,
    category_id: gf.categoryId,
    category_mode:
      gf.categoryId && !gf.includeSubcategories
        ? ("exact" as const)
        : undefined,
    tag_id: gf.tagId,
    from: gf.from ? toRangeStart(gf.from) : undefined,
    to: gf.to ? toRangeEnd(gf.to) : undefined,
  });
}

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-2.5 py-1.5 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

function ReportsPage() {
  const { status, user } = useAuth();
  const navigate = useNavigate();
  const search = Route.useSearch();
  const routeNavigate = Route.useNavigate();
  usePersistedListFilters(LAST_FILTERS_KEY, search, routeNavigate);
  const { t, i18n } = useTranslation();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();
  const weekStart = useWeekStart();
  const recurringPreviewHorizon = useRecurringPreviewHorizon();

  useEffect(() => {
    if (status === "anonymous") {
      navigate({ to: "/login", replace: true });
    }
  }, [status, navigate]);

  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const { openEntry, openRecurring, summaryModals } = useSummaryModals({
    accounts,
    categories,
    tags,
    userId: user?.id,
    displayedDecimalPlaces,
  });

  useEffect(() => {
    Promise.all([
      api.GET("/api/accounts"),
      api.GET("/api/categories"),
      api.GET("/api/tags"),
    ]).then(([a, c, tg]) => {
      setAccounts(a.data ?? []);
      setCategories(c.data ?? []);
      setTags(tg.data ?? []);
    });
  }, []);

  const [generatedFilter, setGeneratedFilter] =
    useState<GeneratedFilter | null>(null);

  const [items, setItems] = useState<Entry[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [sums, setSums] = useState<CurrencySum[]>([]);
  const [income, setIncome] = useState<CurrencySum[]>([]);
  const [outcome, setOutcome] = useState<CurrencySum[]>([]);
  const [count, setCount] = useState(0);
  const [previewItems, setPreviewItems] = useState<
    RecurringTransactionPreviewItem[] | null
  >(null);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const sentinelRef = useRef<HTMLDivElement | null>(null);
  // Coordinates the entries page fetch between this effect and toggleSort's
  // own, independent re-fetch below, so a slow response from one can't
  // clobber state a faster, later request already settled.
  const entriesRequestRef = useRef(0);

  // biome-ignore lint/correctness/useExhaustiveDependencies: search.sort/dir are read fresh at generation time only — sort changes re-fetch independently via toggleSort, without re-running this effect or its summary/preview requests.
  useEffect(() => {
    if (!generatedFilter) return;
    const requestId = ++entriesRequestRef.current;
    setLoading(true);
    setItems([]);
    setNextCursor(null);
    setSums([]);
    setIncome([]);
    setOutcome([]);
    setCount(0);
    setPreviewItems(null);
    let cancelled = false;
    api
      .GET("/api/entries", {
        params: {
          query: buildEntriesQuery(
            generatedFilter,
            search.sort ?? "booking_timestamp",
            search.dir ?? "desc",
          ),
        },
      })
      .then(({ data }) => {
        if (cancelled || entriesRequestRef.current !== requestId) return;
        setItems(data?.items ?? []);
        setNextCursor(data?.next_cursor ?? null);
        setLoading(false);
      });
    api
      .GET("/api/entries/summary", {
        params: { query: buildSummaryQuery(generatedFilter) },
      })
      .then(({ data }) => {
        if (cancelled) return;
        setSums(data?.sums ?? []);
        setIncome(data?.income ?? []);
        setOutcome(data?.outcome ?? []);
        setCount(data?.count ?? 0);
      });
    // The Upcoming preview is fetched alongside the entries/summary
    // requests, but never folds into either — see web-client-reports'
    // "Previewed occurrences never affect the report's per-currency sum".
    if (generatedFilter.previewCutoff) {
      fetchRecurringPreview({
        accountIds: generatedFilter.accountId
          ? [generatedFilter.accountId]
          : undefined,
        categoryId: generatedFilter.categoryId,
        categoryMode:
          generatedFilter.categoryId && !generatedFilter.includeSubcategories
            ? "exact"
            : undefined,
        tagId: generatedFilter.tagId,
        to: generatedFilter.previewCutoff,
      }).then(({ data }) => {
        if (!cancelled) setPreviewItems(data?.items ?? []);
      });
    }
    return () => {
      cancelled = true;
    };
  }, [generatedFilter]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: loadMore closes over current state; re-subscribing on it would just re-run this identically.
  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || !nextCursor || loading) return;
    const observer = new IntersectionObserver((entries) => {
      const first = entries[0];
      if (first?.isIntersecting) {
        loadMore();
      }
    });
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [nextCursor, loading]);

  async function loadMore() {
    if (loadingMore || !nextCursor || !generatedFilter) return;
    setLoadingMore(true);
    const { data } = await api.GET("/api/entries", {
      params: {
        query: buildEntriesQuery(
          generatedFilter,
          search.sort ?? "booking_timestamp",
          search.dir ?? "desc",
          nextCursor,
        ),
      },
    });
    setItems((prev) => [...prev, ...(data?.items ?? [])]);
    setNextCursor(data?.next_cursor ?? null);
    setLoadingMore(false);
  }

  function patchSearch(patch: Partial<ReportsSearch>) {
    routeNavigate({ search: (prev) => ({ ...prev, ...patch }) });
  }

  /** Sorts the results table immediately, independent of the "Generate
   * report" gate — see web-client-reports' "The report result table is
   * sortable by date or amount": it reorders the already-generated result
   * set rather than changing which entries are included, so it never marks
   * the report stale and never waits for "Generate report" again. */
  function toggleSort(column: Sort) {
    const currentSort = search.sort ?? "booking_timestamp";
    const currentDir = search.dir ?? "desc";
    const nextDir: Dir =
      currentSort === column ? (currentDir === "asc" ? "desc" : "asc") : "desc";
    patchSearch({ sort: column, dir: nextDir });

    if (!generatedFilter) return;
    const requestId = ++entriesRequestRef.current;
    setLoading(true);
    setItems([]);
    setNextCursor(null);
    api
      .GET("/api/entries", {
        params: {
          query: buildEntriesQuery(generatedFilter, column, nextDir),
        },
      })
      .then(({ data }) => {
        if (entriesRequestRef.current !== requestId) return;
        setItems(data?.items ?? []);
        setNextCursor(data?.next_cursor ?? null);
        setLoading(false);
      });
  }

  function selectCategory(id: string) {
    patchSearch(
      id
        ? { category_id: id }
        : { category_id: undefined, include_subcategories: undefined },
    );
  }

  function selectTag(id: string) {
    patchSearch({ tag_id: id || undefined });
  }

  function generateReport() {
    const today = new Date();
    const effective = resolveEffectiveRange(
      { range: search.range, from: search.from, to: search.to },
      weekStart,
      undefined,
      today,
    );
    const showRecurring = search.show_recurring ?? false;
    const cutoff = resolvePreviewCutoff(
      effective.to,
      recurringPreviewHorizon,
      today,
    );
    setGeneratedFilter({
      categoryId: search.category_id,
      includeSubcategories: search.include_subcategories ?? true,
      tagId: search.tag_id,
      accountId: search.account_id,
      from: effective.from,
      to: effective.to,
      showRecurring,
      previewCutoff:
        showRecurring && cutoff >= todayDateString(today) ? cutoff : undefined,
    });
  }

  /** Whether the live filter controls have moved away from the filters the
   * displayed report was generated with — the signal for the "results are
   * out of date" hint, not for re-fetching anything on its own. `gf.from`/
   * `gf.to` are the *resolved* dates a preset produced at generation time,
   * so staleness compares against the live filter's own resolved dates,
   * not its raw `range`/`from`/`to` representation. */
  function isStale(gf: GeneratedFilter, s: ReportsSearch): boolean {
    const liveEffective = resolveEffectiveRange(
      { range: s.range, from: s.from, to: s.to },
      weekStart,
      undefined,
      new Date(),
    );
    return (
      gf.categoryId !== s.category_id ||
      gf.includeSubcategories !== (s.include_subcategories ?? true) ||
      gf.tagId !== s.tag_id ||
      gf.accountId !== s.account_id ||
      gf.from !== liveEffective.from ||
      gf.to !== liveEffective.to ||
      gf.showRecurring !== (s.show_recurring ?? false)
    );
  }

  if (status !== "authenticated") {
    return null;
  }

  const categoryOptions = flattenCategoryTree(categories);
  const hasGenerated = generatedFilter !== null;
  const stale = generatedFilter !== null && isStale(generatedFilter, search);

  return (
    <section className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-6 py-12 sm:px-10">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("reports.title")}
        </h1>
      </header>

      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("reports.filters.category")}
          <select
            className={inputClass}
            value={search.category_id ?? ""}
            onChange={(e) => selectCategory(e.target.value)}
          >
            <option value="">{t("reports.filters.selectCategory")}</option>
            {categoryOptions.map((c) => (
              <option key={c.id} value={c.id}>
                {c.label}
                {c.shared &&
                  ` — ${t("categories.shared.badgeTitle", { owner: c.ownerName ?? "" })}`}
              </option>
            ))}
          </select>
        </label>

        {search.category_id && (
          <label className="flex items-center gap-2 pb-1.5 text-sm">
            <input
              type="checkbox"
              checked={search.include_subcategories ?? true}
              onChange={(e) =>
                patchSearch({
                  include_subcategories: e.target.checked ? undefined : false,
                })
              }
            />
            {t("reports.filters.includeSubcategories")}
          </label>
        )}

        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("reports.filters.tag")}
          <select
            className={inputClass}
            value={search.tag_id ?? ""}
            onChange={(e) => selectTag(e.target.value)}
          >
            <option value="">{t("reports.filters.selectTag")}</option>
            {tags.map((tag) => (
              <option key={tag.id} value={tag.id}>
                {tag.name}
                {tag.shared &&
                  ` — ${t("tags.shared.badgeTitle", { owner: tag.owner_name ?? "" })}`}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("reports.filters.account")}
          <select
            className={inputClass}
            value={search.account_id ?? ""}
            onChange={(e) =>
              patchSearch({ account_id: e.target.value || undefined })
            }
          >
            <option value="">{t("reports.filters.allAccounts")}</option>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.title}
              </option>
            ))}
          </select>
        </label>

        <DateRangeFilter
          value={{ range: search.range, from: search.from, to: search.to }}
          weekStart={weekStart}
          onChange={(patch) => patchSearch(patch)}
          fromLabel={t("reports.filters.from")}
          toLabel={t("reports.filters.to")}
        />

        <label className="flex items-center gap-2 pb-1.5 text-sm">
          <input
            type="checkbox"
            checked={search.show_recurring ?? false}
            onChange={(e) =>
              patchSearch({ show_recurring: e.target.checked || undefined })
            }
          />
          {t("recurringPreview.toggleLabel")}
        </label>

        <button
          type="button"
          onClick={generateReport}
          className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {t("reports.generate")}
        </button>
      </div>

      {stale && (
        <p className="text-sm text-amber-600 dark:text-amber-400">
          {t("reports.staleHint")}
        </p>
      )}

      {!hasGenerated && (
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("reports.beforeGenerate")}
        </p>
      )}

      {hasGenerated && sums.length > 0 && (
        <div className="flex flex-wrap items-center gap-4 rounded-lg border border-black/10 px-4 py-3 dark:border-white/10">
          <span className="text-xs font-medium uppercase text-zinc-500 dark:text-zinc-400">
            {t("reports.sumsTitle")}
          </span>
          {sums.map((s) => {
            const currencyIncome = income.find(
              (i) => i.currency === s.currency,
            );
            const currencyOutcome = outcome.find(
              (o) => o.currency === s.currency,
            );
            return (
              <span key={s.currency} className="flex items-baseline gap-3">
                <span
                  className={`font-mono text-lg tabular-nums ${amountColorClass(s.amount)}`}
                >
                  {formatAmount(
                    s.amount,
                    s.currency,
                    displayedDecimalPlaces,
                    i18n.resolvedLanguage ?? "en",
                  )}
                </span>
                {currencyIncome && (
                  <span className="text-sm text-zinc-500 dark:text-zinc-400">
                    {t("reports.income")}{" "}
                    <span
                      className={`font-mono tabular-nums ${amountColorClass(currencyIncome.amount)}`}
                    >
                      {formatAmount(
                        currencyIncome.amount,
                        s.currency,
                        displayedDecimalPlaces,
                        i18n.resolvedLanguage ?? "en",
                      )}
                    </span>
                  </span>
                )}
                {currencyOutcome && (
                  <span className="text-sm text-zinc-500 dark:text-zinc-400">
                    {t("reports.outcome")}{" "}
                    <span
                      className={`font-mono tabular-nums ${amountColorClass(-currencyOutcome.amount)}`}
                    >
                      {formatAmount(
                        -currencyOutcome.amount,
                        s.currency,
                        displayedDecimalPlaces,
                        i18n.resolvedLanguage ?? "en",
                      )}
                    </span>
                  </span>
                )}
              </span>
            );
          })}
        </div>
      )}

      {generatedFilter?.previewCutoff && (
        <UpcomingBlock
          items={previewItems}
          accounts={accounts}
          displayedDecimalPlaces={displayedDecimalPlaces}
          locale={i18n.resolvedLanguage ?? "en"}
          onOpenRecurring={openRecurring}
        />
      )}

      {hasGenerated && (
        <>
          <div className="overflow-x-auto rounded-lg border border-black/10 dark:border-white/10">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-black/10 text-xs uppercase text-zinc-500 dark:border-white/10 dark:text-zinc-400">
                <tr>
                  <th className="px-3 py-2 font-medium">
                    <button
                      type="button"
                      onClick={() => toggleSort("booking_timestamp")}
                      className="flex items-center gap-1"
                    >
                      {t("reports.columns.date")}
                      {(search.sort ?? "booking_timestamp") ===
                        "booking_timestamp" && (
                        <span>
                          {(search.dir ?? "desc") === "asc" ? "↑" : "↓"}
                        </span>
                      )}
                    </button>
                  </th>
                  <th className="px-3 py-2 font-medium">
                    {t("reports.columns.title")}
                  </th>
                  <th className="px-3 py-2 font-medium">
                    {t("reports.columns.account")}
                  </th>
                  <th className="px-3 py-2 text-right font-medium">
                    <button
                      type="button"
                      onClick={() => toggleSort("amount")}
                      className="flex items-center gap-1"
                    >
                      {t("reports.columns.amount")}
                      {search.sort === "amount" && (
                        <span>
                          {(search.dir ?? "desc") === "asc" ? "↑" : "↓"}
                        </span>
                      )}
                    </button>
                  </th>
                </tr>
              </thead>
              <tbody>
                {items.map((entry) => (
                  // A self-transfer entry can appear twice in this list —
                  // once per account it touches, when both are in scope
                  // (see account-entries) — sharing the same id but a
                  // different account_id, so the key needs both.
                  <tr
                    key={`${entry.id}-${entry.account_id}`}
                    className="border-b border-black/5 last:border-0 dark:border-white/5"
                  >
                    <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                      {new Date(entry.booking_timestamp).toLocaleString(
                        i18n.resolvedLanguage,
                      )}
                    </td>
                    <td className="px-3 py-2 font-medium">
                      <button
                        type="button"
                        onClick={() => openEntry(entry)}
                        className="text-left font-medium underline-offset-2 hover:underline"
                      >
                        {entry.title}
                      </button>
                      <RecurringTransactionBadge
                        recurringTransactionId={entry.recurring_transaction_id}
                        onOpen={openRecurring}
                      />
                      <SelfTransferBadge
                        kind={entry.kind}
                        onOpen={() => openEntry(entry)}
                      />
                    </td>
                    <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                      {(() => {
                        const account = accounts.find(
                          (a) => a.id === entry.account_id,
                        );
                        return account ? (
                          <AccountLabel account={account} iconSize={16} />
                        ) : (
                          <span className="italic">
                            {t("entries.notShared")}
                          </span>
                        );
                      })()}
                    </td>
                    <td
                      className={`px-3 py-2 text-right font-mono tabular-nums ${amountColorClass(entry.amount)}`}
                    >
                      {formatAmount(
                        entry.amount,
                        entry.account_currency ?? "",
                        displayedDecimalPlaces,
                        i18n.resolvedLanguage ?? "en",
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {!loading && count === 0 && (
            <p className="text-sm text-zinc-500 dark:text-zinc-400">
              {t("reports.emptyFiltered")}
            </p>
          )}

          {(loading || loadingMore) && (
            <p className="text-center text-sm text-zinc-500 dark:text-zinc-400">
              {t("reports.loading")}
            </p>
          )}

          <div ref={sentinelRef} className="h-1" />
        </>
      )}

      {summaryModals}
    </section>
  );
}
