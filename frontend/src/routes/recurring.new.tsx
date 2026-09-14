import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import {
  emptyRecurringTransactionForm,
  RecurringTransactionForm,
} from "../components/RecurringTransactionForm";
import { compact } from "../lib/compact";

export const Route = createFileRoute("/recurring/new")({
  component: NewRecurringTransaction,
});

function todayLocalDate(): string {
  const d = new Date();
  const tzOffsetMs = d.getTimezoneOffset() * 60_000;
  return new Date(d.getTime() - tzOffsetMs).toISOString().slice(0, 10);
}

function NewRecurringTransaction() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  return (
    <section className="mx-auto flex w-full max-w-xl flex-col gap-6 px-6 py-12 sm:px-10">
      <h1 className="text-2xl font-semibold tracking-tight">
        {t("recurring.new.title")}
      </h1>
      <RecurringTransactionForm
        initial={{
          ...emptyRecurringTransactionForm,
          starts_on: todayLocalDate(),
        }}
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
                title: values.title,
                category_id: values.category_id,
                amount: values.amount,
                interval_unit: values.interval_unit,
                interval_count: values.interval_count,
                starts_on: values.starts_on,
                ...compact({
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
          setSubmitting(false);
          if (!response.ok || !data) {
            setError(t("recurring.form.saveError"));
            return;
          }
          navigate({ to: "/recurring" });
        }}
      />
    </section>
  );
}
