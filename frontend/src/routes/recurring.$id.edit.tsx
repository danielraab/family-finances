import {
  Description,
  Dialog,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { CategoryLabel } from "../components/CategoryLabel";
import {
  RecurringTransactionForm,
  type RecurringTransactionFormValues,
} from "../components/RecurringTransactionForm";
import { amountColorClass, amountToInput, formatAmount } from "../lib/amount";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";

type Category = components["schemas"]["Category"];
type Entry = components["schemas"]["Entry"];

// Mirrors entries.index.tsx's own ledger page size — this is a secondary
// section, not the primary ledger, but there's no reason for its first
// page to be a different size.
const PAGE_SIZE = 30;

export const Route = createFileRoute("/recurring/$id/edit")({
  component: EditRecurringTransaction,
});

function EditRecurringTransaction() {
  const { id } = Route.useParams();
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();

  const [values, setValues] = useState<RecurringTransactionFormValues | null>(
    null,
  );
  const [linkedEntryCount, setLinkedEntryCount] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const [categories, setCategories] = useState<Category[]>([]);
  const [linkedItems, setLinkedItems] = useState<Entry[]>([]);
  const [linkedNextCursor, setLinkedNextCursor] = useState<string | null>(
    null,
  );
  const [linkedLoading, setLinkedLoading] = useState(false);
  const [linkedLoadingMore, setLinkedLoadingMore] = useState(false);

  useEffect(() => {
    let cancelled = false;
    api
      .GET("/api/recurring-transactions/{id}", { params: { path: { id } } })
      .then(({ data }) => {
        if (cancelled || !data) return;
        setLinkedEntryCount(data.linked_entry_count);
        setValues({
          account_id: data.account_id,
          title: data.title,
          description: data.description ?? "",
          category_id: data.category_id,
          counterparty: data.counterparty ?? "",
          location: data.location ?? "",
          tagNames: [],
          amountMagnitude: amountToInput(Math.abs(data.amount)),
          negative: data.amount < 0,
          interval_unit: data.interval_unit,
          interval_count: String(data.interval_count),
          starts_on: data.starts_on,
          ends_on: data.ends_on ?? "",
        });
        if (data.tag_ids.length > 0) {
          api.GET("/api/tags").then(({ data: tags }) => {
            if (cancelled) return;
            const names = data.tag_ids
              .map((tagId) => tags?.find((tg) => tg.id === tagId)?.name)
              .filter((name): name is string => Boolean(name));
            setValues((prev) => (prev ? { ...prev, tagNames: names } : prev));
          });
        }
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  // Fetched only once linked_entry_count resolves to something greater
  // than 0 — the template's own response already carries that count, so
  // this skips an empty-result round trip for the common case of a
  // freshly created template with nothing linked yet.
  useEffect(() => {
    if (linkedEntryCount <= 0) return;
    let cancelled = false;
    setLinkedLoading(true);
    Promise.all([
      api.GET("/api/entries", {
        params: {
          query: { recurring_transaction_id: id, limit: PAGE_SIZE },
        },
      }),
      api.GET("/api/categories"),
    ]).then(([entriesRes, categoriesRes]) => {
      if (cancelled) return;
      setLinkedItems(entriesRes.data?.items ?? []);
      setLinkedNextCursor(entriesRes.data?.next_cursor ?? null);
      setCategories(categoriesRes.data ?? []);
      setLinkedLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, [linkedEntryCount, id]);

  async function loadMoreLinked() {
    if (linkedLoadingMore || !linkedNextCursor) return;
    setLinkedLoadingMore(true);
    const { data } = await api.GET("/api/entries", {
      params: {
        query: {
          recurring_transaction_id: id,
          limit: PAGE_SIZE,
          after: linkedNextCursor,
        },
      },
    });
    setLinkedItems((prev) => [...prev, ...(data?.items ?? [])]);
    setLinkedNextCursor(data?.next_cursor ?? null);
    setLinkedLoadingMore(false);
  }

  async function handleDelete() {
    setConfirmingDelete(false);
    setDeleteError(null);
    const { response } = await api.DELETE("/api/recurring-transactions/{id}", {
      params: { path: { id } },
    });
    if (response.ok) {
      navigate({ to: "/recurring" });
      return;
    }
    setDeleteError(t("recurring.edit.deleteError"));
  }

  if (!values) {
    return null;
  }

  return (
    <section className="mx-auto flex w-full max-w-xl flex-col gap-8 px-6 py-12 sm:px-10">
      <h1 className="text-2xl font-semibold tracking-tight">
        {t("recurring.edit.title")}
      </h1>

      <RecurringTransactionForm
        initial={values}
        submitLabel={t("recurring.form.save")}
        submitting={submitting}
        serverError={error}
        onSubmit={async (body) => {
          setSubmitting(true);
          setError(null);
          const { data, response } = await api.PATCH(
            "/api/recurring-transactions/{id}",
            {
              params: { path: { id } },
              body: {
                account_id: body.account_id,
                title: body.title,
                description: body.description,
                category_id: body.category_id,
                counterparty: body.counterparty,
                location: body.location,
                tag_ids: body.tag_ids,
                amount: body.amount,
                interval_unit: body.interval_unit,
                interval_count: body.interval_count,
                starts_on: body.starts_on,
                ends_on: body.ends_on || null,
              },
            },
          );
          setSubmitting(false);
          if (!response.ok || !data) {
            setError(t("recurring.form.saveError"));
            return;
          }
          navigate({ to: "/recurring" });
        }}
      />

      <section className="flex flex-col gap-3 border-t border-black/10 pt-6 dark:border-white/10">
        <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
          {t("recurring.edit.linkedTransactions.heading")}
        </h2>
        {linkedEntryCount === 0 ? (
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            {t("recurring.edit.linkedTransactions.empty")}
          </p>
        ) : linkedLoading ? (
          <p className="text-sm text-zinc-500 dark:text-zinc-400">…</p>
        ) : (
          <>
            <ul className="flex flex-col divide-y divide-black/5 dark:divide-white/5">
              {linkedItems.map((item) => {
                const category = categories.find(
                  (c) => c.id === item.category_id,
                );
                return (
                  <Link
                    key={item.id}
                    to="/entries/$entryId/edit"
                    params={{ entryId: item.id }}
                    className="flex items-center justify-between gap-2 rounded px-1 py-1.5 text-sm transition-colors hover:bg-black/[.04] dark:hover:bg-white/[.06]"
                  >
                    <div className="flex min-w-0 flex-col">
                      <span className="truncate font-medium">
                        {item.title}
                      </span>
                      <span className="flex items-center gap-1 text-xs text-zinc-500 dark:text-zinc-400">
                        {new Date(item.booking_timestamp).toLocaleDateString(
                          i18n.language,
                        )}
                        {category && (
                          <>
                            {" · "}
                            <CategoryLabel category={category} iconSize={14} />
                          </>
                        )}
                      </span>
                    </div>
                    <span
                      className={`shrink-0 font-mono text-sm tabular-nums ${amountColorClass(item.amount)}`}
                    >
                      {formatAmount(
                        item.amount,
                        item.account_currency ?? "",
                        displayedDecimalPlaces,
                        i18n.language,
                      )}
                    </span>
                  </Link>
                );
              })}
            </ul>
            {linkedNextCursor && (
              <div>
                <button
                  type="button"
                  onClick={loadMoreLinked}
                  disabled={linkedLoadingMore}
                  className="rounded-md border border-black/10 px-3 py-1.5 text-xs font-medium text-zinc-600 transition-colors hover:bg-black/[.04] disabled:opacity-60 dark:border-white/10 dark:text-zinc-400 dark:hover:bg-white/[.08]"
                >
                  {t("recurring.edit.linkedTransactions.loadMore")}
                </button>
              </div>
            )}
          </>
        )}
      </section>

      <section className="flex flex-col gap-3 border-t border-black/10 pt-6 dark:border-white/10">
        <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
          {t("recurring.edit.dangerZone")}
        </h2>
        <div>
          <button
            type="button"
            disabled={linkedEntryCount > 0}
            onClick={() => setConfirmingDelete(true)}
            title={
              linkedEntryCount > 0
                ? t("recurring.edit.deleteBlockedHint", {
                    count: linkedEntryCount,
                  })
                : undefined
            }
            className="rounded-md border border-red-200 px-3 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:bg-transparent dark:border-red-900/50 dark:text-red-400 dark:hover:bg-red-950/30"
          >
            {t("recurring.edit.delete")}
          </button>
          {linkedEntryCount > 0 && (
            <p className="mt-2 text-xs text-zinc-500 dark:text-zinc-400">
              {t("recurring.edit.deleteBlockedHint", {
                count: linkedEntryCount,
              })}
            </p>
          )}
          {deleteError && (
            <p className="mt-2 text-sm text-red-600 dark:text-red-400">
              {deleteError}
            </p>
          )}
        </div>
      </section>

      <Dialog
        open={confirmingDelete}
        onClose={() => setConfirmingDelete(false)}
        className="relative z-50"
      >
        <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="flex w-full max-w-sm flex-col gap-4 rounded-lg bg-white p-6 dark:bg-neutral-900">
            <DialogTitle className="text-base font-semibold">
              {t("recurring.edit.confirmDeleteTitle")}
            </DialogTitle>
            <Description className="text-sm text-zinc-600 dark:text-zinc-400">
              {t("recurring.edit.confirmDeleteBody")}
            </Description>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setConfirmingDelete(false)}
                className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
              >
                {t("accounts.edit.confirm.cancel")}
              </button>
              <button
                type="button"
                onClick={handleDelete}
                className="rounded-md bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700"
              >
                {t("recurring.edit.delete")}
              </button>
            </div>
          </DialogPanel>
        </div>
      </Dialog>
    </section>
  );
}
