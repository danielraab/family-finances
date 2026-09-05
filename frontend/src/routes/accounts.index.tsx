import { createFileRoute, Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { formatAmount } from "../lib/amount";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";

export const Route = createFileRoute("/accounts/")({
  component: AccountsOverview,
});

type Account = components["schemas"]["Account"];
type AccountType = components["schemas"]["AccountType"];

function AccountStatus({
  account,
  t,
}: {
  account: Account;
  t: (key: string) => string;
}) {
  const closed =
    account.closing_date !== undefined &&
    new Date(account.closing_date) <= new Date();
  if (account.disabled) {
    return (
      <span className="inline-flex items-center rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-800 dark:bg-amber-900/40 dark:text-amber-300">
        {t("accounts.status.disabled")}
      </span>
    );
  }
  if (closed) {
    return (
      <span className="inline-flex items-center rounded-full bg-black/[.06] px-2 py-0.5 text-xs font-medium text-zinc-600 dark:bg-white/10 dark:text-zinc-400">
        {t("accounts.status.closed")}
      </span>
    );
  }
  return (
    <span className="inline-flex items-center rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300">
      {t("accounts.status.open")}
    </span>
  );
}

function AccountsOverview() {
  const { t, i18n } = useTranslation();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();
  const [accounts, setAccounts] = useState<Account[] | null>(null);
  const [types, setTypes] = useState<AccountType[]>([]);
  const [balances, setBalances] = useState<Record<string, number>>({});

  useEffect(() => {
    let cancelled = false;
    Promise.all([api.GET("/api/accounts"), api.GET("/api/account-types")]).then(
      ([accountsRes, typesRes]) => {
        if (cancelled) return;
        setAccounts(accountsRes.data ?? []);
        setTypes(typesRes.data ?? []);
      },
    );
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!accounts) return;
    let cancelled = false;
    Promise.all(
      accounts.map((account) =>
        api
          .GET("/api/accounts/{id}/balance", {
            params: { path: { id: account.id } },
          })
          .then(({ data }) => [account.id, data?.balance ?? 0] as const),
      ),
    ).then((results) => {
      if (cancelled) return;
      setBalances(Object.fromEntries(results));
    });
    return () => {
      cancelled = true;
    };
  }, [accounts]);

  const typeName = (typeId: string) =>
    types.find((type) => type.id === typeId)?.name ?? typeId;

  return (
    <section className="mx-auto flex w-full max-w-4xl flex-col gap-6 px-6 py-12 sm:px-10">
      <header className="flex items-center justify-between gap-4">
        <h1 className="text-2xl font-semibold tracking-tight">
          {t("accounts.title")}
        </h1>
        <Link
          to="/accounts/new"
          className="rounded-md bg-zinc-900 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-zinc-700 dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-200"
        >
          {t("accounts.create")}
        </Link>
      </header>

      {accounts === null ? null : accounts.length === 0 ? (
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("accounts.empty")}
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {accounts.map((account) => (
            <li key={account.id}>
              <Link
                to="/accounts/$accountId"
                params={{ accountId: account.id }}
                className="flex items-center justify-between gap-4 rounded-lg border border-black/10 px-4 py-3 transition-colors hover:bg-black/[.02] dark:border-white/10 dark:hover:bg-white/[.04]"
              >
                <div className="flex flex-col gap-0.5">
                  <span className="font-medium">{account.title}</span>
                  <span className="text-xs text-zinc-500 dark:text-zinc-400">
                    {typeName(account.type_id)} · {account.currency}
                  </span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="font-mono text-sm tabular-nums">
                    {account.id in balances
                      ? formatAmount(
                          balances[account.id] ?? 0,
                          account.currency,
                          displayedDecimalPlaces,
                          i18n.resolvedLanguage ?? "en",
                        )
                      : "…"}
                  </span>
                  <AccountStatus account={account} t={t} />
                </div>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
