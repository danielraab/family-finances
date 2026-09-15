import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { components } from "../api/schema";
import { amountColorClass, formatAmount } from "../lib/amount";
import {
  previewItemKey,
  type RecurringTransactionPreviewItem,
} from "../lib/recurringPreview";
import { AccountLabel } from "./AccountLabel";

type Account = components["schemas"]["Account"];

/**
 * The shared "Upcoming" preview block: a dedicated, non-paginated list of
 * projected future recurring-transaction occurrences, always sorted by
 * booking_timestamp ascending regardless of whatever sort a real list
 * nearby is using — see web-client-entries'/web-client-reports' "Upcoming
 * block" requirements and design.md's "Upcoming block, not interleaved
 * pagination" decision. Presentational only: the caller fetches
 * (`fetchRecurringPreview`) and passes the result in, `null` while loading.
 * An overdue item (item.overdue) renders with a distinct tint but stays in
 * its normal chronological position — never reordered.
 */
export function UpcomingBlock({
  items,
  accounts,
  displayedDecimalPlaces,
  locale,
}: {
  items: RecurringTransactionPreviewItem[] | null;
  accounts: Account[];
  displayedDecimalPlaces: number;
  locale: string;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-black/10 p-4 dark:border-white/10">
      <h3 className="text-xs font-semibold uppercase text-zinc-500 dark:text-zinc-400">
        {t("recurringPreview.heading")}
      </h3>

      {items === null ? (
        <span className="text-sm text-zinc-500 dark:text-zinc-400">…</span>
      ) : items.length === 0 ? (
        <span className="text-sm text-zinc-500 dark:text-zinc-400">
          {t("recurringPreview.empty")}
        </span>
      ) : (
        <ul className="flex flex-col divide-y divide-black/5 dark:divide-white/5">
          {items.map((item) => {
            const account = accounts.find((a) => a.id === item.account_id);
            return (
              <li
                key={previewItemKey(item)}
                className={`flex items-center justify-between gap-2 rounded px-1 py-1.5 text-sm ${
                  item.overdue ? "bg-amber-50 dark:bg-amber-500/10" : ""
                }`}
              >
                <div className="flex min-w-0 flex-col">
                  <span className="truncate font-medium">
                    {item.title}
                    {item.overdue && (
                      <span className="ml-1.5 text-xs font-normal text-amber-600 dark:text-amber-400">
                        {t("recurringPreview.overdue")}
                      </span>
                    )}
                  </span>
                  <span className="flex items-center gap-1 text-xs text-zinc-500 dark:text-zinc-400">
                    {new Date(item.booking_timestamp).toLocaleDateString(
                      locale,
                    )}
                    {account && (
                      <>
                        {" · "}
                        <AccountLabel account={account} iconSize={14} />
                      </>
                    )}
                  </span>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <span
                    className={`font-mono text-sm tabular-nums ${amountColorClass(item.amount)}`}
                  >
                    {formatAmount(
                      item.amount,
                      item.account_currency ?? "",
                      displayedDecimalPlaces,
                      locale,
                    )}
                  </span>
                  <Link
                    to="/entries/new"
                    search={{
                      recurring_transaction_id: item.recurring_transaction_id,
                      booking_timestamp: item.booking_timestamp,
                    }}
                    className="whitespace-nowrap rounded-md border border-black/10 px-2 py-1 text-xs font-medium text-zinc-500 transition-colors hover:bg-black/[.06] hover:text-zinc-900 dark:border-white/10 dark:text-zinc-400 dark:hover:bg-white/[.08] dark:hover:text-zinc-100"
                  >
                    {t("recurring.createTransaction")}
                  </Link>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
