import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { amountColorClass, formatAmount } from "../lib/amount";
import type { Account } from "../lib/useAccountsWithBalances";

function PlusGlyph() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      width={16}
      height={16}
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

/**
 * A single account's card on `/home`: title, financial institute (omitted
 * when unset), and live balance (sign-colored, same rule as `/accounts`),
 * plus a button to log a new entry against this account.
 */
export function AccountCard({
  account,
  balance,
  displayedDecimalPlaces,
  locale,
}: {
  account: Account;
  balance: number | undefined;
  displayedDecimalPlaces: number;
  locale: string;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-black/10 p-4 dark:border-white/10">
      <Link
        to="/accounts/$accountId"
        params={{ accountId: account.id }}
        className="flex flex-1 flex-col gap-1 transition-opacity hover:opacity-80"
      >
        <span className="font-medium">{account.title}</span>
        {account.financial_institute && (
          <span className="text-xs text-zinc-500 dark:text-zinc-400">
            {account.financial_institute}
          </span>
        )}
        <span
          className={`mt-2 font-mono text-lg tabular-nums ${
            balance !== undefined ? amountColorClass(balance) : ""
          }`}
        >
          {balance !== undefined
            ? formatAmount(
                balance,
                account.currency,
                displayedDecimalPlaces,
                locale,
              )
            : "…"}
        </span>
      </Link>
      <Link
        to="/entries/new"
        search={{ account_id: account.id }}
        className="flex items-center gap-1.5 self-start rounded-md px-2 py-1 text-sm text-zinc-500 transition-colors hover:bg-black/[.06] hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-white/[.08] dark:hover:text-zinc-100"
      >
        <PlusGlyph />
        {t("entries.create")}
      </Link>
    </div>
  );
}
