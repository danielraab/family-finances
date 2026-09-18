import { createFileRoute, Link } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { AccountLabel } from "../components/AccountLabel";
import { useAuth } from "../components/AuthProvider";
import { CategoryLabel } from "../components/CategoryLabel";
import { DateRangeFilter } from "../components/DateRangeFilter";
import { BulkActionToolbar } from "../components/entries/BulkActionToolbar";
import { LocationPreviewModal } from "../components/LocationPreviewModal";
import { RecurringTransactionBadge } from "../components/RecurringTransactionBadge";
import { SelfTransferBadge } from "../components/SelfTransferBadge";
import { TagLabel } from "../components/TagLabel";
import { UpcomingBlock } from "../components/UpcomingBlock";
import {
  amountColorClass,
  formatAmount,
  formatSignedAmount,
} from "../lib/amount";
import { flattenCategoryTree } from "../lib/categoryTree";
import { compact } from "../lib/compact";
import { resolveEffectiveRange } from "../lib/dateRangePresets";
import { type Coordinates, parseLocation } from "../lib/location";
import {
  fetchRecurringPreview,
  type RecurringTransactionPreviewItem,
  resolvePreviewCutoff,
  todayDateString,
} from "../lib/recurringPreview";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";
import { useRecurringPreviewHorizon } from "../lib/useRecurringPreviewHorizon";
import { useWeekStart } from "../lib/useWeekStart";

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
  range?: string | undefined;
  from?: string | undefined;
  to?: string | undefined;
  q?: string | undefined;
  sort?: Sort | undefined;
  dir?: Dir | undefined;
  show_recurring?: boolean | undefined;
  // Not a real filter — a one-shot navigation instruction (see
  // "returning to the entries list" below). Never combined with another
  // filter param by anything in this app; if it somehow is, it's ignored
  // and the explicit filter(s) apply as usual.
  last?: boolean | undefined;
};

const PAGE_SIZE = 30;
const DEFAULT_RANGE_PRESET = "last_2_weeks" as const;

// The most recently applied filter/search/sort state, persisted per-browser
// so a visitor returning via /entries?last=true (after saving an edit or
// completing a self-transfer conversion) lands back where they were,
// instead of the account_id-only redirect this used to be. See
// web-client-entries' "Returning to the entry ledger restores the
// last-applied filters" requirement.
const LAST_FILTERS_KEY = "ff:entries-last-filters";

/** Every EntriesSearch field except `last` itself. */
type PersistedFilters = Omit<EntriesSearch, "last">;

function hasFilterParams(search: EntriesSearch): boolean {
  return (
    search.account_id !== undefined ||
    search.category_id !== undefined ||
    search.tag_id !== undefined ||
    search.kind !== undefined ||
    search.range !== undefined ||
    search.from !== undefined ||
    search.to !== undefined ||
    search.q !== undefined ||
    search.sort !== undefined ||
    search.dir !== undefined ||
    search.show_recurring !== undefined
  );
}

/** True only for the transient "resolve ?last=true" tick — last present
 * and nothing else, the one case this page redirects on rather than
 * rendering/persisting as-is. */
function isPendingLastResolution(search: EntriesSearch): boolean {
  return Boolean(search.last) && !hasFilterParams(search);
}

