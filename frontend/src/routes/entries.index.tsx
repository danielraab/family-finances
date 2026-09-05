import { createFileRoute, Link } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { formatAmount } from "../lib/amount";
import { flattenCategoryTree } from "../lib/categoryTree";
import { compact } from "../lib/compact";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";

type Account = components["schemas"]["Account"];
type Category = components["schemas"]["Category"];
type Tag = components["schemas"]["Tag"];
type Entry = components["schemas"]["Entry"];
type EntryKind = components["schemas"]["EntryKind"];
type Sort = "booking_timestamp" | "amount";
type Dir = "asc" | "desc";

type EntriesSearch = {
  account_id?: string | undefined;
  category_id?: string | undefined;
  tag_id?: string | undefined;
  kind?: EntryKind | undefined;
  from?: string | undefined;
  to?: string | undefined;
  q?: string | undefined;
  sort?: Sort | undefined;
  dir?: Dir | undefined;
};

const PAGE_SIZE = 30;

function asString(v: unknown): string | undefined {
  return typeof v === "string" && v !== "" ? v : undefined;
}

export const Route = createFileRoute("/entries/")({
  validateSearch: (search: Record<string, unknown>): EntriesSearch => ({
    account_id: asString(search["account_id"]),
    category_id: asString(search["category_id"]),
    tag_id: asString(search["tag_id"]),
    kind:
      search["kind"] === "transaction" ||
      search["kind"] === "balance_adjustment"
        ? search["kind"]
        : undefined,
    from: asString(search["from"]),
    to: asString(search["to"]),
    q: asString(search["q"]),
    sort: search["sort"] === "amount" ? "amount" : undefined,
    dir: search["dir"] === "asc" ? "asc" : undefined,
  }),
  component: EntriesListPage,
});

function toRangeStart(date: string): string {
  return new Date(`${date}T00:00:00.000Z`).toISOString();
}
function toRangeEnd(date: string): string {
  return new Date(`${date}T23:59:59.999Z`).toISOString();
}

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-2.5 py-1.5 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

