import { createFileRoute, Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { AccountLabel } from "../components/AccountLabel";
import { amountColorClass, formatAmount } from "../lib/amount";
import type { Account } from "../lib/useAccountsWithBalances";
import { useAccountsWithBalances } from "../lib/useAccountsWithBalances";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";

export const Route = createFileRoute("/accounts/")({
  component: AccountsOverview,
});

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
      <span className="inline-flex shrink-0 items-center rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-800 dark:bg-amber-900/40 dark:text-amber-300">
        {t("accounts.status.disabled")}
      </span>
    );
  }
  if (closed) {
    return (
      <span className="inline-flex shrink-0 items-center rounded-full bg-black/[.06] px-2 py-0.5 text-xs font-medium text-zinc-600 dark:bg-white/10 dark:text-zinc-400">
        {t("accounts.status.closed")}
      </span>
    );
  }
  return (
    <span className="inline-flex shrink-0 items-center rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300">
      {t("accounts.status.open")}
    </span>
  );
}

function PlusGlyph() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={18}
      height={18}
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M12 5v14" />
      <path d="M5 12h14" />
    </svg>
  );
}

function AccountsOverview() {
  const { t, i18n } = useTranslation();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();
  const { accounts, types, balances } = useAccountsWithBalances();

  const typeName = (typeId: string) =>
    types.find((type) => type.id === typeId)?.title ?? typeId;

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
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {accounts.map((account) => (
            <div
              key={account.id}
              className="flex flex-col gap-3 rounded-lg border border-black/10 p-4 dark:border-white/10"
            >
              <Link
                to="/accounts/$accountId"
                params={{ accountId: account.id }}
                className="flex flex-1 flex-col gap-1 transition-opacity hover:opacity-80"
              >
                <div className="flex items-start justify-between gap-2">
                  <AccountLabel
                    account={account}
                    className="min-w-0 font-medium"
                  />
                  <AccountStatus account={account} t={t} />
                </div>
                <span className="text-xs text-zinc-500 dark:text-zinc-400">
                  {typeName(account.type_id)} · {account.currency}
                </span>
                <span
                  className={`mt-2 font-mono text-lg tabular-nums ${
                    account.id in balances
                      ? amountColorClass(balances[account.id] ?? 0)
                      : ""
                  }`}
                >
                  {account.id in balances
                    ? formatAmount(
                        balances[account.id] ?? 0,
                        account.currency,
                        displayedDecimalPlaces,
                        i18n.resolvedLanguage ?? "en",
                      )
                    : "…"}
                </span>
              </Link>
              <Link
                to="/entries/new"
                search={{ account_id: account.id }}
                className="flex items-center gap-1.5 self-end rounded-md border border-black/10 px-2 py-1 text-sm text-zinc-500 transition-colors hover:bg-black/[.06] hover:text-zinc-900 dark:border-white/10 dark:text-zinc-400 dark:hover:bg-white/[.08] dark:hover:text-zinc-100"
              >
                <PlusGlyph />
                {t("entries.create")}
              </Link>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
