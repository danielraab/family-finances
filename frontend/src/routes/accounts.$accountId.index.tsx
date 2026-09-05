import { createFileRoute, Link } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "../api/client";
import type { components } from "../api/schema";
import { formatAmount } from "../lib/amount";
import { useDisplayedDecimalPlaces } from "../lib/useDisplayedDecimalPlaces";

export const Route = createFileRoute("/accounts/$accountId/")({
  component: AccountDetails,
});

type Account = components["schemas"]["Account"];
type Entry = components["schemas"]["Entry"];

const RECENT_LIMIT = 5;

function AccountDetails() {
  const { accountId } = Route.useParams();
  const { t, i18n } = useTranslation();
  const displayedDecimalPlaces = useDisplayedDecimalPlaces();
  const [account, setAccount] = useState<Account | null | undefined>(undefined);
  const [balance, setBalance] = useState<number | null>(null);
  const [recent, setRecent] = useState<Entry[] | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .GET("/api/accounts/{id}", { params: { path: { id: accountId } } })
      .then(({ data }) => {
        if (!cancelled) setAccount(data ?? null);
      });
    api
      .GET("/api/accounts/{id}/balance", {
        params: { path: { id: accountId } },
      })
      .then(({ data }) => {
        if (!cancelled) setBalance(data?.balance ?? null);
      });
    api
      .GET("/api/entries", {
        params: {
          query: {
            account_id: [accountId],
            sort: "booking_timestamp",
            dir: "desc",
            limit: RECENT_LIMIT,
          },
        },
      })
      .then(({ data }) => {
        if (!cancelled) setRecent(data?.items ?? []);
      });
    return () => {
      cancelled = true;
    };
  }, [accountId]);

  if (account === undefined) {
    return null;
  }
  if (account === null) {
    return (
      <section className="mx-auto flex w-full max-w-3xl flex-col gap-4 px-6 py-12 sm:px-10">
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("accounts.notFound")}
        </p>
      </section>
    );
  }

  return (
    <section className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-6 py-12 sm:px-10">
      <header className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-semibold tracking-tight">
            {account.title}
          </h1>
          {account.description && (
            <p className="text-sm text-zinc-500 dark:text-zinc-400">
              {account.description}
            </p>
          )}
        </div>
        <Link
          to="/accounts/$accountId/edit"
          params={{ accountId }}
          className="rounded-md border border-black/15 px-3 py-2 text-sm font-medium transition-colors hover:bg-black/[.04] dark:border-white/15 dark:hover:bg-white/[.06]"
        >
          {t("accounts.details.edit")}
        </Link>
      </header>

      <dl className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-3">
        <div>
          <dt className="text-zinc-500 dark:text-zinc-400">
            {t("accounts.details.balance")}
          </dt>
          <dd className="font-mono text-lg tabular-nums">
            {balance === null
              ? "…"
              : formatAmount(
                  balance,
                  account.currency,
                  displayedDecimalPlaces,
                  i18n.resolvedLanguage ?? "en",
                )}
          </dd>
        </div>
        <div>
          <dt className="text-zinc-500 dark:text-zinc-400">
            {t("accounts.form.currency")}
          </dt>
          <dd>{account.currency}</dd>
        </div>
        <div>
          <dt className="text-zinc-500 dark:text-zinc-400">
            {t("accounts.form.openingDate")}
          </dt>
          <dd>{account.opening_date}</dd>
        </div>
        {account.closing_date && (
          <div>
            <dt className="text-zinc-500 dark:text-zinc-400">
              {t("accounts.form.closingDate")}
            </dt>
            <dd>{account.closing_date}</dd>
          </div>
        )}
        {account.financial_institute && (
          <div>
            <dt className="text-zinc-500 dark:text-zinc-400">
              {t("accounts.form.financialInstitute")}
            </dt>
            <dd>{account.financial_institute}</dd>
          </div>
        )}
        {account.disabled && (
          <div>
            <dt className="text-zinc-500 dark:text-zinc-400">
              {t("accounts.details.status")}
            </dt>
            <dd>{t("accounts.status.disabled")}</dd>
          </div>
        )}
      </dl>

      <section className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-zinc-500 dark:text-zinc-400">
            {t("accounts.details.recentEntries")}
          </h2>
          <Link
            to="/entries"
            search={{ account_id: accountId }}
            className="text-sm font-medium text-zinc-600 underline underline-offset-2 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
          >
            {t("accounts.details.seeAll")}
          </Link>
        </div>

        {recent === null ? null : recent.length === 0 ? (
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            {t("accounts.details.noEntries")}
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {recent.map((entry) => (
              <li key={entry.id}>
                <Link
                  to="/entries/$entryId/edit"
                  params={{ entryId: entry.id }}
                  className="flex items-center justify-between gap-4 rounded-lg border border-black/10 px-4 py-3 transition-colors hover:bg-black/[.02] dark:border-white/10 dark:hover:bg-white/[.04]"
                >
                  <div className="flex flex-col gap-0.5">
                    <span className="font-medium">{entry.title}</span>
                    <span className="text-xs text-zinc-500 dark:text-zinc-400">
                      {new Date(entry.booking_timestamp).toLocaleDateString(
                        i18n.resolvedLanguage,
                      )}
                    </span>
                  </div>
                  <span className="font-mono text-sm tabular-nums">
                    {formatAmount(
                      entry.amount,
                      account.currency,
                      displayedDecimalPlaces,
                      i18n.resolvedLanguage ?? "en",
                    )}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>
    </section>
  );
}