function EntriesListPage() {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();
  const { t, i18n } = useTranslation();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();

  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [qDraft, setQDraft] = useState(search.q ?? "");

  const [items, setItems] = useState<Entry[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const sentinelRef = useRef<HTMLDivElement | null>(null);
  const qDebounceRef = useRef<number | undefined>(undefined);

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

  const searchKey = JSON.stringify(search);

  function buildQuery(after?: string) {
    return compact({
      account_id: search.account_id ? [search.account_id] : undefined,
      category_id: search.category_id,
      tag_id: search.tag_id,
      kind: search.kind,
      from: search.from ? toRangeStart(search.from) : undefined,
      to: search.to ? toRangeEnd(search.to) : undefined,
      q: search.q,
      sort: search.sort ?? "booking_timestamp",
      dir: search.dir ?? "desc",
      after,
      limit: PAGE_SIZE,
    });
  }

  // biome-ignore lint/correctness/useExhaustiveDependencies: searchKey is the stable dependency; search itself is a new object each render.
  useEffect(() => {
    setQDraft(search.q ?? "");
    setLoading(true);
    let cancelled = false;
    api
      .GET("/api/entries", { params: { query: buildQuery() } })
      .then(({ data }) => {
        if (cancelled) return;
        setItems(data?.items ?? []);
        setNextCursor(data?.next_cursor ?? null);
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [searchKey]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: loadMore is re-created each render and closes over current state; re-subscribing on it would just re-run this identically.
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
    if (loadingMore || !nextCursor) return;
    setLoadingMore(true);
    const { data } = await api.GET("/api/entries", {
      params: { query: buildQuery(nextCursor) },
    });
    setItems((prev) => [...prev, ...(data?.items ?? [])]);
    setNextCursor(data?.next_cursor ?? null);
    setLoadingMore(false);
  }

  function patchSearch(patch: Partial<EntriesSearch>) {
    navigate({ search: (prev) => ({ ...prev, ...patch }) });
  }

  function toggleSort(column: Sort) {
    if (
      search.sort === column ||
      (!search.sort && column === "booking_timestamp")
    ) {
      patchSearch({ sort: column, dir: search.dir === "asc" ? "desc" : "asc" });
    } else {
      patchSearch({ sort: column, dir: "desc" });
    }
  }

  const categoryOptions = flattenCategoryTree(categories);
  const accountCurrency = (accountId: string) =>
    accounts.find((a) => a.id === accountId)?.currency ?? "";

  return (
    <section className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-6 py-12 sm:px-10">
      <header className="flex items-center justify-between gap-4">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("entries.title")}
        </h1>
        <Link
          to="/entries/new"
          search={search.account_id ? { account_id: search.account_id } : {}}
          className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {t("entries.create")}
        </Link>
      </header>

      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("entries.filters.account")}
          <select
            className={inputClass}
            value={search.account_id ?? ""}
            onChange={(e) =>
              patchSearch({ account_id: e.target.value || undefined })
            }
          >
            <option value="">{t("entries.filters.allAccounts")}</option>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.title}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("entries.filters.category")}
          <select
            className={inputClass}
            value={search.category_id ?? ""}
            onChange={(e) =>
              patchSearch({ category_id: e.target.value || undefined })
            }
          >
            <option value="">{t("entries.filters.allCategories")}</option>
            {categoryOptions.map((c) => (
              <option key={c.id} value={c.id}>
                {c.label}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("entries.filters.tag")}
          <select
            className={inputClass}
            value={search.tag_id ?? ""}
            onChange={(e) =>
              patchSearch({ tag_id: e.target.value || undefined })
            }
          >
            <option value="">{t("entries.filters.allTags")}</option>
            {tags.map((tag) => (
              <option key={tag.id} value={tag.id}>
                {tag.name}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("entries.filters.kind")}
          <select
            className={inputClass}
            value={search.kind ?? ""}
            onChange={(e) =>
              patchSearch({
                kind: (e.target.value || undefined) as EntryKind | undefined,
              })
            }
          >
            <option value="">{t("entries.filters.allKinds")}</option>
            <option value="transaction">{t("entries.kind.transaction")}</option>
            <option value="balance_adjustment">
              {t("entries.kind.balanceAdjustment")}
            </option>
          </select>
        </label>

        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("entries.filters.from")}
          <input
            type="date"
            className={inputClass}
            value={search.from ?? ""}
            onChange={(e) => patchSearch({ from: e.target.value || undefined })}
          />
        </label>

        <label className="flex flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("entries.filters.to")}
          <input
            type="date"
            className={inputClass}
            value={search.to ?? ""}
            onChange={(e) => patchSearch({ to: e.target.value || undefined })}
          />
        </label>

        <label className="flex flex-1 flex-col gap-1 text-xs font-medium text-zinc-500 dark:text-zinc-400">
          {t("entries.filters.search")}
          <input
            type="search"
            className={inputClass}
            value={qDraft}
            onChange={(e) => {
              const value = e.target.value;
              setQDraft(value);
              window.clearTimeout(qDebounceRef.current);
              qDebounceRef.current = window.setTimeout(
                () => patchSearch({ q: value || undefined }),
                300,
              );
            }}
          />
        </label>
      </div>

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
                  {t("entries.columns.date")}
                  {(search.sort ?? "booking_timestamp") ===
                    "booking_timestamp" && (
                    <span>{(search.dir ?? "desc") === "asc" ? "↑" : "↓"}</span>
                  )}
                </button>
              </th>
              <th className="px-3 py-2 font-medium">
                {t("entries.columns.title")}
              </th>
              <th className="px-3 py-2 font-medium">
                {t("entries.columns.account")}
              </th>
              <th className="px-3 py-2 text-right font-medium">
                <button
                  type="button"
                  onClick={() => toggleSort("amount")}
                  className="flex items-center gap-1"
                >
                  {t("entries.columns.amount")}
                  {search.sort === "amount" && (
                    <span>{(search.dir ?? "desc") === "asc" ? "↑" : "↓"}</span>
                  )}
                </button>
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
                <td className="px-3 py-2">
                  <Link
                    to="/entries/$entryId/edit"
                    params={{ entryId: entry.id }}
                    className="font-medium underline-offset-2 hover:underline"
                  >
                    {entry.title}
                  </Link>
                </td>
                <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                  {accounts.find((a) => a.id === entry.account_id)?.title ??
                    entry.account_id}
                </td>
                <td className="px-3 py-2 text-right font-mono tabular-nums">
                  {formatAmount(
                    entry.amount,
                    accountCurrency(entry.account_id),
                    displayedDecimalPlaces,
                    i18n.resolvedLanguage ?? "en",
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {!loading && items.length === 0 && (
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {search.account_id ||
          search.category_id ||
          search.tag_id ||
          search.kind ||
          search.from ||
          search.to ||
          search.q
            ? t("entries.emptyFiltered")
            : t("entries.emptyAll")}
        </p>
      )}

      {(loading || loadingMore) && (
        <p className="text-center text-sm text-zinc-500 dark:text-zinc-400">
          {t("entries.loading")}
        </p>
      )}

      <div ref={sentinelRef} className="h-1" />
    </section>
  );
}