function loadPersistedFilters(): PersistedFilters {
  try {
    const raw = localStorage.getItem(LAST_FILTERS_KEY);
    if (!raw) return {};
    return JSON.parse(raw) as PersistedFilters;
  } catch {
    return {};
  }
}

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
    range: asString(search["range"]),
    from: asString(search["from"]),
    to: asString(search["to"]),
    q: asString(search["q"]),
    sort: search["sort"] === "amount" ? "amount" : undefined,
    dir: search["dir"] === "asc" ? "asc" : undefined,
    show_recurring:
      typeof search["show_recurring"] === "boolean"
        ? search["show_recurring"]
        : undefined,
    last: search["last"] === true ? true : undefined,
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
  const { user } = useAuth();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();
  const weekStart = useWeekStart();
  const recurringPreviewHorizon = useRecurringPreviewHorizon();
  const today = new Date();
  const effectiveRange = resolveEffectiveRange(
    { range: search.range, from: search.from, to: search.to },
    weekStart,
    DEFAULT_RANGE_PRESET,
    today,
  );
  const previewCutoff = resolvePreviewCutoff(
    effectiveRange.to,
    recurringPreviewHorizon,
    today,
  );
  // A cutoff already before today has nothing to preview — the toggle can
  // be on, but the block simply doesn't render (see design.md).
  const showUpcoming =
    (search.show_recurring ?? false) && previewCutoff >= todayDateString(today);

  const [accounts, setAccounts] = useState<Account[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [qDraft, setQDraft] = useState(search.q ?? "");

  const [items, setItems] = useState<Entry[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [previewItems, setPreviewItems] = useState<
    RecurringTransactionPreviewItem[] | null
  >(null);
  const [previewLocation, setPreviewLocation] = useState<Coordinates | null>(
    null,
  );
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

  // Resolves a bare ?last=true (no other filter param) to the persisted
  // filter state — or a bare /entries if nothing has been persisted yet —
  // replacing the URL so the ?last=true hop doesn't linger in browser
  // history. last alongside any other filter param is left alone; the
  // explicit filters apply as usual (see isPendingLastResolution).
  useEffect(() => {
    if (!isPendingLastResolution(search)) return;
    navigate({ search: loadPersistedFilters(), replace: true });
  }, [search, navigate]);

  // Persists the current filter/search/sort state on every real change, so
  // a later /entries?last=true can restore it. Skipped on the transient
  // "resolving ?last=true" tick above, so that redirect's own (empty, or
  // stale) search never overwrites what was actually persisted.
  useEffect(() => {
    if (isPendingLastResolution(search)) return;
    const { last: _last, ...persisted } = search;
    try {
      localStorage.setItem(LAST_FILTERS_KEY, JSON.stringify(persisted));
    } catch {
      // Unavailable (private browsing, blocked storage, …) — filters
      // simply won't be restored next time; nothing else depends on this.
    }
  }, [search]);

  const searchKey = JSON.stringify({ ...search, weekStart });

  function buildQuery(after?: string) {
    return compact({
      account_id: search.account_id ? [search.account_id] : undefined,
      category_id: search.category_id,
      tag_id: search.tag_id,
      kind: search.kind,
      from: effectiveRange.from ? toRangeStart(effectiveRange.from) : undefined,
      to: effectiveRange.to ? toRangeEnd(effectiveRange.to) : undefined,
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
    setSelectedIds(new Set());
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

  const previewKey = JSON.stringify({
    accountId: search.account_id,
    categoryId: search.category_id,
    tagId: search.tag_id,
    cutoff: previewCutoff,
    showUpcoming,
  });

  // biome-ignore lint/correctness/useExhaustiveDependencies: previewKey is the stable dependency for the values it embeds.
  useEffect(() => {
    if (!showUpcoming) {
      setPreviewItems(null);
      return;
    }
    let cancelled = false;
    setPreviewItems(null);
    fetchRecurringPreview({
      accountIds: search.account_id ? [search.account_id] : undefined,
      categoryId: search.category_id,
      tagId: search.tag_id,
      to: previewCutoff,
    }).then(({ data }) => {
      if (!cancelled) setPreviewItems(data?.items ?? []);
    });
    return () => {
      cancelled = true;
    };
  }, [previewKey]);

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

  // Reloads the ledger's first page under the current filters without
  // touching the URL — used by the bulk-action run modal's "Reload list",
  // since the applied changes may have moved entries out of the active
  // filter (e.g. a bulk recategorize while filtered by category). Also
  // re-fetches tags: "Add tags"/"Set tags" can create a brand-new tag via
  // resolveTagIds, and without this the toolbar's stale `tags` list would
  // neither offer it as a suggestion nor recognize it as already existing
  // on a follow-up bulk action, creating a duplicate instead of reusing it.
  async function reloadAfterBulkAction() {
    setLoading(true);
    const [{ data }, { data: tg }] = await Promise.all([
      api.GET("/api/entries", { params: { query: buildQuery() } }),
      api.GET("/api/tags"),
    ]);
    setItems(data?.items ?? []);
    setNextCursor(data?.next_cursor ?? null);
    setTags(tg ?? []);
    setLoading(false);
    setSelectedIds(new Set());
  }

  function toggleSelectAll() {
    setSelectedIds((prev) =>
      items.length > 0 && items.every((e) => prev.has(e.id))
        ? new Set()
        : new Set(items.map((e) => e.id)),
    );
  }

  function toggleSelectOne(id: string) {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
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
  const categoryById = new Map(categories.map((c) => [c.id, c]));
  const tagById = new Map(tags.map((tg) => [tg.id, tg]));
  return (
    <section className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-6 py-12 sm:px-10">
      <header className="flex items-center justify-between gap-4">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("entries.title")}
        </h1>
        <div className="flex items-center gap-3">
          <Link
            to="/entries/import"
            search={search.account_id ? { account_id: search.account_id } : {}}
            className="rounded-md border border-black/15 px-3 py-2 text-sm font-medium transition-colors hover:bg-black/[.04] dark:border-white/15 dark:hover:bg-white/[.06]"
          >
            {t("entries.import.linkLabel")}
          </Link>
          <Link
            to="/entries/new"
            search={search.account_id ? { account_id: search.account_id } : {}}
            className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
          >
            {t("entries.create")}
          </Link>
        </div>
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
                {c.shared &&
                  ` — ${t("categories.shared.badgeTitle", { owner: c.ownerName ?? "" })}`}
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
                {tag.shared &&
                  ` — ${t("tags.shared.badgeTitle", { owner: tag.owner_name ?? "" })}`}
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

        <DateRangeFilter
          value={{ range: search.range, from: search.from, to: search.to }}
          weekStart={weekStart}
          defaultPreset={DEFAULT_RANGE_PRESET}
          onChange={(patch) => patchSearch(patch)}
          fromLabel={t("entries.filters.from")}
          toLabel={t("entries.filters.to")}
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

      {showUpcoming && (
        <UpcomingBlock
          items={previewItems}
          accounts={accounts}
          displayedDecimalPlaces={displayedDecimalPlaces}
          locale={i18n.resolvedLanguage ?? "en"}
        />
      )}

      {selectedIds.size > 0 && (
        <BulkActionToolbar
          items={items}
          selectedIds={selectedIds}
          onClear={() => setSelectedIds(new Set())}
          categories={categories}
          tags={tags}
          onReload={reloadAfterBulkAction}
        />
      )}

      <div className="overflow-x-auto rounded-lg border border-black/10 dark:border-white/10">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-black/10 text-xs uppercase text-zinc-500 dark:border-white/10 dark:text-zinc-400">
            <tr>
              <th className="w-8 px-3 py-2">
                <input
                  type="checkbox"
                  checked={
                    items.length > 0 &&
                    items.every((e) => selectedIds.has(e.id))
                  }
                  onChange={toggleSelectAll}
                  aria-label={t("entries.bulk.selectAll")}
                />
              </th>
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
                {t("entries.columns.account")}
              </th>
              <th className="px-3 py-2 font-medium">
                {t("entries.columns.title")}
              </th>
              <th className="px-3 py-2 font-medium">
                {t("entries.columns.category")}
              </th>
              <th className="px-3 py-2 font-medium">
                {t("entries.columns.tags")}
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
              // A self-transfer entry can appear twice in this list — once
              // per account it touches, when both are in scope (see
              // account-entries) — sharing the same id but a different
              // account_id, so the key needs both to stay unique.
              <tr
                key={`${entry.id}-${entry.account_id}`}
                className="border-b border-black/5 last:border-0 dark:border-white/5"
              >
                <td className="px-3 py-2">
                  <input
                    type="checkbox"
                    checked={selectedIds.has(entry.id)}
                    onChange={() => toggleSelectOne(entry.id)}
                    aria-label={t("entries.bulk.selectEntry", {
                      title: entry.title,
                    })}
                  />
                </td>
                <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                  {(() => {
                    const bookedAt = new Date(entry.booking_timestamp);
                    const lang = i18n.resolvedLanguage ?? "en";
                    return (
                      <div className="flex flex-col leading-tight">
                        <span className="whitespace-nowrap">
                          {bookedAt.toLocaleDateString(lang, {
                            weekday: "short",
                            year: "numeric",
                            month: "numeric",
                            day: "numeric",
                          })}
                        </span>
                        <span className="whitespace-nowrap text-xs">
                          {bookedAt.toLocaleTimeString(lang, {
                            hour: "2-digit",
                            minute: "2-digit",
                          })}
                        </span>
                      </div>
                    );
                  })()}
                </td>
                <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                  {(() => {
                    const account = accounts.find(
                      (a) => a.id === entry.account_id,
                    );
                    return account ? (
                      <AccountLabel account={account} iconSize={16} />
                    ) : (
                      <span className="italic">{t("entries.notShared")}</span>
                    );
                  })()}
                </td>
                <td className="px-3 py-2">
                  <Link
                    to="/entries/$entryId/edit"
                    params={{ entryId: entry.id }}
                    className="font-medium underline-offset-2 hover:underline"
                  >
                    {entry.title}
                  </Link>
                  {(() => {
                    const coords = parseLocation(entry.location);
                    if (!coords) return null;
                    return (
                      <button
                        type="button"
                        onClick={() => setPreviewLocation(coords)}
                        aria-label={t("entries.viewLocation")}
                        title={t("entries.viewLocation")}
                        className="ml-1.5 align-middle text-sm text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200"
                      >
                        🌐
                      </button>
                    );
                  })()}
                  <RecurringTransactionBadge
                    recurringTransactionId={entry.recurring_transaction_id}
                  />
                  <SelfTransferBadge entryId={entry.id} kind={entry.kind} />
                  {entry.counterparty && (
                    <span className="block text-xs text-zinc-500 dark:text-zinc-400">
                      {entry.counterparty}
                    </span>
                  )}
                  {entry.created_by !== user?.id && entry.created_by_name && (
                    <span className="block text-xs text-zinc-500 dark:text-zinc-400">
                      {t("entries.createdBy", {
                        name: entry.created_by_name,
                      })}
                    </span>
                  )}
                </td>
                <td className="px-3 py-2 text-zinc-500 dark:text-zinc-400">
                  {(() => {
                    if (!entry.category_id) {
                      return <span aria-hidden>—</span>;
                    }
                    const category = categoryById.get(entry.category_id);
                    return category ? (
                      <CategoryLabel category={category} iconSize={16} />
                    ) : (
                      <span className="italic">{t("entries.notShared")}</span>
                    );
                  })()}
                </td>
                <td className="px-3 py-2">
                  {(() => {
                    if (entry.tag_ids.length === 0) {
                      return (
                        <span
                          aria-hidden
                          className="text-zinc-500 dark:text-zinc-400"
                        >
                          —
                        </span>
                      );
                    }
                    const known = entry.tag_ids
                      .map((tagId) => tagById.get(tagId))
                      .filter((tag) => tag !== undefined);
                    const hasUnknown = known.length < entry.tag_ids.length;
                    return (
                      <div className="flex flex-wrap gap-1">
                        {known.map((tag) => (
                          <TagLabel
                            key={tag.id}
                            tag={tag}
                            className="rounded-full bg-black/[.06] px-2 py-0.5 text-xs font-medium dark:bg-white/10"
                          />
                        ))}
                        {hasUnknown && (
                          <span className="rounded-full bg-black/[.06] px-2 py-0.5 text-xs font-medium italic text-zinc-500 dark:bg-white/10 dark:text-zinc-400">
                            {t("entries.notShared")}
                          </span>
                        )}
                      </div>
                    );
                  })()}
                </td>
                <td className="px-3 py-2 text-right">
                  <div className="flex flex-col items-end gap-0.5">
                    <span
                      className={`font-mono tabular-nums ${amountColorClass(
                        entry.kind === "balance_adjustment"
                          ? (entry.balance ?? 0)
                          : entry.amount,
                      )} ${entry.kind === "balance_adjustment" ? "underline" : ""}`}
                    >
                      {formatAmount(
                        entry.kind === "balance_adjustment"
                          ? (entry.balance ?? 0)
                          : entry.amount,
                        entry.account_currency ?? "",
                        displayedDecimalPlaces,
                        i18n.resolvedLanguage ?? "en",
                      )}
                    </span>
                    {entry.kind === "balance_adjustment" && (
                      <span className="font-mono text-xs tabular-nums text-zinc-500 dark:text-zinc-400">
                        {formatSignedAmount(
                          entry.amount,
                          entry.account_currency ?? "",
                          displayedDecimalPlaces,
                          i18n.resolvedLanguage ?? "en",
                        )}
                      </span>
                    )}
                  </div>
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
          effectiveRange.from ||
          effectiveRange.to ||
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

      <LocationPreviewModal
        open={previewLocation !== null}
        onClose={() => setPreviewLocation(null)}
        coordinates={previewLocation}
      />
    </section>
  );
}
