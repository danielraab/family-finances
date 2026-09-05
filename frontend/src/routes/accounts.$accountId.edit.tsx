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
import { AccountForm, type AccountFormValues } from "../components/AccountForm";

export const Route = createFileRoute("/accounts/$accountId/edit")({
  component: EditAccount,
});

type ConfirmKind = "disable" | "enable" | "delete";

function EditAccount() {
  const { accountId } = Route.useParams();
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [values, setValues] = useState<AccountFormValues | null>(null);
  const [disabled, setDisabled] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirming, setConfirming] = useState<ConfirmKind | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .GET("/api/accounts/{id}", { params: { path: { id: accountId } } })
      .then(({ data }) => {
        if (cancelled || !data) return;
        setDisabled(data.disabled);
        setValues({
          title: data.title,
          description: data.description ?? "",
          type_id: data.type_id,
          currency: data.currency,
          financial_institute: data.financial_institute ?? "",
          opening_date: data.opening_date,
          closing_date: data.closing_date ?? "",
        });
      });
    return () => {
      cancelled = true;
    };
  }, [accountId]);

  async function performConfirmed() {
    const kind = confirming;
    setConfirming(null);
    if (!kind) return;

    if (kind === "disable") {
      const { data } = await api.POST("/api/accounts/{id}/disable", {
        params: { path: { id: accountId } },
      });
      if (data) setDisabled(data.disabled);
    } else if (kind === "enable") {
      const { data } = await api.POST("/api/accounts/{id}/enable", {
        params: { path: { id: accountId } },
      });
      if (data) setDisabled(data.disabled);
    } else {
      const { response } = await api.DELETE("/api/accounts/{id}", {
        params: { path: { id: accountId } },
      });
      if (response.ok) {
        navigate({ to: "/accounts" });
      }
    }
  }

  if (!values) {
    return null;
  }

  return (
    <section className="mx-auto flex w-full max-w-xl flex-col gap-8 px-6 py-12 sm:px-10">
      <h1 className="text-2xl font-semibold tracking-tight">
        {t("accounts.edit.title")}
      </h1>

      <AccountForm
        initial={values}
        submitLabel={t("accounts.form.save")}
        submitting={submitting}
        serverError={error}
        onSubmit={async (body) => {
          setSubmitting(true);
          setError(null);
          const { data, response } = await api.PATCH("/api/accounts/{id}", {
            params: { path: { id: accountId } },
            body,
          });
          setSubmitting(false);
          if (!response.ok || !data) {
            setError(t("accounts.form.saveError"));
            return;
          }
          navigate({ to: "/accounts/$accountId", params: { accountId } });
        }}
      />

      <section className="flex flex-col gap-3 border-t border-black/10 pt-6 dark:border-white/10">
        <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
          {t("accounts.edit.dangerZone")}
        </h2>
        <div className="flex flex-wrap gap-2">
          {disabled ? (
            <button
              type="button"
              onClick={() => setConfirming("enable")}
              className="rounded-md border border-black/15 px-3 py-2 text-sm font-medium transition-colors hover:bg-black/[.04] dark:border-white/15 dark:hover:bg-white/[.06]"
            >
              {t("accounts.edit.enable")}
            </button>
          ) : (
            <button
              type="button"
              onClick={() => setConfirming("disable")}
              className="rounded-md border border-black/15 px-3 py-2 text-sm font-medium transition-colors hover:bg-black/[.04] dark:border-white/15 dark:hover:bg-white/[.06]"
            >
              {t("accounts.edit.disable")}
            </button>
          )}
          <button
            type="button"
            onClick={() => setConfirming("delete")}
            className="rounded-md border border-red-200 px-3 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 dark:border-red-900/50 dark:text-red-400 dark:hover:bg-red-950/30"
          >
            {t("accounts.edit.delete")}
          </button>
        </div>
      </section>

      <Dialog
        open={confirming !== null}
        onClose={() => setConfirming(null)}
        className="relative z-50"
      >
        <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
        <div className="fixed inset-0 flex items-center justify-center p-4">
          <DialogPanel className="flex w-full max-w-sm flex-col gap-4 rounded-lg bg-white p-6 dark:bg-neutral-900">
            {confirming && (
              <>
                <DialogTitle className="text-base font-semibold">
                  {t(`accounts.edit.confirm.${confirming}Title`)}
                </DialogTitle>
                <Description className="text-sm text-zinc-600 dark:text-zinc-400">
                  {t(`accounts.edit.confirm.${confirming}Body`)}
                </Description>
                <div className="flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={() => setConfirming(null)}
                    className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
                  >
                    {t("accounts.edit.confirm.cancel")}
                  </button>
                  <button
                    type="button"
                    onClick={performConfirmed}
                    className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
                  >
                    {t("accounts.edit.confirm.confirmAction")}
                  </button>
                </div>
              </>
            )}
          </DialogPanel>
        </div>
      </Dialog>
    </section>
  );
}
