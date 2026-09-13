import {
  Description,
  Dialog,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import {
  RecurringTransactionForm,
  type RecurringTransactionFormValues,
} from "../components/RecurringTransactionForm";
import { amountToInput } from "../lib/amount";

export const Route = createFileRoute("/recurring/$id/edit")({
  component: EditRecurringTransaction,
});

function EditRecurringTransaction() {
  const { id } = Route.useParams();
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [values, setValues] = useState<RecurringTransactionFormValues | null>(
    null,
  );
  const [linkedEntryCount, setLinkedEntryCount] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

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
