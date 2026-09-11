import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { AccountLabel } from "../components/AccountLabel";
import { useAuth } from "../components/AuthProvider";
import { amountColorClass, formatAmount } from "../lib/amount";
import { flattenCategoryTree } from "../lib/categoryTree";
import { compact } from "../lib/compact";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";

type Account = components["schemas"]["Account"];
type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];
type Entry = components["schemas"]["Entry"];
type CurrencySum = components["schemas"]["CurrencySum"];

type ReportsSearch = {
  category_id?: string | undefined;
  include_subcategories?: boolean | undefined;
  tag_id?: string | undefined;
  account_id?: string | undefined;
  from?: string | undefined;
  to?: string | undefined;
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
};

const PAGE_SIZE = 30;

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
    from: asString(search["from"]),
    to: asString(search["to"]),
  }),
  component: ReportsPage,
});

function toRangeStart(date: string): string {
  return new Date(`${date}T00:00:00.000Z`).toISOString();
}
function toRangeEnd(date: string): string {
  return new Date(`${date}T23:59:59.999Z`).toISOString();
}

function buildEntriesQuery(gf: GeneratedFilter, after?: string) {
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
    sort: "booking_timestamp" as const,
    dir: "desc" as const,
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
  const { status } = useAuth();
  const navigate = useNavigate();
  const search = Route.useSearch();
  const routeNavigate = Route.useNavigate();
  const { t, i18n } = useTranslation();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();

  useEffect(() => {
    if (status === "anonymous") {
      navigate({ to: "/login", replace: true });
    }
  }, [status, navigate]);

  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);

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
  const [count, setCount] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!generatedFilter) return;
    setLoading(true);
    setItems([]);
    setNextCursor(null);
    setSums([]);
    setCount(0);
    let cancelled = false;
    api
      .GET("/api/entries", {
        params: { query: buildEntriesQuery(generatedFilter) },
      })
      .then(({ data }) => {
        if (cancelled) return;
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
        setCount(data?.count ?? 0);
      });
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
      params: { query: buildEntriesQuery(generatedFilter, nextCursor) },
    });
    setItems((prev) => [...prev, ...(data?.items ?? [])]);
    setNextCursor(data?.next_cursor ?? null);
    setLoadingMore(false);
  }

  function patchSearch(patch: Partial<ReportsSearch>) {
    routeNavigate({ search: (prev) => ({ ...prev, ...patch }) });
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
    setGeneratedFilter({
      categoryId: search.category_id,
      includeSubcategories: search.include_subcategories ?? true,
      tagId: search.tag_id,
      accountId: search.account_id,
      from: search.from,
      to: search.to,
    });
  }

  /** Whether the live filter controls have moved away from the filters the
   * displayed report was generated with — the signal for the "results are
   * out of date" hint, not for re-fetching anything on its own. */
  function isStale(gf: GeneratedFilter, s: ReportsSearch): boolean {
    return (
      gf.categoryId !== s.category_id ||
      gf.includeSubcategories !== (s.include_subcategories ?? true) ||
      gf.tagId !== s.tag_id ||
      gf.accountId !== s.account_id ||
      gf.from !== s.from ||
      gf.to !== s.to
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

        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("reports.filters.from")}
          <input
            type="date"
            className={inputClass}
            value={search.from ?? ""}
            onChange={(e) => patchSearch({ from: e.target.value || undefined })}
          />
        </label>

        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("reports.filters.to")}
          <input
            type="date"
            className={inputClass}
            value={search.to ?? ""}
            onChange={(e) => patchSearch({ to: e.target.value || undefined })}
          />
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
          {sums.map((s) => (
            <span
              key={s.currency}
              className={`font-mono text-lg tabular-nums ${amountColorClass(s.amount)}`}
            >
              {formatAmount(
                s.amount,
                s.currency,
                displayedDecimalPlaces,
                i18n.resolvedLanguage ?? "en",
              )}
            </span>
          ))}
        </div>
      )}

      {hasGenerated && (
        <>
          <div className="overflow-x-auto rounded-lg border border-black/10 dark:border-white/10">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-black/10 text-xs uppercase text-zinc-500 dark:border-white/10 dark:text-zinc-400">
                <tr>
                  <th className="px-3 py-2 font-medium">
                    {t("reports.columns.date")}
                  </th>
                  <th className="px-3 py-2 font-medium">
                    {t("reports.columns.title")}
                  </th>
                  <th className="px-3 py-2 font-medium">
                    {t("reports.columns.account")}
                  </th>
                  <th className="px-3 py-2 text-right font-medium">
                    {t("reports.columns.amount")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {items.map((entry) => (
                  <tr
                    key={entry.id}
                    className="border-b border-black/5 last:border-0 dark:border-white/5"
                  >
                    <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                      {new Date(entry.booking_timestamp).toLocaleString(
                        i18n.resolvedLanguage,
                      )}
                    </td>
                    <td className="px-3 py-2 font-medium">{entry.title}</td>
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
    </section>
  );
}
