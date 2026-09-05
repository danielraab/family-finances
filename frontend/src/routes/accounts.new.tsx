import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import { AccountForm, emptyAccountForm } from "../components/AccountForm";

export const Route = createFileRoute("/accounts/new")({
  component: NewAccount,
});

function NewAccount() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  return (
    <section className="mx-auto flex w-full max-w-xl flex-col gap-6 px-6 py-12 sm:px-10">
      <h1 className="text-2xl font-semibold tracking-tight">
        {t("accounts.new.title")}
      </h1>
      <AccountForm
        initial={emptyAccountForm}
        submitLabel={t("accounts.form.create")}
        submitting={submitting}
        serverError={error}
        onSubmit={async (values) => {
          setSubmitting(true);
          setError(null);
          const { data, response } = await api.POST("/api/accounts", {
            body: values,
          });
          setSubmitting(false);
          if (!response.ok || !data) {
            setError(t("accounts.form.saveError"));
            return;
          }
          navigate({
            to: "/accounts/$accountId",
            params: { accountId: data.id },
          });
        }}
      />
    </section>
  );
}
