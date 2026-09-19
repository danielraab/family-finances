import { Link } from "@tanstack/react-router";
import { Plus } from "lucide-react";
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
 *
 * A row is two lines — title against amount, then date/account against the
 * create action — rather than one. One line forced the shrinkable
 * title/meta column to absorb the whole width deficit left by a `shrink-0`
 * amount and full-text button, which in a dashboard card starved it to
 * 43px and left the meta line painting over the amount. See
 * upcoming-block-row-design's design.md.
 */
export function UpcomingBlock({
  items,
  accounts,
  displayedDecimalPlaces,
  locale,
  onOpenRecurring,
  embedded,
}: {
  items: RecurringTransactionPreviewItem[] | null;
  accounts: Account[];
  displayedDecimalPlaces: number;
  locale: string;
  /** Opens a row's recurring transaction in the read-only summary modal,
   * for that row's own projected date — the summary's Create transaction
   * action books the occurrence the reader clicked, not the template's
   * next suggested one. Required, not optional: an Upcoming row whose
   * title does nothing is the inconsistency this prop exists to remove. */
  onOpenRecurring: (
    recurringTransactionId: string,
    bookingTimestamp: string,
  ) => void;
  /** Set when the block renders inside a card that already draws a
   * bordered, padded box (the dashboard's entry_list card), so it doesn't
   * draw a second one inside it. */
  embedded?: boolean | undefined;
}) {
  const { t } = useTranslation();
  const createLabel = t("recurring.createTransaction");

  return (
    <div
      className={`flex flex-col gap-2 ${
        embedded
          ? "border-b border-black/10 pb-3 dark:border-white/10"
          : "rounded-lg border border-black/10 p-4 dark:border-white/10"
      }`}
    >
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
                className={`flex flex-col gap-0.5 rounded px-1 py-1.5 text-sm ${
                  item.overdue ? "bg-amber-50 dark:bg-amber-500/10" : ""
                }`}
              >
                <div className="flex items-baseline justify-between gap-2">
                  <div className="flex min-w-0 items-baseline gap-1.5">
                    <button
                      type="button"
                      onClick={() =>
                        onOpenRecurring(
                          item.recurring_transaction_id,
                          item.booking_timestamp,
                        )
                      }
                      className="truncate text-left font-medium underline-offset-2 hover:underline"
                    >
                      {item.title}
                    </button>
                    {/* A sibling of the title, never a child of it: inside
                        the title's `truncate` box it was ellipsised along
                        with the title, so the longer the title the less of
                        the marker survived. */}
                    {item.overdue && (
                      <span className="shrink-0 rounded-full bg-amber-100 px-1.5 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
                        {t("recurringPreview.overdue")}
                      </span>
                    )}
                  </div>
                  <span
                    className={`shrink-0 whitespace-nowrap font-mono text-sm tabular-nums ${amountColorClass(item.amount)}`}
                  >
                    {formatAmount(
                      item.amount,
                      item.account_currency ?? "",
                      displayedDecimalPlaces,
                      locale,
                    )}
                  </span>
                </div>

                <div className="flex items-center justify-between gap-2">
                  {/* min-w-0 + overflow-hidden so the account name, whose
                      own label carries a 128px floor, clips inside this
                      line instead of painting over the action beside it. */}
                  <span className="flex min-w-0 items-center gap-1 overflow-hidden text-xs text-zinc-500 dark:text-zinc-400">
                    <span className="shrink-0">
                      {new Date(item.booking_timestamp).toLocaleDateString(
                        locale,
                      )}
                    </span>
                    {account && (
                      <>
                        <span className="shrink-0" aria-hidden="true">
                          ·
                        </span>
                        <AccountLabel account={account} iconSize={14} />
                      </>
                    )}
                  </span>
                  <Link
                    to="/entries/new"
                    search={{
                      recurring_transaction_id: item.recurring_transaction_id,
                      booking_timestamp: item.booking_timestamp,
                    }}
                    aria-label={createLabel}
                    title={createLabel}
                    className="inline-flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md border border-black/10 px-2 py-1 text-xs font-medium text-zinc-500 transition-colors hover:bg-black/[.06] hover:text-zinc-900 dark:border-white/10 dark:text-zinc-400 dark:hover:bg-white/[.08] dark:hover:text-zinc-100"
                  >
                    <Plus size={14} aria-hidden="true" />
                    <span className="hidden sm:inline">{createLabel}</span>
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
