import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { useAuth } from "../components/AuthProvider";

type Account = components["schemas"]["Account"];
type Entry = components["schemas"]["Entry"];
type Role = "sender" | "receiver";

export const Route = createFileRoute("/entries/$entryId/self-transfer")({
  component: ConvertToSelfTransfer,
});

const inputClass =
  "rounded-md border border-black/15 bg-transparent px-3 py-2 text-sm font-normal outline-none transition-colors focus:border-black/40 dark:border-white/15 dark:focus:border-white/40";

function ConvertToSelfTransfer() {
  const { entryId } = Route.useParams();
  const { t } = useTranslation();
  const { user } = useAuth();
  const navigate = useNavigate();

  const [entry, setEntry] = useState<Entry | null | undefined>(undefined);
  const [account, setAccount] = useState<Account | null>(null);
  const [accounts, setAccounts] = useState<Account[]>([]);

  const [toAccountId, setToAccountId] = useState("");
  const [role, setRole] = useState<Role | "">("");

  const [invalidField, setInvalidField] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      api.GET("/api/entries/{id}", { params: { path: { id: entryId } } }),
      api.GET("/api/accounts"),
    ]).then(([entryRes, accountsRes]) => {
      if (cancelled) return;
      setAccounts(accountsRes.data ?? []);
      const e = entryRes.data ?? null;
      setEntry(e);
      if (e) {
        api
          .GET("/api/accounts/{id}", { params: { path: { id: e.account_id } } })
          .then(({ data }) => {
            if (!cancelled) setAccount(data ?? null);
          });
      }
    });
    return () => {
      cancelled = true;
    };
  }, [entryId]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!entry) return;
    if (!toAccountId) {
      setInvalidField("to_account_id");
      return;
    }
    if (role !== "sender" && role !== "receiver") {
      setInvalidField("role");
      return;
    }
    setInvalidField(null);
    setSubmitting(true);
    setError(null);

    const { data, response } = await api.POST(
      "/api/entries/{id}/self-transfer",
      {
        params: { path: { id: entryId } },
        body: { to_account_id: toAccountId, original_account_role: role },
      },
    );
    setSubmitting(false);
    if (!response.ok || !data) {
      setError(t("entries.convertSelfTransfer.saveError"));
      return;
    }
    navigate({ to: "/entries", search: { last: true } });
  }

  if (entry === undefined) {
    return null;
  }
  if (entry === null) {
    return (
      <section className="mx-auto flex w-full max-w-xl flex-col gap-4 px-6 py-12 sm:px-10">
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("entries.notFound")}
        </p>
      </section>
    );
  }

  // Mirrors entries.$entryId.edit.tsx's fullTierAllows — entry_admin/owner
  // may act on any entry on the account; append may act only on one they
  // themselves created.
  const createdBy = entry.created_by;
  function fullTierAllows(acc: Account | null): boolean {
    return (
      acc !== null &&
      (acc.permission === "entry_admin" ||
        acc.permission === "owner" ||
        (acc.permission === "append" && createdBy === user?.id))
    );
  }
  const canConvert = entry.kind === "transaction" && fullTierAllows(account);

  if (!canConvert) {
    return (
      <section className="mx-auto flex w-full max-w-xl flex-col gap-4 px-6 py-12 sm:px-10">
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("entries.convertSelfTransfer.notAvailable")}
        </p>
        <Link
          to="/entries/$entryId/edit"
          params={{ entryId }}
          className="text-sm font-medium underline-offset-2 hover:underline"
        >
          {t("entries.convertSelfTransfer.backToEntry")}
        </Link>
      </section>
    );
  }

  // The counterparty needs the same append+ permission as a self-transfer's
  // Create requires, must not be disabled, must share this entry's
  // account's currency, and can't be the entry's own account — mirrors
  // entries.new.tsx's toAccountOptions.
  const toAccountOptions = accounts.filter(
    (a) =>
      a.permission !== "view" &&
      !a.disabled &&
      a.id !== entry.account_id &&
      (account === null || a.currency === account.currency),
  );

  return (
    <section className="mx-auto flex w-full max-w-xl flex-col gap-8 px-6 py-12 sm:px-10">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("entries.convertSelfTransfer.title")}
        </h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("entries.convertSelfTransfer.description", {
            title: entry.title,
            account: account?.title ?? entry.account_id,
          })}
        </p>
      </div>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <label className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.convertSelfTransfer.toAccount")}
          <select
            value={toAccountId}
            onChange={(e) => setToAccountId(e.target.value)}
            className={inputClass}
            required
          >
            <option value="" disabled>
              {t("entries.convertSelfTransfer.toAccountPlaceholder")}
            </option>
            {toAccountOptions.map((a) => (
              <option key={a.id} value={a.id}>
                {a.title} ({a.currency})
              </option>
            ))}
          </select>
          {invalidField === "to_account_id" && (
            <span className="text-xs font-normal text-red-600 dark:text-red-400">
              {t("entries.convertSelfTransfer.toAccountRequired")}
            </span>
          )}
        </label>

        <fieldset className="flex flex-col gap-1.5 text-sm font-medium">
          {t("entries.convertSelfTransfer.role")}
          <div className="flex flex-col gap-1.5 text-sm font-normal">
            <label className="flex items-center gap-1.5">
              <input
                type="radio"
                name="role"
                checked={role === "sender"}
                onChange={() => setRole("sender")}
              />
              {t("entries.convertSelfTransfer.roleSender", {
                account: account?.title ?? entry.account_id,
              })}
            </label>
            <label className="flex items-center gap-1.5">
              <input
                type="radio"
                name="role"
                checked={role === "receiver"}
                onChange={() => setRole("receiver")}
              />
              {t("entries.convertSelfTransfer.roleReceiver", {
                account: account?.title ?? entry.account_id,
              })}
            </label>
          </div>
          {invalidField === "role" && (
            <span className="text-xs font-normal text-red-600 dark:text-red-400">
              {t("entries.convertSelfTransfer.roleRequired")}
            </span>
          )}
        </fieldset>

        {error && (
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        )}

        <div className="flex items-center gap-2">
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 disabled:opacity-60 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
          >
            {t("entries.convertSelfTransfer.submit")}
          </button>
          <Link
            to="/entries/$entryId/edit"
            params={{ entryId }}
            className="rounded-md px-3 py-2 text-sm font-medium text-zinc-600 hover:bg-black/[.04] dark:text-zinc-400 dark:hover:bg-white/[.06]"
          >
            {t("entries.convertSelfTransfer.cancel")}
          </Link>
        </div>
      </form>
    </section>
  );
}
