import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import {
  emptyRecurringTransactionForm,
  RecurringTransactionForm,
  type RecurringTransactionFormValues,
} from "../components/RecurringTransactionForm";
import { amountToInput } from "../lib/amount";
import { compact } from "../lib/compact";

type Tag = components["schemas"]["Tag"];

type RecurringNewSearch = {
  from_entry_id?: string | undefined;
};

export const Route = createFileRoute("/recurring/new")({
  validateSearch: (search: Record<string, unknown>): RecurringNewSearch => ({
    from_entry_id:
      typeof search["from_entry_id"] === "string"
        ? search["from_entry_id"]
        : undefined,
  }),
  component: NewRecurringTransaction,
});

function todayLocalDate(): string {
  const d = new Date();
  const tzOffsetMs = d.getTimezoneOffset() * 60_000;
  return new Date(d.getTime() - tzOffsetMs).toISOString().slice(0, 10);
}

/** Same local-date conversion as todayLocalDate, but from a given ISO
 * timestamp rather than "now" — used to default starts_on to a prefilled
 * entry's own booking date. */
function localDateFromISO(iso: string): string {
  const d = new Date(iso);
  const tzOffsetMs = d.getTimezoneOffset() * 60_000;
  return new Date(d.getTime() - tzOffsetMs).toISOString().slice(0, 10);
}

function NewRecurringTransaction() {
  const { from_entry_id: fromEntryId } = Route.useSearch();
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loadingEntry, setLoadingEntry] = useState(fromEntryId != null);
  const [initialOverrides, setInitialOverrides] = useState<
    Partial<RecurringTransactionFormValues>
  >({});

  // Prefilling from an existing entry's "create recurring transaction"
  // action (see web-client-entries). Fetched once up front — and the form
  // held back until it resolves — because RecurringTransactionForm only
  // reads its `initial` prop once, at mount, so patching it in after the
  // form has already mounted with empty values would have no effect.
  useEffect(() => {
    if (!fromEntryId) return;
    let cancelled = false;
    Promise.all([
      api.GET("/api/entries/{id}", { params: { path: { id: fromEntryId } } }),
      api.GET("/api/tags"),
    ]).then(([entryRes, tagsRes]) => {
      if (cancelled) return;
      const entry = entryRes.data;
      const allTags: Tag[] = tagsRes.data ?? [];
      // A recurring transaction has no balance-adjustment equivalent, so a
      // non-transaction entry (shouldn't normally be reachable here — the
      // triggering action only ever appears on a transaction entry) is
      // left unprefilled rather than mapped onto nonsensical fields.
      if (entry && entry.kind === "transaction") {
        setInitialOverrides({
          account_id: entry.account_id,
          title: entry.title,
          description: entry.description ?? "",
          category_id: entry.category_id ?? "",
          counterparty: entry.counterparty ?? "",
          location: entry.location ?? "",
          tagNames: entry.tag_ids
            .map((id) => allTags.find((tag) => tag.id === id)?.name)
            .filter((name): name is string => Boolean(name)),
          amountMagnitude: amountToInput(Math.abs(entry.amount)),
          negative: entry.amount < 0,
          starts_on: localDateFromISO(entry.booking_timestamp),
        });
      }
      setLoadingEntry(false);
    });
    return () => {
      cancelled = true;
    };
  }, [fromEntryId]);

  if (loadingEntry) {
    return null;
  }

  return (
    <section className="mx-auto flex w-full max-w-xl flex-col gap-6 px-6 py-12 sm:px-10">
      <h1 className="text-2xl font-semibold tracking-tight">
        {t("recurring.new.title")}
      </h1>
      <RecurringTransactionForm
        initial={{
          ...emptyRecurringTransactionForm,
          starts_on: todayLocalDate(),
          ...initialOverrides,
        }}
        accountLocked={fromEntryId != null}
        submitLabel={t("recurring.form.create")}
        submitting={submitting}
        serverError={error}
        onSubmit={async (values) => {
          setSubmitting(true);
          setError(null);
          const { data, response } = await api.POST(
            "/api/recurring-transactions",
            {
              body: {
                account_id: values.account_id,
                kind: values.kind,
                title: values.title,
                amount: values.amount,
                interval_unit: values.interval_unit,
                interval_count: values.interval_count,
                starts_on: values.starts_on,
                ...compact({
                  // Omitted rather than sent empty for a self-transfer,
                  // which has no category requirement and rejects
                  // counterparty/location outright.
                  category_id: values.category_id || undefined,
                  to_account_id: values.to_account_id || undefined,
                  description: values.description || undefined,
                  counterparty: values.counterparty || undefined,
                  location: values.location || undefined,
                  ends_on: values.ends_on || undefined,
                  tag_ids:
                    values.tag_ids.length > 0 ? values.tag_ids : undefined,
                }),
              },
            },
          );
          if (!response.ok || !data) {
            setSubmitting(false);
            setError(t("recurring.form.saveError"));
            return;
          }
          if (fromEntryId) {
            // Always auto-linked, no opt-out: otherwise NextSuggestedDate
            // stays pinned at starts_on (this entry's own date), and
            // UpcomingBlock would show a phantom duplicate suggestion for
            // a date that's already booked. A failed link here doesn't
            // block navigation — the template itself was already created
            // successfully either way.
            await api.PATCH("/api/entries/{id}", {
              params: { path: { id: fromEntryId } },
              body: { recurring_transaction_id: data.id },
            });
          }
          setSubmitting(false);
          navigate({ to: "/recurring" });
        }}
      />
    </section>
  );
}
